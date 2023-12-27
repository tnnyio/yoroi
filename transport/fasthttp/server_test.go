package fasthttp_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/tnnyio/yoroi/endpoint"
	fastTransport "github.com/tnnyio/yoroi/transport/fasthttp"
	"github.com/tnnyio/yoroi/transport/fasthttp/fasthttptest"
	fh "github.com/valyala/fasthttp"
)

func TestServerBadDecode(t *testing.T) {
	handler := fastTransport.NewServer(
		func(context.Context, interface{}) (interface{}, error) { return struct{}{}, nil },
		func(*fh.RequestCtx) (interface{}, error) { return struct{}{}, errors.New("dang") },
		func(*fh.RequestCtx, interface{}) error { return nil },
	)
	server := fasthttptest.FastServer(t, handler)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Error(err)
	}
	if want, have := http.StatusInternalServerError, resp.StatusCode; want != have {
		t.Errorf("want %d, have %d", want, have)
	}
}

func TestServerBadEndpoint(t *testing.T) {
	handler := fastTransport.NewServer(
		func(context.Context, interface{}) (interface{}, error) { return struct{}{}, errors.New("dang") },
		func(*fh.RequestCtx) (interface{}, error) { return struct{}{}, nil },
		func(*fh.RequestCtx, interface{}) error { return nil },
	)
	server := fasthttptest.FastServer(t, handler)
	defer server.Close()

	resp, _ := http.Get(server.URL)
	if want, have := http.StatusInternalServerError, resp.StatusCode; want != have {
		t.Errorf("want %d, have %d", want, have)
	}
}

func TestServerBadEncode(t *testing.T) {
	handler := fastTransport.NewServer(
		func(context.Context, interface{}) (interface{}, error) { return struct{}{}, nil },
		func(*fh.RequestCtx) (interface{}, error) { return struct{}{}, nil },
		func(*fh.RequestCtx, interface{}) error { return errors.New("dang") },
	)
	server := fasthttptest.FastServer(t, handler)
	defer server.Close()

	resp, _ := http.Get(server.URL)
	if want, have := http.StatusInternalServerError, resp.StatusCode; want != have {
		t.Errorf("want %d, have %d", want, have)
	}
}

func TestServerErrorEncoder(t *testing.T) {
	errTeapot := errors.New("teapot")
	code := func(err error) int {
		if errors.Is(err, errTeapot) {
			return http.StatusTeapot
		}
		return http.StatusInternalServerError
	}
	handler := fastTransport.NewServer(
		func(context.Context, interface{}) (interface{}, error) { return struct{}{}, errTeapot },
		func(*fh.RequestCtx) (interface{}, error) { return struct{}{}, nil },
		func(*fh.RequestCtx, interface{}) error { return nil },
		fastTransport.ServerErrorEncoder[interface{}, interface{}](func(ctx *fh.RequestCtx, err error) { ctx.SetStatusCode(code(err)) }),
	)
	server := fasthttptest.FastServer(t, handler)
	defer server.Close()

	resp, _ := http.Get(server.URL)
	if want, have := http.StatusTeapot, resp.StatusCode; want != have {
		t.Errorf("want %d, have %d", want, have)
	}
}

func TestServerHappyPath(t *testing.T) {
	step, response := testServer(t)
	step()
	resp := <-response
	defer resp.Body.Close()
	buf, _ := io.ReadAll(resp.Body)
	if want, have := http.StatusOK, resp.StatusCode; want != have {
		t.Errorf("want %d, have %d (%s)", want, have, buf)
	}
}

func TestMultipleServerBefore(t *testing.T) {
	var (
		headerKey    = "X-Henlo-Lizer"
		headerVal    = "Helllo you stinky lizard"
		statusCode   = http.StatusTeapot
		responseBody = "go eat a fly ugly\n"
		done         = make(chan struct{})
	)
	handler := fastTransport.NewServer(
		endpoint.Nop,
		func(*fh.RequestCtx) (interface{}, error) {
			return struct{}{}, nil
		},
		func(ctx *fh.RequestCtx, _ interface{}) error {
			ctx.Response.Header.Set(headerKey, headerVal)
			ctx.SetStatusCode(statusCode)
			ctx.SetBody([]byte(responseBody))
			return nil
		},
		fastTransport.ServerBefore[interface{}, interface{}](func(ctx *fh.RequestCtx) {
			ctx.SetUserValue("one", 1)
		}),
		fastTransport.ServerBefore[interface{}, interface{}](func(ctx *fh.RequestCtx) {
			if _, ok := ctx.Value("one").(int); !ok {
				t.Error("Value was not set properly when multiple ServerBefores are used")
			}
			close(done)
		}),
	)

	server := fasthttptest.FastServer(t, handler)
	defer server.Close()

	go http.Get(server.URL)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for finalizer")
	}
}

