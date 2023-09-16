package main

import (
	"fmt"
	"log"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/acm"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/cloudfront"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

var (
	projects = []string{"garden-center"}
)

type WwwProject struct {
	ProjectName string
	ProjectDir  string
	Domain      string
}

func getProjectConfig(ctx *pulumi.Context, projectName string) WwwProject {
	projectConfig := config.New(ctx, projectName)
	return WwwProject{
		ProjectName: projectName,
		ProjectDir:  projectConfig.Require("dir"),
		Domain:      projectConfig.Require("domain"),
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
		// TODO make it more versatile
		// TODO what to do?
		for _, projectName := range projects {
			projectConfig := getProjectConfig(ctx, projectName)
			deployProject(ctx, projectConfig)
		}
		return nil
	})
}

func deployProject(ctx *pulumi.Context, project WwwProject) {
	log.Printf("Deploy WWW id: %s, dir: %s, domain: %s", project.ProjectName, project.ProjectDir, project.Domain)

	domains, err := getDomainWithSubdomains(project.Domain) // TODO empty domain
	handleErr(err)

	contentBucket := createS3Bucket(ctx, project.ProjectName, project.ProjectDir)

	cdn := instantiateCloudfront(ctx, contentBucket, domains)
	createAliasRecords(ctx, cdn, domains)

	// Export the pulumi outputs
	ctx.Export(fmt.Sprintf("%s-bucketName", project.ProjectName), contentBucket.ID())
	ctx.Export(fmt.Sprintf("%s-cloudfrontDomain", project.ProjectName), cdn.DomainName)
	ctx.Export(fmt.Sprintf("%s-bucketEndpoint", project.ProjectName), contentBucket.WebsiteEndpoint.ApplyT(func(websiteEndpoint string) (string, error) {
		return fmt.Sprintf("http://%v", websiteEndpoint), nil
	}).(pulumi.StringOutput))
}

func createS3Bucket(ctx *pulumi.Context, name string, wwwDir string) *s3.Bucket {
	log.Println("Creating content S3 bucket")

	bucketName := fmt.Sprintf("%s-bucket", name)
	bucket, err := s3.NewBucket(ctx, bucketName, &s3.BucketArgs{
		Website: s3.BucketWebsiteArgs{
			IndexDocument: pulumi.String("main.html"),
		},
	})
	handleErr(err)
	_, err = s3.NewBucketOwnershipControls(ctx, "ownership-controls", &s3.BucketOwnershipControlsArgs{
		Bucket: bucket.ID(),
		Rule: &s3.BucketOwnershipControlsRuleArgs{
			ObjectOwnership: pulumi.String("ObjectWriter"),
		},
	})
	handleErr(err)
	// set public access to our bucket
	publicAccessBlock, err := s3.NewBucketPublicAccessBlock(ctx, "public-access-block", &s3.BucketPublicAccessBlockArgs{
		Bucket:          bucket.ID(),
		BlockPublicAcls: pulumi.Bool(false),
	})
	handleErr(err)
	// create S3 buckets with web content
	_, err = filesToBucketObjects(ctx, publicAccessBlock, bucket, wwwDir)
	handleErr(err)
	return bucket
}

func getArnCertificate(ctx *pulumi.Context, domains []string) pulumi.StringOutput {
	mainDomain := domains[0]
	eastRegion, err := aws.NewProvider(ctx, "east", &aws.ProviderArgs{
		Region: pulumi.String("us-east-1"), // AWS Certificate Manager is available only in us east region
	})

	// generate certificate for our domain
	certificate, err := acm.NewCertificate(ctx, "certificate", &acm.CertificateArgs{
		DomainName:              pulumi.String(mainDomain),
		ValidationMethod:        pulumi.String("DNS"),
		SubjectAlternativeNames: stringArrayToPulumiStringArray(domains),
	}, pulumi.Provider(eastRegion))
	handleErr(err)

	zoneId, err := getRoute53HostedZone(ctx, mainDomain)
	handleErr(err)
	log.Printf("DNS Hosted zone: %s", zoneId)

	validationRecords := createValidationRecords(ctx, domains, certificate, zoneId)

	certValidation, err := acm.NewCertificateValidation(ctx, "certificate-validation", &acm.CertificateValidationArgs{
		CertificateArn:        certificate.Arn,
		ValidationRecordFqdns: mapValidationRecordsFqdn(validationRecords),
	}, pulumi.Provider(eastRegion))
	handleErr(err)

	return certValidation.CertificateArn
}

func instantiateCloudfront(ctx *pulumi.Context, contentBucket *s3.Bucket, domains []string) *cloudfront.Distribution {
	mainDomain := domains[0]
	log.Printf("Creating Cloudfront distribution for domain: %s\n", mainDomain)

	logsBucket, err := s3.NewBucket(ctx, "requests-logs", &s3.BucketArgs{
		Acl:    pulumi.String("private"),
		Bucket: pulumi.String(fmt.Sprintf("%s-logs", mainDomain)),
	})
	handleErr(err)

	_, er := s3.NewBucketOwnershipControls(ctx, "logs-ownership-controls", &s3.BucketOwnershipControlsArgs{
		Bucket: logsBucket.ID(),
		Rule: &s3.BucketOwnershipControlsRuleArgs{
			ObjectOwnership: pulumi.String("BucketOwnerPreferred"),
		},
	})
	handleErr(er)

	distribution, err := cloudfront.NewDistribution(ctx, "cdn", &cloudfront.DistributionArgs{
		Enabled:           pulumi.Bool(true),
		Aliases:           stringArrayToPulumiStringArray(domains),
		DefaultRootObject: pulumi.String("main.html"),
		DefaultCacheBehavior: cloudfront.DistributionDefaultCacheBehaviorArgs{
			TargetOriginId:       contentBucket.Arn,
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
			Prefix:         pulumi.String(fmt.Sprintf("%s/", mainDomain)),
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
