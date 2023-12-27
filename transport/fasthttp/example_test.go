package fasthttp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/tnnyio/yoroi/transport/fasthttp/fasthttptest"
	fh "github.com/valyala/fasthttp"
)

func ExamplePopulateRequestContext() {
	handler := NewServer(
		func(ctx context.Context, request interface{}) (response interface{}, err error) {
			fmt.Println("Method", ctx.Value(ContextKeyRequestMethod).(string))
			fmt.Println("RequestPath", ctx.Value(ContextKeyRequestPath).(string))
			fmt.Println("RequestURI", ctx.Value(ContextKeyRequestURI).(string))
			fmt.Println("X-Request-ID", ctx.Value(ContextKeyRequestXRequestID).(string))
			return struct{}{}, nil
		},
		func(_ *fh.RequestCtx) (interface{}, error) { return struct{}{}, nil },
		func(*fh.RequestCtx, interface{}) error { return nil },
		ServerBefore[interface{}, interface{}](PopulateRequestContext),
	)

	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/search?q=sympatico", "http://test/"), nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("X-Request-Id", "a1b2c3d4e5")
	res, err := fasthttptest.FastServerHandler(handler, req)
	if err != nil {
		fmt.Println(res)
		panic(err)
	}

	// Output:
	// Method PATCH
	// RequestPath /search
	// RequestURI http://test/search?q=sympatico
	// X-Request-ID a1b2c3d4e5
}
