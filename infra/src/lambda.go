package main

import (
	"fmt"
	"log"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws"
	"github.com/pulumi/pulumi-archive/sdk/go/archive"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/lambda"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func lambdaRedirect(ctx *pulumi.Context) *lambda.Function {
	lambdaName := "lambda-redirect"
	lambdaArchive := fmt.Sprintf("../dist/%s.zip", lambdaName)

	eastRegion, err := aws.NewProvider(ctx, fmt.Sprintf("lambda-redirect-east"), &aws.ProviderArgs{
		Region: pulumi.String("us-east-1"),
	})
	
	assumeRole, err := iam.GetPolicyDocument(ctx, &iam.GetPolicyDocumentArgs{
		Statements: []iam.GetPolicyDocumentStatement{
			{
				Effect: pulumi.StringRef("Allow"),
				Principals: []iam.GetPolicyDocumentStatementPrincipal{
					{
						Type: "Service",
						Identifiers: []string{
							"lambda.amazonaws.com",
							"edgelambda.amazonaws.com",
						},
					},
				},
				Actions: []string{
					"sts:AssumeRole",
				},
			},
		},
	}, nil)
	handleErr(err)

	log.Println("Creating IAM for redirect lambda")
	iamName := fmt.Sprintf("%s-iam", lambdaName)
	iamForLambda, err := iam.NewRole(ctx, iamName, &iam.RoleArgs{
		Name:             pulumi.String(iamName),
		AssumeRolePolicy: pulumi.String(assumeRole.Json),
	})
	handleErr(err)

	log.Println("Updating lambda policies to allow sending mails")

	cloudfrontPolicy, err := iam.NewPolicy(ctx, "cloudfront_policy", &iam.PolicyArgs{
		Description: pulumi.String("Policy to allow lambda edge execution"),
		Policy: pulumi.String(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Action": [
						"lambda:GetFunction",
						"lambda:EnableReplication*",
						"lambda:DisableReplication*",
						"iam:CreateServiceLinkedRole",
						"cloudfront:UpdateDistribution",
						"cloudfront:UpdateDistribution"
					],
					"Resource": "*"
				}
			]
		}`),
	})
	handleErr(err)

	_, err = iam.NewRolePolicyAttachment(ctx, "cloudfront_policy_attachment", &iam.RolePolicyAttachmentArgs{
		Role:      iamForLambda.Name,
		PolicyArn: cloudfrontPolicy.Arn,
	})
	handleErr(err)

	log.Println("Archiving redirect mjs lambda")
	lambda_lookup_file, err := archive.LookupFile(ctx, &archive.LookupFileArgs{
		Type:       "zip",
		SourceFile: pulumi.StringRef("./lambda/lambda_redirect.mjs"),
		OutputPath: lambdaArchive,
	}, nil)
	handleErr(err)

	lambdaLogging := lambdaLogs(ctx, iamForLambda, lambdaName)

	// Create the Lambda Function itself
	log.Println("Creating redirect lambda")
	lambdaFunction, err := lambda.NewFunction(ctx, lambdaName, &lambda.FunctionArgs{
		Code:           pulumi.NewFileArchive(lambdaArchive),
		Name:           pulumi.String(lambdaName),
		Role:           iamForLambda.Arn,
		Handler:        pulumi.String("lambda_redirect.handler"),
		SourceCodeHash: pulumi.String(lambda_lookup_file.OutputBase64sha256),
		Runtime:        pulumi.String(lambda.RuntimeNodeJS18dX),
		Publish: 	  	pulumi.Bool(true),
	}, pulumi.DependsOn([]pulumi.Resource{lambdaLogging}), pulumi.Provider(eastRegion))

	if err != nil {
		handleErr(err)
	}

	// Export outputs
	ctx.Export("lambda_redirect_arn", lambdaFunction.QualifiedArn)

	return lambdaFunction
}

func lambdaLogs(ctx *pulumi.Context, iamRole *iam.Role, name string) *iam.RolePolicyAttachment {
	// Create log group for lambda
	lambdaLoggingPolicyDocument, err := iam.GetPolicyDocument(ctx, &iam.GetPolicyDocumentArgs{
		Statements: []iam.GetPolicyDocumentStatement{
			{
				Effect: pulumi.StringRef("Allow"),
				Actions: []string{
					"logs:CreateLogGroup",
					"logs:CreateLogStream",
					"logs:PutLogEvents",
				},
				Resources: []string{
					"arn:aws:logs:*:*:*",
				},
			},
		},
	}, nil)
	handleErr(err)

	loggingPolicyName := fmt.Sprintf("%s-logging", name)
	
	lambdaLoggingPolicy, err := iam.NewPolicy(ctx, loggingPolicyName, &iam.PolicyArgs{
		Name:        pulumi.String(loggingPolicyName),
		Path:        pulumi.String("/"),
		Description: pulumi.String("IAM policy for logging from a lambda"),
		Policy:      pulumi.String(lambdaLoggingPolicyDocument.Json),
	})
	handleErr(err)

	lambdaLogs, err := iam.NewRolePolicyAttachment(ctx, name, &iam.RolePolicyAttachmentArgs{
		Role:      pulumi.Any(iamRole.Name),
		PolicyArn: lambdaLoggingPolicy.Arn,
	})
	handleErr(err)

	return lambdaLogs
}
