package main

import (
	"errors"
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/acm"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/cloudfront"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/route53"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}

func filesToBucketObjects(ctx *pulumi.Context, accessBlock *s3.BucketPublicAccessBlock, bucket *s3.Bucket, localPath string, bucketPath string) ([]*s3.BucketObject, error) {
	log.Printf("Processing directory content to the buckets %s\n", localPath)
	files, err := os.ReadDir(localPath)
	handleErr(err)
	buckets := make([]*s3.BucketObject, 0)
	for _, file := range files {
		nextDirPath := filepath.Join(localPath, file.Name())
		nextBucketPath := filepath.Join(bucketPath, file.Name())
		if file.Type().IsDir() {
			recBuckets, err := filesToBucketObjects(ctx, accessBlock, bucket, nextDirPath, nextBucketPath)
			handleErr(err)
			buckets = append(buckets, recBuckets...)
		} else if file.Type().IsRegular() {
			bucketObject, err := bucketObjectConverter(ctx, accessBlock, bucket, nextDirPath, nextBucketPath)
			handleErr(err)
			buckets = append(buckets, bucketObject)
		}
	}
	return buckets, nil
}

func bucketObjectConverter(ctx *pulumi.Context, accessBlock *s3.BucketPublicAccessBlock, bucket *s3.Bucket, localPath string, bucketPath string) (*s3.BucketObject, error) {
	re := regexp.MustCompile("www/[^/]+")
	mimeType := mime.TypeByExtension(filepath.Ext(localPath))
	// Remove wwwDir from the path (as we want to save files directly to the bucket root)
	dstFilePath := re.ReplaceAllString(bucketPath, "")
	log.Printf("Converting file %s to bucket object with mime type %s\n", bucketPath, mimeType)
	return s3.NewBucketObject(ctx, bucketPath, &s3.BucketObjectArgs{
		Key:         pulumi.String(dstFilePath),
		Bucket:      bucket.ID(),
		Acl:         pulumi.String("public-read"),
		Source:      pulumi.NewFileAsset(localPath),
		ContentType: pulumi.String(mimeType),
	}, pulumi.DependsOn([]pulumi.Resource{accessBlock}))
}

func getRoute53HostedZone(ctx *pulumi.Context, targetDomain string) (string, error) {
	// Returns new or existing hosted zone ID
	log.Printf("Looking up hosted zone for domain %s\n", targetDomain)
	zoneLookupFunc := func(domain string) (*route53.LookupZoneResult, error) {
		return route53.LookupZone(ctx, &route53.LookupZoneArgs{
			Name: &domain,
		})
	}
	lookupResult, err := zoneLookupFunc(targetDomain)
	handleErr(err)
	return lookupResult.Id, nil
}

func getDomainAndSubdomain(domain string) (string, string) {
	// returns domain and subdomain
	// example: www.example.com -> example.com, www
	// example: example.com -> example.com, example.com
	split := strings.Split(domain, ".")
	if len(split) > 2 {
		return strings.Join(split[1:], "."), split[0]
	} else {
		return domain, domain
	}
}

func getDomainWithSubdomains(domain string) ([]string, error) {
	// Get default subdomains for the given domain
	// domain must be in the format 'example.com'
	if len(domain) == 0 {
		return []string{}, nil
	} else if len(strings.Split(domain, ".")) > 2 {
		log.Fatalf("Invalid domain format: %s, must be in format 'example.com'\n", domain)
		return []string{}, errors.New("Invalid domain format")
	} else {
		return []string{
			domain,
			"www." + domain,
		}, nil
	}
}

func stringArrayToPulumiStringArray(arr []string) pulumi.StringArray {
	// Converts string array to pulumi string array
	pulumiArr := make(pulumi.StringArray, len(arr))
	for i, v := range arr {
		pulumiArr[i] = pulumi.String(v)
	}
	return pulumiArr
}

func createAliasRecord(ctx *pulumi.Context, distribution *cloudfront.Distribution, domain string) {
	// Creates alias records for the given domain and distribution
	parentDomain, subDomain := getDomainAndSubdomain(domain)
	log.Printf("Creating alias for domain %s\n", domain)
	hzid, err := getRoute53HostedZone(ctx, parentDomain)
	handleErr(err)
	record, err := route53.NewRecord(ctx, domain, &route53.RecordArgs{
		Name:   pulumi.String(subDomain),
		ZoneId: pulumi.String(hzid),
		Type:   pulumi.String("A"),
		Aliases: route53.RecordAliasArray{
			&route53.RecordAliasArgs{
				Name:                 distribution.DomainName,
				ZoneId:               distribution.HostedZoneId,
				EvaluateTargetHealth: pulumi.Bool(true),
			},
		},
	})
	handleErr(err)
	log.Printf("Created alias record %v for domain %s", record.Name, domain)
}

func createAliasRecords(ctx *pulumi.Context, distribution *cloudfront.Distribution, domains []string) {
	// Creates alias records for the given domains and distribution
	for _, domain := range domains {
		createAliasRecord(ctx, distribution, domain)
	}
}

func createValidationRecords(ctx *pulumi.Context, domains []string, certificate *acm.Certificate, hostedZoneId string) []*route53.Record {
	log.Println("Creating validation records for domains ", domains)

	records := make([]*route53.Record, len(domains))

	for i, domain := range domains {
		currentIndex := i
		certValidationDomain, err := route53.NewRecord(ctx, fmt.Sprintf("validation-record-%s", domain), &route53.RecordArgs{
			Name: certificate.DomainValidationOptions.ApplyT(func(options []acm.CertificateDomainValidationOption) string {
				resourceRecordName := options[currentIndex].ResourceRecordName
				log.Printf("Domain %s, DNS resource record name: %v", domain, resourceRecordName)
				return *resourceRecordName
			}).(pulumi.StringOutput),
			Type: certificate.DomainValidationOptions.ApplyT(func(options []acm.CertificateDomainValidationOption) string {
				resourceRecordType := options[currentIndex].ResourceRecordType
				log.Printf("Domain %s, DNS resource record type: %v", domain, resourceRecordType)
				return *resourceRecordType
			}).(pulumi.StringOutput),
			Records: pulumi.StringArray{
				certificate.DomainValidationOptions.ApplyT(func(options []acm.CertificateDomainValidationOption) string {
					recordValue := options[currentIndex].ResourceRecordValue
					log.Printf("Domain %s, DNS record value: %v", domain, recordValue)
					return *recordValue
				}).(pulumi.StringOutput)},
			ZoneId: pulumi.String(hostedZoneId),
			Ttl:    pulumi.Int(60),
		})
		handleErr(err)
		records[currentIndex] = certValidationDomain
	}
	return records
}

func mapValidationRecordsFqdn(validationRecords []*route53.Record) pulumi.StringArray {
	// Maps validation records to fqdn array
	fqdnArray := make(pulumi.StringArray, len(validationRecords))
	for i, record := range validationRecords {
		fqdnArray[i] = record.Fqdn
	}
	return fqdnArray
}

func createBucket(ctx *pulumi.Context, bucketName string) *s3.Bucket {
	bucket, err := s3.NewBucket(ctx, fmt.Sprintf("%s-s3-bucket", bucketName), &s3.BucketArgs{
		Acl:    pulumi.String("private"),
		Bucket: pulumi.String(bucketName),
	})
	handleErr(err)

	_, er := s3.NewBucketOwnershipControls(ctx, fmt.Sprintf("%s-ownership-controls", bucketName), &s3.BucketOwnershipControlsArgs{
		Bucket: bucket.ID(),
		Rule: &s3.BucketOwnershipControlsRuleArgs{
			ObjectOwnership: pulumi.String("BucketOwnerPreferred"),
		},
	})
	handleErr(er) // Creates a new S3 bucket
	return bucket
}

func createContentBucket(ctx *pulumi.Context, project staticSiteProject, blockPublicAccess bool) *s3.Bucket {
	log.Println("Creating content S3 bucket. Index document: ", project.indexDoc)

	bucketName := fmt.Sprintf("%s-bucket", project.name)
	bucket, err := s3.NewBucket(ctx, bucketName, &s3.BucketArgs{
		Website: s3.BucketWebsiteArgs{
			IndexDocument: pulumi.String(project.indexDoc),
			ErrorDocument: pulumi.String(project.errorDoc),
		},
	})
	handleErr(err)
	_, err = s3.NewBucketOwnershipControls(ctx, fmt.Sprintf("%s-ownership-controls", project.name), &s3.BucketOwnershipControlsArgs{
		Bucket: bucket.ID(),
		Rule: &s3.BucketOwnershipControlsRuleArgs{
			ObjectOwnership: pulumi.String("ObjectWriter"),
		},
	})
	handleErr(err)

	// set public access to our bucket
	publicAccessBlock, err := s3.NewBucketPublicAccessBlock(ctx, fmt.Sprintf("%s-public-access-block", project.name), &s3.BucketPublicAccessBlockArgs{
		Bucket:          bucket.ID(),
		BlockPublicAcls: pulumi.Bool(blockPublicAccess),
	})
	handleErr(err)

	// create S3 buckets with web content
	_, err = filesToBucketObjects(ctx, publicAccessBlock, bucket, project.dir, project.bucketPath)
	handleErr(err)

	// Set the CORS configuration for the bucket
	setBucketCors(ctx, bucket, project.cors, project.name)
	return bucket
}

func setBucketCors(ctx *pulumi.Context, bucket *s3.Bucket, cors string, projectName string) {
	if cors != "" {
		_, err := s3.NewBucketCorsConfigurationV2(ctx, fmt.Sprintf("%s-cors-setting", projectName), &s3.BucketCorsConfigurationV2Args{
			Bucket: bucket.ID(),
			CorsRules: s3.BucketCorsConfigurationV2CorsRuleArray{
				&s3.BucketCorsConfigurationV2CorsRuleArgs{
					AllowedHeaders: pulumi.StringArray{pulumi.String("*")},
					AllowedMethods: pulumi.StringArray{pulumi.String("GET")},
					AllowedOrigins: pulumi.StringArray{pulumi.String(cors)},
					MaxAgeSeconds:  pulumi.IntPtr(3000),
				},
			},
		})
		handleErr(err)
	}
}
