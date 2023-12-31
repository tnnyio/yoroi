package jwt

import (
	"context"
	"encoding/pem"
	"sync"
	"testing"
	"time"

	"crypto/ecdsa"
	"crypto/subtle"
	"crypto/x509"

	"github.com/golang-jwt/jwt/v4"
	"github.com/tnnyio/yoroi/endpoint"
)

type customClaims struct {
	MyProperty string `json:"my_property"`
	jwt.RegisteredClaims
}

func (c customClaims) VerifyMyProperty(p string) bool {
	return subtle.ConstantTimeCompare([]byte(c.MyProperty), []byte(p)) != 0
}

var (
	kid            = "kid"
	key            = []byte("test_signing_key")
	myProperty     = "some value"
	method         = jwt.SigningMethodHS256
	invalidMethod  = jwt.SigningMethodRS256
	mapClaims      = jwt.MapClaims{"user": "yoroi"}
	standardClaims = jwt.RegisteredClaims{
		Audience: jwt.ClaimStrings{
			"yoroi",
		}}
	myCustomClaims = customClaims{MyProperty: myProperty, RegisteredClaims: standardClaims}
	// Signed tokens generated at https://jwt.io/
	signedKey         = "eyJhbGciOiJIUzI1NiIsImtpZCI6ImtpZCIsInR5cCI6IkpXVCJ9.eyJ1c2VyIjoieW9yb2kifQ.GfQQ3O1yVdG_6M04LLwWU85m2BBNlFaJknVn--fvYMU"
	standardSignedKey = "eyJhbGciOiJIUzI1NiIsImtpZCI6ImtpZCIsInR5cCI6IkpXVCJ9.eyJhdWQiOlsieW9yb2kiXX0.y-vffOic-coxhF3ezAJpIdqkOKUrA3WzhU3CW2OWAa8"
	customSignedKey   = "eyJhbGciOiJIUzI1NiIsImtpZCI6ImtpZCIsInR5cCI6IkpXVCJ9.eyJteV9wcm9wZXJ0eSI6InNvbWUgdmFsdWUiLCJhdWQiOlsieW9yb2kiXX0.bQPQSn3f9PKlBbvduxefNq1BkRDXutQTYRkslT515n4"
	invalidKey        = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.e30.vKVCKto-Wn6rgz3vBdaZaCBGfCBDTXOENSo_X2Gq7qA"
	malformedKey      = "malformed.jwt.token"
)

func signingValidator[O interface{}](t *testing.T, signer endpoint.Endpoint[O], expectedKey string) {
	var ctx interface{}
	ctx, err := signer(context.Background(), struct{}{})
	if err != nil {
		t.Fatalf("Signer returned error: %s", err)
	}

	token, ok := ctx.(context.Context).Value(JWTContextKey).(string)
	if !ok {
		t.Fatal("Token did not exist in context")
	}

	if token != expectedKey {
		t.Fatalf("JWTs did not match: expecting %s got %s", expectedKey, token)
	}
}

