package main

import (
	"fmt"
	"log"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/acm"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/cloudfront"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/lambda"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

var (
	projects = []string{"sramek-garden-center", "sramek-transportation"}
)

type staticSiteProject struct {
	name       string
	dir        string
	bucketPath string
	domain     string
	indexDoc   string
	errorDoc   string
	cors       string
}

func getProjectConfig(ctx *pulumi.Context, projectName string) staticSiteProject {
	projectConfig := config.New(ctx, projectName)
	return staticSiteProject{
		name:       projectName,
		dir:        projectConfig.Require("dir"),
		domain:     projectConfig.Get("domain"),
		bucketPath: projectConfig.Get("bucket-path"),
		indexDoc:   projectConfig.Require("index-doc"),
		errorDoc:   projectConfig.Require("error-doc"),
		cors:       projectConfig.Get("cors"), // TODO needed?
	}
}

func main() {
	// Static website deployment using AWS:
	// -- s3 bucket
	// -- cloudfront
	// -- route53
	// -- amazon certificate manager

	log.Println("Deploying static website infrastructure")

	pulumi.Run(func(ctx *pulumi.Context) error {
		logsBucket := createBucket(ctx, "request-logs-sramek-infra")

		log.Println("Deploying global lambda functions")
		redirectLambda := lambdaRedirect(ctx)

		log.Println("Deploying websites")
		for _, projectName := range projects {
			projectConfig := getProjectConfig(ctx, projectName)
			deployProject(ctx, projectConfig, logsBucket, redirectLambda)
		}

		// TODO: Read domain from config (after migration domain to AWS)
		simpleMailService(ctx, "sramek-autodoprava.cz")
		return nil
	})
}

func deployProject(ctx *pulumi.Context, project staticSiteProject, logsBucket *s3.Bucket, redirectLambda *lambda.Function) {
	log.Printf("Deploy WWW id: %s, dir: %s, domain: %s", project.name, project.dir, project.domain)

	domains, err := getDomainWithSubdomains(project.domain)
	log.Println("Used domains: ", domains)
	handleErr(err)

	contentBucket := createContentBucket(ctx, project, false)

	if len(domains) > 0 {
		cdn := instantiateCloudfront(ctx, contentBucket, logsBucket, domains, project.indexDoc, project.name, redirectLambda)
		createAliasRecords(ctx, cdn, domains)
		ctx.Export(fmt.Sprintf("%s-cloudfrontDomain", project.name), cdn.DomainName)
	} else {
		log.Println("No domains provided, skipping Cloudfront distribution")
	}

	ctx.Export(fmt.Sprintf("%s-bucketName", project.name), contentBucket.ID())
	ctx.Export(fmt.Sprintf("%s-bucketEndpoint", project.name), contentBucket.WebsiteEndpoint.ApplyT(func(websiteEndpoint string) (string, error) {
		return fmt.Sprintf("http://%v", websiteEndpoint), nil
	}).(pulumi.StringOutput))
}

func getArnCertificate(ctx *pulumi.Context, domains []string) pulumi.StringOutput {
	mainDomain := domains[0]

	eastRegion, err := aws.NewProvider(ctx, fmt.Sprintf("%s-east", mainDomain), &aws.ProviderArgs{
		Region: pulumi.String("us-east-1"), // AWS Certificate Manager is available only in us east region
	})

	// generate certificate for our domain
	certificate, err := acm.NewCertificate(ctx, fmt.Sprintf("%s-certificate", mainDomain), &acm.CertificateArgs{
		DomainName:              pulumi.String(mainDomain),
		ValidationMethod:        pulumi.String("DNS"),
		SubjectAlternativeNames: stringArrayToPulumiStringArray(domains),
	}, pulumi.Provider(eastRegion))
	handleErr(err)

	zoneId, err := getRoute53HostedZone(ctx, mainDomain)
	handleErr(err)
	log.Printf("DNS Hosted zone: %s", zoneId)

	validationRecords := createValidationRecords(ctx, domains, certificate, zoneId)

	certValidation, err := acm.NewCertificateValidation(ctx, fmt.Sprintf("%s-certificate-validation", mainDomain), &acm.CertificateValidationArgs{
		CertificateArn:        certificate.Arn,
		ValidationRecordFqdns: mapValidationRecordsFqdn(validationRecords),
	}, pulumi.Provider(eastRegion))
	handleErr(err)

	return certValidation.CertificateArn
}

func instantiateCloudfront(
	ctx *pulumi.Context,
	contentBucket *s3.Bucket,
	logsBucket *s3.Bucket,
	domains []string,
	indexDoc string,
	projectName string,
	redirectLambda *lambda.Function) *cloudfront.Distribution {
	mainDomain := domains[0]
	log.Printf("Creating Cloudfront distribution for project: %s\n", projectName)

	viewerLambdaAssociation := cloudfront.DistributionDefaultCacheBehaviorLambdaFunctionAssociationArgs{
		// Redirect lambda handles redirecting from non www domain to www domain
		EventType:   pulumi.String("viewer-request"),
		LambdaArn:   redirectLambda.QualifiedArn,
		IncludeBody: pulumi.Bool(false),
	}

	distribution, err := cloudfront.NewDistribution(ctx, fmt.Sprintf("%s-cdn", mainDomain), &cloudfront.DistributionArgs{
		Enabled:           pulumi.Bool(true),
		Aliases:           stringArrayToPulumiStringArray(domains),
		DefaultRootObject: pulumi.String(indexDoc),
		DefaultCacheBehavior: cloudfront.DistributionDefaultCacheBehaviorArgs{
			TargetOriginId: contentBucket.Arn,
			LambdaFunctionAssociations: cloudfront.DistributionDefaultCacheBehaviorLambdaFunctionAssociationArray{
				viewerLambdaAssociation,
			},
			ViewerProtocolPolicy: pulumi.String("redirect-to-https"),
			AllowedMethods:       pulumi.StringArray{pulumi.String("GET"), pulumi.String("HEAD")},
			CachedMethods:        pulumi.StringArray{pulumi.String("GET"), pulumi.String("HEAD")},
			ForwardedValues: cloudfront.DistributionDefaultCacheBehaviorForwardedValuesArgs{
				Cookies: cloudfront.DistributionDefaultCacheBehaviorForwardedValuesCookiesArgs{
					Forward: pulumi.String("none"),
				},
				QueryString: pulumi.Bool(false),
			},
			MinTtl:     pulumi.Int(0),
			MaxTtl:     pulumi.Int(60 * 10), // 10 minutes
			DefaultTtl: pulumi.Int(60 * 10), // 10 minutes
			Compress:   pulumi.Bool(true),
		},
		Origins: cloudfront.DistributionOriginArray{
			cloudfront.DistributionOriginArgs{
				OriginId:   contentBucket.Arn,
				DomainName: contentBucket.WebsiteEndpoint,
				CustomOriginConfig: cloudfront.DistributionOriginCustomOriginConfigArgs{
					OriginProtocolPolicy: pulumi.String("http-only"),
					HttpPort:             pulumi.Int(80),
					HttpsPort:            pulumi.Int(443),
					OriginSslProtocols:   pulumi.StringArray{pulumi.String("TLSv1.2")},
				},
			},
		},
		PriceClass: pulumi.String("PriceClass_100"),

		// Put access logs to the bucket we created before
		LoggingConfig: cloudfront.DistributionLoggingConfigArgs{
			Bucket:         logsBucket.BucketDomainName,
			IncludeCookies: pulumi.Bool(false),
			Prefix:         pulumi.String(fmt.Sprintf("%s/", projectName)),
		},

		// Set restrictions for our websites, at this moment we don't need any
		Restrictions: cloudfront.DistributionRestrictionsArgs{
			GeoRestriction: cloudfront.DistributionRestrictionsGeoRestrictionArgs{
				RestrictionType: pulumi.String("none"),
			},
		},
		// Use the distribution certificate
		ViewerCertificate: cloudfront.DistributionViewerCertificateArgs{
			AcmCertificateArn: getArnCertificate(ctx, domains),
			SslSupportMethod:  pulumi.String("sni-only"),
		},

		// It takes around 15min to create cloudfront distribution, so we don't want to wait for it
		WaitForDeployment: pulumi.Bool(false),
	})
	handleErr(err)
	return distribution
}
