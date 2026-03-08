// Proxy implementation from https://github.com/build-on-aws/golang-apis-on-aws-lambda/blob/main/http-stdlib/main.go

package main

import (
	"context"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
)

var httpLambda *httpadapter.HandlerAdapterV2

func init() {
	http.HandleFunc("/", authMiddleware(handler))

	httpLambda = httpadapter.NewV2(http.DefaultServeMux)
}

func Handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return httpLambda.ProxyWithContext(ctx, req)
}

func main() {
	lambda.Start(Handler)
}
