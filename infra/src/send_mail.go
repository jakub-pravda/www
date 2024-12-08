package main

import (
	"log"

	"github.com/pulumi/pulumi-archive/sdk/go/archive"
	apigateway "github.com/pulumi/pulumi-aws-apigateway/sdk/go/apigateway"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/lambda"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ses"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func emailFormApiGateway(ctx *pulumi.Context, lambda_mail *lambda.Function, lambda_cors *lambda.Function) *apigateway.RestAPI {
	postMethod := apigateway.MethodPOST
	optionMethod := apigateway.MethodOPTIONS
	restAPI, err := apigateway.NewRestAPI(ctx, "email-form", &apigateway.RestAPIArgs{
		StageName: pulumi.String("prod"),
		Routes: []apigateway.RouteArgs{
			{
				Path:         "/",
				Method:       &postMethod,
				EventHandler: lambda_mail,
			},
			{
				Path:         "/",
				Method:       &optionMethod,
				EventHandler: lambda_cors,
			},
		},
	})
	handleErr(err)
	return restAPI
}

func simpleMailService(ctx *pulumi.Context, emailDomain string) {
	log.Println("emailForm - Setting AWS SES")

	log.Println("emailForm - Setting AWS SES email identities")
	sesIdentity(ctx, emailDomain)

	log.Println("emailForm - Setting lambda functions")
	lambdaEmailForm := lambdaEmailForm(ctx)
	lambdaCors := lambdaEmailFormCors(ctx)

	log.Println("emailForm - API Gateway settings")
	restApi := emailFormApiGateway(ctx, lambdaEmailForm, lambdaCors)

	log.Println("emailForm - Setting AWS SES complete")
	ctx.Export("email_form_url", &restApi.Url)
}

func sesIdentity(ctx *pulumi.Context, emailDomain string) {
	_, err := ses.NewDomainIdentity(ctx, emailDomain, &ses.DomainIdentityArgs{
		Domain: pulumi.String(emailDomain),
	})
	handleErr(err)
	// TODO - verify domain (after migration domain to AWS)
}

func lambdaEmailForm(ctx *pulumi.Context) *lambda.Function {
	lambdaArchive := "../dist/lambda_send_mail.zip"

	assumeRole, err := iam.GetPolicyDocument(ctx, &iam.GetPolicyDocumentArgs{
		Statements: []iam.GetPolicyDocumentStatement{
			{
				Effect: pulumi.StringRef("Allow"),
				Principals: []iam.GetPolicyDocumentStatementPrincipal{
					{
						Type: "Service",
						Identifiers: []string{
							"lambda.amazonaws.com",
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

	log.Println("Creating IAM for sending mail lambda")
	iamForLambda, err := iam.NewRole(ctx, "iam_for_lambda", &iam.RoleArgs{
		Name:             pulumi.String("iam_for_lambda"),
		AssumeRolePolicy: pulumi.String(assumeRole.Json),
	})
	handleErr(err)

	log.Println("Updating lambda policies to allow sending mails")

	sesPolicy, err := iam.NewPolicy(ctx, "ses_policy", &iam.PolicyArgs{
		Description: pulumi.String("Policy to allow sending mails through lambda"),
		Policy: pulumi.String(`{
            "Version": "2012-10-17",
            "Statement": [
                {
                    "Effect": "Allow",
                    "Action": [
                        "ses:SendEmail",
                        "ses:SendRawEmail"
                    ],
                    "Resource": "*"
                }
            ]
        }`),
	})
	handleErr(err)

	_, err = iam.NewRolePolicyAttachment(ctx, "ses_policy_attachment", &iam.RolePolicyAttachmentArgs{
		Role:      iamForLambda.Name,
		PolicyArn: sesPolicy.Arn,
	})
	handleErr(err)

	log.Println("Archiving send_mail.js lambda")
	lambda_lookup_file, err := archive.LookupFile(ctx, &archive.LookupFileArgs{
		Type:       "zip",
		SourceFile: pulumi.StringRef("./lambda/lambda_send_mail.mjs"),
		OutputPath: lambdaArchive,
	}, nil)
	handleErr(err)

	lambdaLogging := lambdaEmailFormLogs(ctx, iamForLambda, "email_form")

	// Create the Lambda Function itself
	log.Println("Creating email_form lambda")
	lambdaFunction, err := lambda.NewFunction(ctx, "email_form", &lambda.FunctionArgs{
		Code:           pulumi.NewFileArchive(lambdaArchive),
		Name:           pulumi.String("lambda-email-form"),
		Role:           iamForLambda.Arn,
		Handler:        pulumi.String("lambda_send_mail.handler"),
		SourceCodeHash: pulumi.String(lambda_lookup_file.OutputBase64sha256),
		Runtime:        pulumi.String(lambda.RuntimeNodeJS18dX),
	}, pulumi.DependsOn([]pulumi.Resource{lambdaLogging}))

	if err != nil {
		handleErr(err)
	}
	return lambdaFunction
}

func lambdaEmailFormCors(ctx *pulumi.Context) *lambda.Function {
	lambdaArchive := "../dist/lambda_cors.zip"

	assumeRole, err := iam.GetPolicyDocument(ctx, &iam.GetPolicyDocumentArgs{
		Statements: []iam.GetPolicyDocumentStatement{
			{
				Effect: pulumi.StringRef("Allow"),
				Principals: []iam.GetPolicyDocumentStatementPrincipal{
					{
						Type: "Service",
						Identifiers: []string{
							"lambda.amazonaws.com",
						},
					},
				},
				Actions: []string{
					"sts:AssumeRole",
				},
			},
		},
	}, nil)

	if err != nil {
		handleErr(err)
	}

	log.Println("Creating IAM for cors lambda")
	iamForLambda, err := iam.NewRole(ctx, "lambda_cors_iam", &iam.RoleArgs{
		Name:             pulumi.String("lambda_cors_iam"),
		AssumeRolePolicy: pulumi.String(assumeRole.Json),
	})

	if err != nil {
		handleErr(err)
	}

	log.Println("Archiving cors.mjs lambda")
	lambda_lookup_file, err := archive.LookupFile(ctx, &archive.LookupFileArgs{
		Type:       "zip",
		SourceFile: pulumi.StringRef("./lambda/lambda_cors.mjs"),
		OutputPath: lambdaArchive,
	}, nil)

	if err != nil {
		handleErr(err)
	}

	log.Println("Creating cors lambda")
	lambdaFunction, err := lambda.NewFunction(ctx, "cors", &lambda.FunctionArgs{
		Code:           pulumi.NewFileArchive(lambdaArchive),
		Name:           pulumi.String("lambda-cors"),
		Role:           iamForLambda.Arn,
		Handler:        pulumi.String("lambda_cors.handler"),
		SourceCodeHash: pulumi.String(lambda_lookup_file.OutputBase64sha256),
		Runtime:        pulumi.String(lambda.RuntimeNodeJS18dX),
	})

	if err != nil {
		handleErr(err)
	}
	return lambdaFunction
}

func lambdaEmailFormLogs(ctx *pulumi.Context, iamRole *iam.Role, name string) *iam.RolePolicyAttachment {
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

	loggingPolicyName := name + "_logging"

	lambdaLoggingPolicy, err := iam.NewPolicy(ctx, loggingPolicyName, &iam.PolicyArgs{
		Name:        pulumi.String(loggingPolicyName),
		Path:        pulumi.String("/"),
		Description: pulumi.String("IAM policy for logging from a lambda"),
		Policy:      pulumi.String(lambdaLoggingPolicyDocument.Json),
	})
	handleErr(err)

	lambdaLogs, err := iam.NewRolePolicyAttachment(ctx, "lambda_logs", &iam.RolePolicyAttachmentArgs{
		Role:      pulumi.Any(iamRole.Name),
		PolicyArn: lambdaLoggingPolicy.Arn,
	})
	handleErr(err)

	return lambdaLogs
}
