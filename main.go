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
	wwwGardenCenter = "garden-center"
)

func main() {
	// What I'm using:
	// -- s3 bucket
	// -- cloudfront
	// -- route53
	// -- amazon certificate manager

	log.Println("Deploying infrastructure for our static websites")

	pulumi.Run(func(ctx *pulumi.Context) error {
		// TODO do it more versatile
		gardenCenterConfig := config.New(ctx, wwwGardenCenter)
		wwwDir := gardenCenterConfig.Require("dir")
		targetDomain := gardenCenterConfig.Require("domain")
		log.Printf("Deploy WWW id: %s, dir: %s, domain: %s", wwwGardenCenter, wwwDir, targetDomain)

		// crate domain, subdomains array
		subdomains, err := getDefaultSubdomains(targetDomain)
		handleErr(err)
		allDomains := append([]string{targetDomain}, subdomains...)

		contentBucket := createS3Bucket(ctx, wwwGardenCenter, wwwDir)
		cdn := instantiateCloudfront(ctx, contentBucket, targetDomain, subdomains)
		createAliasRecords(ctx, cdn, allDomains)

		// Export the pulumi outputs
		ctx.Export("bucketName", contentBucket.ID())
		ctx.Export("cloudfrontDomain", cdn.DomainName)
		ctx.Export("bucketEndpoint", contentBucket.WebsiteEndpoint.ApplyT(func(websiteEndpoint string) (string, error) {
			return fmt.Sprintf("http://%v", websiteEndpoint), nil
		}).(pulumi.StringOutput))
		return nil
	})
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

func getArnCertificate(ctx *pulumi.Context, targetDomain string, subdomains []string) pulumi.StringOutput {
	eastRegion, err := aws.NewProvider(ctx, "east", &aws.ProviderArgs{
		Region: pulumi.String("us-east-1"), // AWS Certificate Manager is not available in other regions
	})

	// generate certificate for our domain
	certificate, err := acm.NewCertificate(ctx, "certificate", &acm.CertificateArgs{
		DomainName:              pulumi.String(targetDomain),
		ValidationMethod:        pulumi.String("DNS"),
		SubjectAlternativeNames: stringArrayToPulumiStringArray(subdomains),
	}, pulumi.Provider(eastRegion))
	handleErr(err)

	zoneId, err := getRoute53HostedZone(ctx, targetDomain)
	handleErr(err)
	log.Printf("DNS Hosted zone: %s", zoneId)

	validationRecords := createValidationRecords(ctx, append(subdomains, targetDomain), certificate, zoneId)

	certValidation, err := acm.NewCertificateValidation(ctx, "certificate-validation", &acm.CertificateValidationArgs{
		CertificateArn:        certificate.Arn,
		ValidationRecordFqdns: mapValidationRecordsFqdn(validationRecords),
	}, pulumi.Provider(eastRegion))
	handleErr(err)

	return certValidation.CertificateArn
}

func instantiateCloudfront(ctx *pulumi.Context, contentBucket *s3.Bucket, targetDomain string, subdomains []string) *cloudfront.Distribution {
	log.Printf("Creating Cloudfront distribution for domain: %s\n", targetDomain)

	allDomains := append(subdomains, targetDomain)
	logsBucket, err := s3.NewBucket(ctx, "requests-logs", &s3.BucketArgs{
		Acl:    pulumi.String("private"),
		Bucket: pulumi.String(fmt.Sprintf("%s-logs", targetDomain)),
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
		Aliases:           stringArrayToPulumiStringArray(allDomains),
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
			Prefix:         pulumi.String(fmt.Sprintf("%s/", targetDomain)),
		},

		// Set restrictions for our websites, at this moment we don't need any
		Restrictions: cloudfront.DistributionRestrictionsArgs{
			GeoRestriction: cloudfront.DistributionRestrictionsGeoRestrictionArgs{
				RestrictionType: pulumi.String("none"),
			},
		},
		// It takes around 15min to create cloudfront distribution, so we don't want to wait for it

		// Use the distribution certificate
		ViewerCertificate: cloudfront.DistributionViewerCertificateArgs{
			AcmCertificateArn: getArnCertificate(ctx, targetDomain, subdomains),
			SslSupportMethod:  pulumi.String("sni-only"),
		},

		WaitForDeployment: pulumi.Bool(false),
	})
	handleErr(err)
	return distribution
}
