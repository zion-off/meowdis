// Proxy implementation from https://github.com/build-on-aws/golang-apis-on-aws-lambda/blob/main/http-stdlib/main.go

package main

import (
	"context"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
)

var httpLambda *httpadapter.HandlerAdapter

func init() {
	http.HandleFunc("/", handler)

	httpLambda = httpadapter.New(http.DefaultServeMux)
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return httpLambda.ProxyWithContext(ctx, req)
}

func main() {
	lambda.Start(Handler)
}
