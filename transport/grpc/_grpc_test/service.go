package test

import "context"

type Service interface {
	Test(ctx context.Context, a string, b int64) (context.Context, string, error)
	TestB(ctx context.Context, a string, b int64) (context.Context, string, error)
	TestNil(ctx context.Context, i []byte) (context.Context, string, error)
}

type TestRequest struct {
	A string
	B int64
}

type TestNilRequest struct {
	A []byte
}

type TestResponse struct {
	Ctx context.Context
	V   string
}
