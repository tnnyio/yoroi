package test

import (
	"context"

	"github.com/tnnyio/yoroi/transport"
	"github.com/tnnyio/yoroi/transport/grpc/_grpc_test/pb"
)

func encodeRequest(ctx context.Context, req interface{}) (interface{}, error) {
	r, ok := req.(TestRequest)
	if !ok {
		return nil, transport.InvalidRequest
	}
	return &pb.TestRequest{A: r.A, B: r.B}, nil
}

func decodeRequest(ctx context.Context, req interface{}) (TestRequest, error) {
	r := req.(*pb.TestRequest)
	return TestRequest{A: r.A, B: r.B}, nil
}

func encodeNilRequest(ctx context.Context, req interface{}) (interface{}, error) {
	r, ok := req.(TestNilRequest)
	if !ok {
		return nil, transport.InvalidRequest
	}
	return &pb.TestNilRequest{A: r.A}, nil
}

func decodeNilRequest(ctx context.Context, req interface{}) (TestNilRequest, error) {
	r := req.(*pb.TestNilRequest)
	return TestNilRequest{A: r.A}, nil
}

func encodeResponse(ctx context.Context, resp *TestResponse) (interface{}, error) {
	return &pb.TestResponse{V: resp.V}, nil
}

func decodeResponse(ctx context.Context, resp interface{}) (interface{}, error) {
	r := resp.(*pb.TestResponse)
	return &TestResponse{V: r.V, Ctx: ctx}, nil
}
