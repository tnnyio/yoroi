package test

import (
	"context"
	"fmt"

	"github.com/tnnyio/yoroi/endpoint"
	grpctransport "github.com/tnnyio/yoroi/transport/grpc"
	"github.com/tnnyio/yoroi/transport/grpc/_grpc_test/pb"
)

type service struct{}

type testServerBinding struct {
	pb.UnimplementedTestServer
	methodA grpctransport.Handler
	methodB grpctransport.Handler
}

type testNilServerBinding struct {
	pb.UnimplementedTestNilServer
	methodA grpctransport.Handler
}

func (service) Test(ctx context.Context, a string, b int64) (context.Context, string, error) {
	return context.TODO(), fmt.Sprintf("%s = %d", a, b), nil
}

func (service) TestB(ctx context.Context, a string, b int64) (context.Context, string, error) {
	return context.TODO(), fmt.Sprintf("%d = %s", b, a), nil
}

func (service) TestNil(ctx context.Context, i []byte) (context.Context, string, error) {
	return context.TODO(), fmt.Sprintf("%v", i), nil
}

func NewService() Service {
	return service{}
}

func makeTestEndpoint(svc Service) endpoint.Endpoint[*TestResponse] {
	return func(ctx context.Context, request interface{}) (*TestResponse, error) {
		req := request.(TestRequest)
		newCtx, v, err := svc.Test(ctx, req.A, req.B)
		return &TestResponse{
			V:   v,
			Ctx: newCtx,
		}, err
	}
}

func makeTestBEndpoint(svc Service) endpoint.Endpoint[*TestResponse] {
	return func(ctx context.Context, request interface{}) (*TestResponse, error) {
		req := request.(TestRequest)
		newCtx, v, err := svc.TestB(ctx, req.A, req.B)
		return &TestResponse{
			V:   v,
			Ctx: newCtx,
		}, err
	}
}

func makeTestNilEndpoint(svc Service) endpoint.Endpoint[*TestResponse] {
	return func(ctx context.Context, request interface{}) (*TestResponse, error) {
		req := request.(TestNilRequest)
		newCtx, v, err := svc.TestNil(ctx, req.A)
		return &TestResponse{
			V:   v,
			Ctx: newCtx,
		}, err
	}
}

func (b *testServerBinding) Test(ctx context.Context, req *pb.TestRequest) (*pb.TestResponse, error) {
	_, response, err := b.methodA.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return response.(*pb.TestResponse), nil
}

func (b *testServerBinding) TestB(ctx context.Context, req *pb.TestRequest) (*pb.TestResponse, error) {
	_, response, err := b.methodB.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return response.(*pb.TestResponse), nil
}

func (b *testNilServerBinding) TestNil(ctx context.Context, req *pb.TestNilRequest) (*pb.TestResponse, error) {
	_, response, err := b.methodA.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return response.(*pb.TestResponse), nil
}

func NewTestServiceBinding(svc Service) *testServerBinding {
	return &testServerBinding{
		methodA: grpctransport.NewServer[TestRequest, *TestResponse](
			makeTestEndpoint(svc),
			decodeRequest,
			encodeResponse,
			grpctransport.ServerBefore[TestRequest, *TestResponse](
				extractCorrelationID,
			),
			grpctransport.ServerBefore[TestRequest, *TestResponse](
				displayServerRequestHeaders,
			),
			grpctransport.ServerAfter[TestRequest, *TestResponse](
				injectResponseHeader,
				injectResponseTrailer,
				injectConsumedCorrelationID,
			),
			grpctransport.ServerAfter[TestRequest, *TestResponse](
				displayServerResponseHeaders,
				displayServerResponseTrailers,
			),
		),
		methodB: grpctransport.NewServer[TestRequest, *TestResponse](
			makeTestBEndpoint(svc),
			decodeRequest,
			encodeResponse,
			grpctransport.ServerBefore[TestRequest, *TestResponse](
				extractCorrelationID,
			),
			grpctransport.ServerBefore[TestRequest, *TestResponse](
				displayServerRequestHeaders,
			),
			grpctransport.ServerAfter[TestRequest, *TestResponse](
				injectResponseHeader,
				injectResponseTrailer,
				injectConsumedCorrelationID,
			),
			grpctransport.ServerAfter[TestRequest, *TestResponse](
				displayServerResponseHeaders,
				displayServerResponseTrailers,
			),
		),
	}
}

func NewTestNilServiceBinding(svc Service) *testNilServerBinding {
	return &testNilServerBinding{
		methodA: grpctransport.NewServer[TestNilRequest, *TestResponse](
			makeTestNilEndpoint(svc),
			decodeNilRequest,
			encodeResponse,
			grpctransport.ServerBefore[TestNilRequest, *TestResponse](
				extractCorrelationID,
			),
			grpctransport.ServerBefore[TestNilRequest, *TestResponse](
				displayServerRequestHeaders,
			),
			grpctransport.ServerAfter[TestNilRequest, *TestResponse](
				injectResponseHeader,
				injectResponseTrailer,
				injectConsumedCorrelationID,
			),
			grpctransport.ServerAfter[TestNilRequest, *TestResponse](
				displayServerResponseHeaders,
				displayServerResponseTrailers,
			),
		),
	}
}
