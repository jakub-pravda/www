package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"fmt"
)

var nameId = "sramek-garden-store"

func main() {
	createS3Bucket()
}

func createS3Bucket() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create an AWS resource (S3 Bucket)
		bucketName := fmt.Sprintf("%s-bucket", nameId)
		bucket, err := s3.NewBucket(ctx, bucketName, &s3.BucketArgs{
			Website: s3.BucketWebsiteArgs{
				IndexDocument: pulumi.String("main.html"),
			},
		}); handleErr(err)

		_, err = s3.NewBucketOwnershipControls(ctx, "ownership-controls", &s3.BucketOwnershipControlsArgs{
			Bucket: bucket.ID(),
			Rule: &s3.BucketOwnershipControlsRuleArgs{
				ObjectOwnership: pulumi.String("ObjectWriter"),
			},
		}); handleErr(err)

		// set public access to our bucket
		publicAccessBlock, err := s3.NewBucketPublicAccessBlock(ctx, "public-access-block", &s3.BucketPublicAccessBlockArgs{
    		Bucket:          bucket.ID(),
    		BlockPublicAcls: pulumi.Bool(false),
		}); handleErr(err)

		_, err = filesToBucketObjects(ctx, publicAccessBlock, bucket, "./www"); handleErr(err)

		// Export the name of the bucket
		ctx.Export("bucketName", bucket.ID())
		ctx.Export("bucketEndpoint", bucket.WebsiteEndpoint.ApplyT(func(websiteEndpoint string) (string, error) {
    		return fmt.Sprintf("http://%v", websiteEndpoint), nil
		}).(pulumi.StringOutput))
		return nil
	})
}
