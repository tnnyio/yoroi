package endpoint_test

import (
	"context"
	"fmt"

	"github.com/tnnyio/yoroi/endpoint"
)

func ExampleChain() {
	e := endpoint.Chain[int](
		annotate[int]("first"),
		annotate[int]("second"),
		annotate[int]("third"),
	)(myEndpoint[int])

	if _, err := e(ctx, req); err != nil {
		panic(err)
	}

	// Output:
	// first pre
	// second pre
	// third pre
	// my endpoint!
	// third post
	// second post
	// first post
}

var (
	ctx = context.Background()
	req = struct{}{}
)

func annotate[T interface{}](s string) endpoint.Middleware[T] {
	return func(next endpoint.Endpoint[T]) endpoint.Endpoint[T] {
		return func(ctx context.Context, request interface{}) (T, error) {
			fmt.Println(s, "pre")
			defer fmt.Println(s, "post")
			return next(ctx, request)
		}
	}
}

func myEndpoint[T interface{}](context.Context, interface{}) (r T, err error) {
	fmt.Println("my endpoint!")
	return r, nil
}