func TestMultipleServerAfter(t *testing.T) {
	var (
		headerKey    = "X-Henlo-Lizer"
		headerVal    = "Helllo you stinky lizard"
		statusCode   = http.StatusTeapot
		responseBody = "go eat a fly ugly\n"
		done         = make(chan struct{})
	)
	handler := fastTransport.NewServer(
		endpoint.Nop,
		func(*fh.RequestCtx) (interface{}, error) {
			return struct{}{}, nil
		},
		func(ctx *fh.RequestCtx, _ interface{}) error {
			ctx.Response.Header.Set(headerKey, headerVal)
			ctx.SetStatusCode(statusCode)
			ctx.SetBody([]byte(responseBody))
			return nil
		},
		fastTransport.ServerAfter[interface{}, interface{}](func(ctx *fh.RequestCtx) {
			ctx.SetUserValue("one", 1)
		}),
		fastTransport.ServerAfter[interface{}, interface{}](func(ctx *fh.RequestCtx) {
			if _, ok := ctx.Value("one").(int); !ok {
				t.Error("Value was not set properly when multiple ServerAfters are used")
			}

			close(done)
		}),
	)

	server := fasthttptest.FastServer(t, handler)
	defer server.Close()
	go http.Get(server.URL)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for finalizer")
	}
}

func TestServerFinalizer(t *testing.T) {
	var (
		headerKey    = "X-Henlo-Lizer"
		headerVal    = "Helllo you stinky lizard"
		statusCode   = http.StatusTeapot
		responseBody = "go eat a fly ugly\n"
		done         = make(chan struct{})
	)
	handler := fastTransport.NewServer(
		endpoint.Nop,
		func(*fh.RequestCtx) (interface{}, error) {
			return struct{}{}, nil
		},
		func(ctx *fh.RequestCtx, _ interface{}) error {
			ctx.Response.Header.Set(headerKey, headerVal)
			ctx.SetStatusCode(statusCode)
			ctx.Response.SetBody([]byte(responseBody))
			return nil
		},
		fastTransport.ServerFinalizer[interface{}, interface{}](func(ctx *fh.RequestCtx) {
			if want, have := statusCode, ctx.Response.StatusCode(); want != have {
				t.Errorf("StatusCode: want %d, have %d", want, have)
			}

			responseHeader := ctx.Value(fastTransport.ContextKeyResponseHeaders).(*fh.ResponseHeader)
			var header http.Header = make(http.Header)
			responseHeader.VisitAll(func(key, value []byte) {
				header[string(key)] = []string{string(value)}
			})
			if want, have := headerVal, header.Get(headerKey); want != have {
				t.Errorf("%s: want %q, have %q", headerKey, want, have)
			}

			responseSize := ctx.Value(fastTransport.ContextKeyResponseSize).(int64)
			if want, have := int64(len(responseBody)), responseSize; want != have {
				t.Errorf("response size: want %d, have %d", want, have)
			}

			close(done)
		}),
	)

	server := fasthttptest.FastServer(t, handler)
	defer server.Close()
	go http.Get(server.URL)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for finalizer")
	}
}

type enhancedResponse struct {
	Foo string `json:"foo"`
}

func (e enhancedResponse) StatusCode() int      { return http.StatusPaymentRequired }
func (e enhancedResponse) Headers() http.Header { return http.Header{"X-Edward": []string{"Snowden"}} }

func TestEncodeJSONResponse(t *testing.T) {
	handler := fastTransport.NewServer(
		func(context.Context, interface{}) (interface{}, error) { return enhancedResponse{Foo: "bar"}, nil },
		func(*fh.RequestCtx) (interface{}, error) { return struct{}{}, nil },
		fastTransport.EncodeJSONResponse,
	)

	server := fasthttptest.FastServer(t, handler)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if want, have := http.StatusPaymentRequired, resp.StatusCode; want != have {
		t.Errorf("StatusCode: want %d, have %d", want, have)
	}
	if want, have := "Snowden", resp.Header.Get("X-Edward"); want != have {
		t.Errorf("X-Edward: want %q, have %q", want, have)
	}
	buf, _ := io.ReadAll(resp.Body)
	if want, have := `{"foo":"bar"}`, strings.TrimSpace(string(buf)); want != have {
		t.Errorf("Body: want %s, have %s", want, have)
	}
}

type multiHeaderResponse struct{}

func (multiHeaderResponse) Headers() http.Header {
	return http.Header{"Vary": []string{"Origin", "User-Agent"}}
}

func TestAddMultipleHeaders(t *testing.T) {
	handler := fastTransport.NewServer(
		func(context.Context, interface{}) (interface{}, error) { return multiHeaderResponse{}, nil },
		func(*fh.RequestCtx) (interface{}, error) { return struct{}{}, nil },
		fastTransport.EncodeJSONResponse,
	)

	server := fasthttptest.FastServer(t, handler)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	expect := map[string]map[string]struct{}{"Vary": {"Origin": struct{}{}, "User-Agent": struct{}{}}}
	for k, vls := range resp.Header {
		for _, v := range vls {
			delete((expect[k]), v)
		}
		if len(expect[k]) != 0 {
			t.Errorf("Header: unexpected header %s: %v", k, expect[k])
		}
	}
}

