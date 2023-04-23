package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"os"
	"log"
	"mime"
	"path/filepath"
	"strings"
)

var wwwDir = "./www"

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
	dstFilePath := strings.Replace(path, wwwDir, "", 1)

	log.Printf("Converting file %s to bucket object with mime type %s\n", path, mimeType)
	return s3.NewBucketObject(ctx, path, &s3.BucketObjectArgs{
		Key: 			pulumi.String(dstFilePath),
		Bucket: 		bucket.ID(),
		Acl: 			pulumi.String("public-read"),
		Source: 		pulumi.NewFileAsset(path),
		ContentType: 	pulumi.String(mimeType),
	}, pulumi.DependsOn([]pulumi.Resource{accessBlock}))
}
