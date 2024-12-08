package main

import (
	"log"

	apigateway "github.com/pulumi/pulumi-aws-apigateway/sdk/go/apigateway"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/lambda"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ses"
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