var es256Key = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgevZzL1gdAFr88hb2
OF/2NxApJCzGCEDdfSp6VQO30hyhRANCAAQRWz+jn65BtOMvdyHKcvjBeBSDZH2r
1RTwjmYSi9R/zpBnuQ4EiMnCqfMPWiZqB4QdbAd0E7oH50VpuZ1P087G
-----END PRIVATE KEY-----`

func es236PrivateKeyFromPem(t *testing.T, k string) *ecdsa.PrivateKey {
	block, _ := pem.Decode([]byte(k))
	if block == nil {
		t.Error("block is nil")
		return nil
	}
	p, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		t.Error(err)
	}
	key, ok := p.(*ecdsa.PrivateKey)
	if !ok {
		t.Error("Failed to get ecdsa key")
	}
	return key
}

func TestNewSigner(t *testing.T) {
	e := func(ctx context.Context, i interface{}) (context.Context, error) { return ctx, nil }

	signer := NewSigner[context.Context](kid, key, method, mapClaims)(e)
	signingValidator(t, signer, signedKey)

	signer = NewSigner[context.Context](kid, key, method, standardClaims)(e)
	signingValidator(t, signer, standardSignedKey)

	signer = NewSigner[context.Context](kid, key, method, myCustomClaims)(e)
	signingValidator(t, signer, customSignedKey)

	{
		// ES256
		pKey := es236PrivateKeyFromPem(t, es256Key)
		signer = NewSigner[context.Context](kid, pKey, jwt.SigningMethodES256, standardClaims)(e)
		ctx, err := signer(context.Background(), struct{}{})
		if err != nil {
			t.Fatalf("Signer returned error: %s", err)
		}

		_, ok := ctx.Value(JWTContextKey).(string)
		if !ok {
			t.Fatal("Token did not exist in context")
		}
	}
}

func TestJWTParser(t *testing.T) {
	e := func(ctx context.Context, i interface{}) (interface{}, error) { return ctx, nil }

	keys := func(token *jwt.Token) (interface{}, error) {
		return key, nil
	}

	parser := NewParser[interface{}](keys, method, MapClaimsFactory)(e)

	// No Token is passed into the parser
	_, err := parser(context.Background(), struct{}{})
	if err == nil {
		t.Error("Parser should have returned an error")
	}

	if err != ErrTokenContextMissing {
		t.Errorf("unexpected error returned, expected: %s got: %s", ErrTokenContextMissing, err)
	}

	// Invalid Token is passed into the parser
	ctx := context.WithValue(context.Background(), JWTContextKey, invalidKey)
	_, err = parser(ctx, struct{}{})
	if err == nil {
		t.Error("Parser should have returned an error")
	}

	// Invalid Method is used in the parser
	badParser := NewParser[interface{}](keys, invalidMethod, MapClaimsFactory)(e)
	ctx = context.WithValue(context.Background(), JWTContextKey, signedKey)
	_, err = badParser(ctx, struct{}{})
	if err == nil {
		t.Error("Parser should have returned an error")
	}

	if err != ErrUnexpectedSigningMethod {
		t.Errorf("unexpected error returned, expected: %s got: %s", ErrUnexpectedSigningMethod, err)
	}

	// Invalid key is used in the parser
	invalidKeys := func(token *jwt.Token) (interface{}, error) {
		return []byte("bad"), nil
	}

	badParser = NewParser[interface{}](invalidKeys, method, MapClaimsFactory)(e)
	ctx = context.WithValue(context.Background(), JWTContextKey, signedKey)
	_, err = badParser(ctx, struct{}{})
	if err == nil {
		t.Error("Parser should have returned an error")
	}

	// Correct token is passed into the parser
	ctx = context.WithValue(context.Background(), JWTContextKey, signedKey)
	ctx1, err := parser(ctx, struct{}{})
	if err != nil {
		t.Fatalf("Parser returned error: %s", err)
	}

	cl, ok := ctx1.(context.Context).Value(JWTClaimsContextKey).(jwt.MapClaims)
	if !ok {
		t.Fatal("Claims were not passed into context correctly")
	}

	if cl["user"] != mapClaims["user"] {
		t.Fatalf("JWT Claims.user did not match: expecting %s got %s", mapClaims["user"], cl["user"])
	}

	// Test for malformed token error response
	parser = NewParser[interface{}](keys, method, StandardClaimsFactory)(e)
	ctx = context.WithValue(context.Background(), JWTContextKey, malformedKey)
	_, err = parser(ctx, struct{}{})
	if want, have := ErrTokenMalformed, err; want != have {
		t.Fatalf("Expected %+v, got %+v", want, have)
	}

	// Test for expired token error response
	parser = NewParser[interface{}](keys, method, StandardClaimsFactory)(e)
	expired := jwt.NewWithClaims(method, jwt.StandardClaims{ExpiresAt: time.Now().Unix() - 100})
	token, err := expired.SignedString(key)
	if err != nil {
		t.Fatalf("Unable to Sign Token: %+v", err)
	}
	ctx = context.WithValue(context.Background(), JWTContextKey, token)
	_, err = parser(ctx, struct{}{})
	if want, have := ErrTokenExpired, err; want != have {
		t.Fatalf("Expected %+v, got %+v", want, have)
	}

	// Test for not activated token error response
	parser = NewParser[interface{}](keys, method, StandardClaimsFactory)(e)
	notactive := jwt.NewWithClaims(method, jwt.StandardClaims{NotBefore: time.Now().Unix() + 100})
	token, err = notactive.SignedString(key)
	if err != nil {
		t.Fatalf("Unable to Sign Token: %+v", err)
	}
	ctx = context.WithValue(context.Background(), JWTContextKey, token)
	_, err = parser(ctx, struct{}{})
	if want, have := ErrTokenNotActive, err; want != have {
		t.Fatalf("Expected %+v, got %+v", want, have)
	}

	// test valid standard claims token
	parser = NewParser[interface{}](keys, method, StandardClaimsFactory)(e)
	ctx = context.WithValue(context.Background(), JWTContextKey, standardSignedKey)
	ctx1, err = parser(ctx, struct{}{})
	if err != nil {
		t.Fatalf("Parser returned error: %s", err)
	}
	stdCl, ok := ctx1.(context.Context).Value(JWTClaimsContextKey).(*jwt.RegisteredClaims)
	if !ok {
		t.Fatal("Claims were not passed into context correctly")
	}
	if !stdCl.VerifyAudience("yoroi", true) {
		t.Fatalf("JWT jwt.StandardClaims.Audience did not match: expecting %s got %s", standardClaims.Audience, stdCl.Audience)
	}

	// test valid customized claims token
	parser = NewParser[interface{}](keys, method, func() jwt.Claims { return &customClaims{} })(e)
	ctx = context.WithValue(context.Background(), JWTContextKey, customSignedKey)
	ctx1, err = parser(ctx, struct{}{})
	if err != nil {
		t.Fatalf("Parser returned error: %s", err)
	}
	custCl, ok := ctx1.(context.Context).Value(JWTClaimsContextKey).(*customClaims)
	if !ok {
		t.Fatal("Claims were not passed into context correctly")
	}
	if !custCl.VerifyAudience("yoroi", true) {
		t.Fatalf("JWT customClaims.Audience did not match: expecting %s got %s", standardClaims.Audience, custCl.Audience)
	}
	if !custCl.VerifyMyProperty(myProperty) {
		t.Fatalf("JWT customClaims.MyProperty did not match: expecting %s got %s", myProperty, custCl.MyProperty)
	}
}

func TestIssue562(t *testing.T) {
	var (
		kf  = func(token *jwt.Token) (interface{}, error) { return []byte("secret"), nil }
		e   = NewParser[interface{}](kf, jwt.SigningMethodHS256, MapClaimsFactory)(endpoint.Nop)
		key = JWTContextKey
		val = "eyJhbGciOiJIUzI1NiIsImtpZCI6ImtpZCIsInR5cCI6IkpXVCJ9.eyJ1c2VyIjoiZ28ta2l0In0.14M2VmYyApdSlV_LZ88ajjwuaLeIFplB8JpyNy0A19E"
		ctx = context.WithValue(context.Background(), key, val)
	)
	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e(ctx, struct{}{}) // fatal error: concurrent map read and map write
		}()
	}
	wg.Wait()
}
