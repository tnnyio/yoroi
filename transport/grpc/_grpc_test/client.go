package test

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"github.com/tnnyio/yoroi/endpoint"
	grpctransport "github.com/tnnyio/yoroi/transport/grpc"
	"github.com/tnnyio/yoroi/transport/grpc/_grpc_test/pb"
)

type clientBinding[O interface{}] struct {
	test    endpoint.Endpoint[O]
	testB   endpoint.Endpoint[O]
	testNil endpoint.Endpoint[O]
}

func (c *clientBinding[O]) Test(ctx context.Context, a string, b int64) (context.Context, string, error) {
	var response interface{}
	var err error
	response, err = c.test(ctx, TestRequest{A: a, B: b})
	if err != nil {
		return nil, "", err
	}
	r, ok := response.(*TestResponse)
	if !ok {
		return nil, "", err
	}
	return r.Ctx, r.V, nil
}

func (c *clientBinding[O]) TestB(ctx context.Context, a string, b int64) (context.Context, string, error) {
	var response interface{}
	var err error
	response, err = c.testB(ctx, TestRequest{A: a, B: b})
	if err != nil {
		return nil, "", err
	}
	r, ok := response.(*TestResponse)
	if !ok {
		return nil, "", err
	}
	return r.Ctx, r.V, nil
}

func (c *clientBinding[O]) TestNil(ctx context.Context, i []byte) (context.Context, string, error) {
	var response interface{}
	var err error
	response, err = c.testNil(ctx, TestNilRequest{A: i})
	if err != nil {
		return nil, "", err
	}
	r, ok := response.(*TestResponse)
	if !ok {
		return nil, "", err
	}
	return r.Ctx, r.V, nil
}

func NewClient[O interface{}](cc *grpc.ClientConn) Service {
	c := grpctransport.NewClient[TestRequest, O](
		cc,
		"pb.Test",
		"Test",
		func(ctx context.Context, i TestRequest) (request interface{}, err error) {
			return encodeRequest(ctx, i)
		},
		func(ctx context.Context, i interface{}) (response O, err error) {
			r, err := decodeResponse(ctx, i)
			{
				var ok bool
				if response, ok = r.(O); !ok {
					return response, fmt.Errorf("error decoding response")
				}
			}
			return
		},
		&pb.TestResponse{},
		grpctransport.ClientBefore[TestRequest, O](
			injectCorrelationID,
		),
		grpctransport.ClientBefore[TestRequest, O](
			displayClientRequestHeaders,
		),
		grpctransport.ClientAfter[TestRequest, O](
			displayClientResponseHeaders,
			displayClientResponseTrailers,
		),
		grpctransport.ClientAfter[TestRequest, O](
			extractConsumedCorrelationID,
		),
	)

	cb := grpctransport.NewClient[TestRequest, O](
		cc,
		"pb.Test",
		"TestB",
		func(ctx context.Context, i TestRequest) (request interface{}, err error) {
			return encodeRequest(ctx, i)
		},
		func(ctx context.Context, i interface{}) (response O, err error) {
			r, err := decodeResponse(ctx, i)
			{
				var ok bool
				if response, ok = r.(O); !ok {
					return response, fmt.Errorf("error decoding response")
				}
			}
			return
		},
		&pb.TestResponse{},
		grpctransport.ClientBefore[TestRequest, O](
			injectCorrelationID,
		),
		grpctransport.ClientBefore[TestRequest, O](
			displayClientRequestHeaders,
		),
		grpctransport.ClientAfter[TestRequest, O](
			displayClientResponseHeaders,
			displayClientResponseTrailers,
		),
		grpctransport.ClientAfter[TestRequest, O](
			extractConsumedCorrelationID,
		),
	)

	cn := grpctransport.NewClient[TestNilRequest, O](
		cc,
		"pb.TestNil",
		"TestNil",
		func(ctx context.Context, i TestNilRequest) (request interface{}, err error) {
			return encodeNilRequest(ctx, i)
		},
		func(ctx context.Context, i interface{}) (response O, err error) {
			r, err := decodeResponse(ctx, i)
			{
				var ok bool
				if response, ok = r.(O); !ok {
					return response, fmt.Errorf("error decoding response")
				}
			}
			return
		},
		&pb.TestResponse{},
		grpctransport.ClientBefore[TestNilRequest, O](
			injectCorrelationID,
		),
		grpctransport.ClientBefore[TestNilRequest, O](
			displayClientRequestHeaders,
		),
		grpctransport.ClientAfter[TestNilRequest, O](
			displayClientResponseHeaders,
			displayClientResponseTrailers,
		),
		grpctransport.ClientAfter[TestNilRequest, O](
			extractConsumedCorrelationID,
		),
	)
	return &clientBinding[O]{
		test:    c.Endpoint(),
		testB:   cb.Endpoint(),
		testNil: cn.Endpoint(),
	}
}
