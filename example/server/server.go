package server

import (
	"context"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"gopkg.in/src-d/proteus.v1/example"

	"google.golang.org/grpc"
)

// NewServer returns a new server serving in the given address
func NewServer(addr string) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	grpcServer := grpc.NewServer()
	example.RegisterExampleServiceServer(grpcServer, example.NewExampleServiceServer())
	go grpcServer.Serve(lis)
	return grpcServer, nil
}

// RunGRPCGatewayServer executes the GRPC-Gateway server on the given addr.
// Will forward all requests to the GRPC server at the given endpoint.
func RunGRPCGatewayServer(addr, endpoint string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := example.RegisterExampleServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
	if err != nil {
		return err
	}

	return http.ListenAndServe(addr, mux)
}
