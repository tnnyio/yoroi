package main

import (
	"context"
	"fmt"

	"github.com/tnnyio/yoroi/example/src/proto"
	grpcTransport "github.com/tnnyio/yoroi/transport/grpc"
)

type greetService struct{}

func (*greetService) TestA(ctx context.Context, a string, b int64) (context.Context, string, error) {
	fmt.Println("Great called")
	return context.TODO(), fmt.Sprintf("%s = %d", a, b), nil
}

func (*greetService) TestB(ctx context.Context, b []byte) (context.Context, bool, error) {
	return context.TODO(), true, nil
}

func NewGreetService() *greetService {
	return &greetService{}
}

type greetServiceBinding struct {
	proto.UnimplementedGreetServiceServer
	testA grpcTransport.Handler
	testB grpcTransport.Handler
}

func (b *greetServiceBinding) TestA(ctx context.Context, i *proto.TestARequest) (*proto.TestAResponse, error) {
	_, v, err := b.testA.ServeGRPC(ctx, i)
	if err != nil {
		return nil, err
	}
	return v.(*proto.TestAResponse), err
}

func (b *greetServiceBinding) TestB(ctx context.Context, i *proto.TestBRequest) (*proto.TestBResponse, error) {
	_, v, err := b.testB.ServeGRPC(ctx, i)
	if err != nil {
		return nil, err
	}
	return v.(*proto.TestBResponse), err
}

func NewGreetServiceBinding(svc GreetService) *greetServiceBinding {
	return &greetServiceBinding{
		testA: grpcTransport.NewServer[TestARequest, *TestResponse[string]](
			// NewEndpoint[TestARequest, TestResponse[string]](svc),
			func(ctx context.Context, request interface{}) (*TestResponse[string], error) {
				req := request.(TestARequest)
				newCtx, v, err := svc.TestA(ctx, req.A, req.B)
				return &TestResponse[string]{
					V:   v,
					Ctx: newCtx,
				}, err
			},
			decodeTestARequest,
			encodeTestAResponse,
		),
	}
}

func decodeTestARequest(ctx context.Context, i interface{}) (TestARequest, error) {
	r := i.(*proto.TestARequest)
	return TestARequest{A: r.A, B: r.B}, nil
}

func encodeTestAResponse(ctx context.Context, r *TestResponse[string]) (interface{}, error) {
	return &proto.TestAResponse{V: r.V}, nil
}