type multiHeaderResponseError struct {
	multiHeaderResponse
	msg string
}

func (m multiHeaderResponseError) Error() string {
	return m.msg
}

func TestAddMultipleHeadersErrorEncoder(t *testing.T) {
	errStr := "oh no"
	handler := fastTransport.NewServer(
		func(context.Context, interface{}) (interface{}, error) {
			return nil, multiHeaderResponseError{msg: errStr}
		},
		func(*fh.RequestCtx) (interface{}, error) { return struct{}{}, nil },
		fastTransport.EncodeJSONResponse,
	)

	server := fasthttptest.FastServer(t, handler)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	expect := map[string]map[string]struct{}{"Vary": {"Origin": struct{}{}, "User-Agent": struct{}{}}}
	for k, vls := range resp.Header {
		for _, v := range vls {
			delete((expect[k]), v)
		}
		if len(expect[k]) != 0 {
			t.Errorf("Header: unexpected header %s: %v", k, expect[k])
		}
	}
	if b, _ := io.ReadAll(resp.Body); errStr != string(b) {
		t.Errorf("ErrorEncoder: got: %q, expected: %q", b, errStr)
	}
}

type noContentResponse struct{}

func (e noContentResponse) StatusCode() int { return http.StatusNoContent }

func TestEncodeNoContent(t *testing.T) {
	handler := fastTransport.NewServer(
		func(context.Context, interface{}) (interface{}, error) { return noContentResponse{}, nil },
		func(*fh.RequestCtx) (interface{}, error) { return struct{}{}, nil },
		fastTransport.EncodeJSONResponse,
	)

	server := fasthttptest.FastServer(t, handler)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if want, have := http.StatusNoContent, resp.StatusCode; want != have {
		t.Errorf("StatusCode: want %d, have %d", want, have)
	}
	buf, _ := io.ReadAll(resp.Body)
	if want, have := 0, len(buf); want != have {
		t.Errorf("Body: want no content, have %d bytes", have)
	}
}

type enhancedError struct{}

func (e enhancedError) Error() string                { return "enhanced error" }
func (e enhancedError) StatusCode() int              { return http.StatusTeapot }
func (e enhancedError) MarshalJSON() ([]byte, error) { return []byte(`{"err":"enhanced"}`), nil }
func (e enhancedError) Headers() http.Header         { return http.Header{"X-Enhanced": []string{"1"}} }

func TestEnhancedError(t *testing.T) {
	handler := fastTransport.NewServer(
		func(context.Context, interface{}) (interface{}, error) { return nil, enhancedError{} },
		func(*fh.RequestCtx) (interface{}, error) { return struct{}{}, nil },
		func(_ *fh.RequestCtx, _ interface{}) error { return nil },
	)

	server := fasthttptest.FastServer(t, handler)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if want, have := http.StatusTeapot, resp.StatusCode; want != have {
		t.Errorf("StatusCode: want %d, have %d", want, have)
	}
	if want, have := "1", resp.Header.Get("X-Enhanced"); want != have {
		t.Errorf("X-Enhanced: want %q, have %q", want, have)
	}
	buf, _ := io.ReadAll(resp.Body)
	if want, have := `{"err":"enhanced"}`, strings.TrimSpace(string(buf)); want != have {
		t.Errorf("Body: want %s, have %s", want, have)
	}
}

func TestNoOpRequestDecoder(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://test/", nil)
	if err != nil {
		t.Error("Failed to create request")
	}
	handler := fastTransport.NewServer(
		func(ctx context.Context, request interface{}) (interface{}, error) {
			if request != nil {
				t.Error("Expected nil request in endpoint when using NopRequestDecoder")
			}
			return nil, nil
		},
		fastTransport.NopRequestDecoder,
		fastTransport.EncodeJSONResponse,
	)
	resp, err := fasthttptest.FastServerHandler(handler, req)
	if err != nil {
		t.Error(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d but got %d", http.StatusOK, resp.StatusCode)
	}
}

func testServer(t *testing.T) (step func(), resp <-chan *http.Response) {
	var (
		stepch   = make(chan bool)
		endpoint = func(context.Context, interface{}) (interface{}, error) { <-stepch; return struct{}{}, nil }
		response = make(chan *http.Response)
		handler  = fastTransport.NewServer(
			endpoint,
			func(*fh.RequestCtx) (interface{}, error) { return struct{}{}, nil },
			func(*fh.RequestCtx, interface{}) error { return nil },
			fastTransport.ServerBefore[interface{}, interface{}](func(*fh.RequestCtx) {}),
			fastTransport.ServerAfter[interface{}, interface{}](func(*fh.RequestCtx) {}),
		)
	)
	go func() {
		server := fasthttptest.FastServer(t, handler)
		defer server.Close()

		resp, err := http.Get(server.URL)
		if err != nil {
			t.Error(err)
			return
		}
		response <- resp
	}()
	return func() { stepch <- true }, response
}
