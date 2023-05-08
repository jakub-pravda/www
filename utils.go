package main

import (
	"errors"
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
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

func filesToBucketObjects(ctx *pulumi.Context, accessBlock *s3.BucketPublicAccessBlock, bucket *s3.Bucket, dirPath string) ([]*s3.BucketObject, error) {
	log.Printf("Processing directory content to the buckets %s\n", dirPath)
	files, err := os.ReadDir(dirPath)
	handleErr(err)
	buckets := make([]*s3.BucketObject, 0)
	for _, file := range files {
		filePath := dirPath + "/" + file.Name()
		if file.Type().IsDir() {
			recBuckets, err := filesToBucketObjects(ctx, accessBlock, bucket, filePath)
			handleErr(err)
			buckets = append(buckets, recBuckets...)
		} else if file.Type().IsRegular() {
			bucketObject, err := bucketObjectConverter(ctx, accessBlock, bucket, filePath)
			handleErr(err)
			buckets = append(buckets, bucketObject)
		}
	}
	return buckets, nil
}

func bucketObjectConverter(ctx *pulumi.Context, accessBlock *s3.BucketPublicAccessBlock, bucket *s3.Bucket, path string) (*s3.BucketObject, error) {
	mimeType := mime.TypeByExtension(filepath.Ext(path))
	// remove wwwDir from the path (as we want to save files directly to the bucket root)
	dstFilePath := removeRootDir(path)
	log.Printf("Converting file %s to bucket object with mime type %s\n", path, mimeType)
	return s3.NewBucketObject(ctx, path, &s3.BucketObjectArgs{
		Key:         pulumi.String(dstFilePath),
		Bucket:      bucket.ID(),
		Acl:         pulumi.String("public-read"),
		Source:      pulumi.NewFileAsset(path),
		ContentType: pulumi.String(mimeType),
	}, pulumi.DependsOn([]pulumi.Resource{accessBlock}))
}

func getRoute53HostedZone(ctx *pulumi.Context, targetDomain string) (string, error) {
	// returns new or existing hosted zone ID
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

func removeRootDir(path string) string {
	index := strings.Index(path, "/")
	if index != -1 {
		return path[index+1:]
	} else {
		return path
	}
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

func getDefaultSubdomains(domain string) ([]string, error) {
	// Get default subdomains for the given domain
	// domain must be in format 'example.com'
	if len(strings.Split(domain, ".")) > 2 {
		log.Fatalf("Invalid domain format: %s, must be in format 'example.com'\n", domain)
		return []string{}, errors.New("Invalid domain format")
	} else {
		return []string{
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
	records := make([]*route53.Record, len(domains))

	for i, domain := range domains {
		certValidationDomain, err := route53.NewRecord(ctx, fmt.Sprintf("%s-validation", domain), &route53.RecordArgs{
			Name: certificate.DomainValidationOptions.ApplyT(func(options []acm.CertificateDomainValidationOption) string {
				resourceRecordName := options[i].ResourceRecordName
				log.Printf("Domain: %s, DNS resource record name: %v", domain, resourceRecordName)
				return *resourceRecordName
			}).(pulumi.StringOutput),
			Type: certificate.DomainValidationOptions.ApplyT(func(options []acm.CertificateDomainValidationOption) string {
				resourceRecordType := options[i].ResourceRecordType
				log.Printf("Domain %s, DNS resource record type: %v", domain, resourceRecordType)
				return *resourceRecordType
			}).(pulumi.StringOutput),
			Records: pulumi.StringArray{
				certificate.DomainValidationOptions.ApplyT(func(options []acm.CertificateDomainValidationOption) string {
					recordValue := options[i].ResourceRecordValue
					log.Printf("Domain %s, DNS record value: %v", domain, recordValue)
					return *recordValue
				}).(pulumi.StringOutput)},
			ZoneId: pulumi.String(hostedZoneId),
			Ttl:    pulumi.Int(60),
		})
		handleErr(err)
		records[i] = certValidationDomain
	}
	return records
}

func mapValidationRecordsFqdn(validationRecords []*route53.Record) pulumi.StringArray {
	fqdnArray := make(pulumi.StringArray, len(validationRecords))
	for i, record := range validationRecords {
		fqdnArray[i] = record.Fqdn
	}
	return fqdnArray
}
