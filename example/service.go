package main

import (
	"context"
)

type GreetService interface {
	TestA(context.Context, string, int64) (context.Context, string, error)
	TestB(context.Context, []byte) (context.Context, bool, error)
}

type TestARequest struct {
	A string
	B int64
}

type TestBRequest struct {
	A []byte
}

type TestResponse[T interface{}] struct {
	Ctx context.Context
	V   T
}
