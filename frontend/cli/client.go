package main

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"quadsmith/api/gen/quadsmith/quadsmithconnect"
)

func newVerboseInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if msg, ok := req.Any().(proto.Message); ok {
				fmt.Printf(">>> REQUEST (%s) >>>\n%s\n", req.Spec().Procedure, protojson.Format(msg))
			}
			resp, err := next(ctx, req)
			if err != nil {
				fmt.Printf("<<< ERROR <<<\n%v\n", err)
			} else if msg, ok := resp.Any().(proto.Message); ok {
				fmt.Printf("<<< RESPONSE <<<\n%s\n", protojson.Format(msg))
			}
			return resp, err
		}
	}
}

func getClient() quadsmithconnect.QuadsmithAPIClient {
	if client == nil {
		opts := []connect.ClientOption{}
		if verbose {
			opts = append(opts, connect.WithInterceptors(newVerboseInterceptor()))
		}
		client = quadsmithconnect.NewQuadsmithAPIClient(
			http.DefaultClient,
			"http://localhost:8080",
			opts...,
		)
	}
	return client
}
