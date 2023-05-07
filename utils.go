package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/cloudfront"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/route53"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"os"
	"log"
	"mime"
	"path/filepath"
	"strings"
)

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}

func filesToBucketObjects(ctx *pulumi.Context, accessBlock *s3.BucketPublicAccessBlock, bucket *s3.Bucket, dirPath string) ([]*s3.BucketObject, error) {
	log.Printf("Processing directory content to the buckets %s\n", dirPath)
	files, err := os.ReadDir(dirPath); handleErr(err)
	buckets := make([]*s3.BucketObject, 0)
	for _, file := range files {
		filePath := dirPath + "/" + file.Name()
		if file.Type().IsDir() {
			recBuckets, err := filesToBucketObjects(ctx, accessBlock, bucket, filePath); handleErr(err)
			buckets = append(buckets, recBuckets...)
		} else if ( file.Type().IsRegular() ) {
			bucketObject, err := bucketObjectConverter(ctx, accessBlock, bucket, filePath); handleErr(err)
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
		Key: 			pulumi.String(dstFilePath),
		Bucket: 		bucket.ID(),
		Acl: 			pulumi.String("public-read"),
		Source: 		pulumi.NewFileAsset(path),
		ContentType: 	pulumi.String(mimeType),
	}, pulumi.DependsOn([]pulumi.Resource{accessBlock}))
}

func getOrCreateRoute53HostedZone(ctx *pulumi.Context, targetDomain string) (string, error) {
	// returns new or existing hosted zone ID
	log.Printf("Looking up hosted zone for domain %s\n", targetDomain)
	zoneLookupFunc := func (domain string) (*route53.LookupZoneResult, error) {
		return route53.LookupZone(ctx, &route53.LookupZoneArgs{
			Name: &domain,
		})
	}
	
	//zoneCreateFunc := func (domain string) (*route53.Zone, error) {
	//	return route53.NewZone(ctx, domain, &route53.ZoneArgs{
	//		Name: pulumi.String(domain),
	//		Comment: pulumi.String("Hosted zone for " + domain),
	//	})
	//}
	
	lookupResult, err := zoneLookupFunc(targetDomain); handleErr(err)
	return lookupResult.Id, nil

	//if zoneLookupFuncResult, err := zoneLookupFunc(targetDomain); err != nil {
	//	if strings.Contains(err.Error(), "no matching Route53Zone found") {
	//		log.Print("Hosted zone not found, creating new one")
	//		if zoneCreateFuncResult, err := zoneCreateFunc(targetDomain); err != nil {
	//			return "", err
	//		} else {
	//			return zoneCreateFuncResult.ZoneId, nil
	//		}
	//	} else {
	//		return "", err
	//	}
	//} else {
	//	return zoneLookupFuncResult.Id, nil
	//}
}

func createAliasRecord(ctx *pulumi.Context, domain string, distribution *cloudfront.Distribution) {
	hzid, err := getOrCreateRoute53HostedZone(ctx, domain); handleErr(err)
	record, err := route53.NewRecord(ctx, "www", &route53.RecordArgs{
		Name: pulumi.String(domain),
		ZoneId: pulumi.String(hzid),
		Type: pulumi.String("A"),
		Aliases: route53.RecordAliasArray{
			&route53.RecordAliasArgs{
				Name: distribution.DomainName,
				ZoneId: distribution.HostedZoneId,
				EvaluateTargetHealth: pulumi.Bool(true),
			},
		},
	}); handleErr(err)
	log.Printf("Created alias record %v for domain %s\n", record.Name, domain)
}

func removeRootDir(path string) string {
	index := strings.Index(path, "/")
	if index != -1 {
		return path[index+1:]
	} else {
		return path
	}
}