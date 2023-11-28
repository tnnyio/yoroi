package jwt

import (
	"context"
	"errors"

	"github.com/golang-jwt/jwt/v4"
	"github.com/tnnyio/yoroi/endpoint"
)

type contextKey string

const (
	// JWTContextKey holds the key used to store a JWT in the context.
	JWTContextKey contextKey = "JWTToken"

	// JWTTokenContextKey is an alias for JWTContextKey.
	//
	// Deprecated: prefer JWTContextKey.
	JWTTokenContextKey = JWTContextKey

	// JWTClaimsContextKey holds the key used to store the JWT Claims in the
	// context.
	JWTClaimsContextKey contextKey = "JWTClaims"
)

var (
	// ErrTokenContextMissing denotes a token was not passed into the parsing
	// middleware's context.
	ErrTokenContextMissing = errors.New("token up for parsing was not passed through the context")

	// ErrTokenInvalid denotes a token was not able to be validated.
	ErrTokenInvalid = errors.New("JWT was invalid")

	// ErrTokenExpired denotes a token's expire header (exp) has since passed.
	ErrTokenExpired = errors.New("JWT is expired")

	// ErrTokenMalformed denotes a token was not formatted as a JWT.
	ErrTokenMalformed = errors.New("JWT is malformed")

	// ErrTokenNotActive denotes a token's not before header (nbf) is in the
	// future.
	ErrTokenNotActive = errors.New("token is not valid yet")

	// ErrUnexpectedSigningMethod denotes a token was signed with an unexpected
	// signing method.
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method")
)

// NewSigner creates a new JWT generating middleware, specifying key ID,
// signing string, signing method and the claims you would like it to contain.
// Tokens are signed with a Key ID header (kid) which is useful for determining
// the key to use for parsing. Particularly useful for clients.
func NewSigner[O interface{}](kid string, key []byte, method jwt.SigningMethod, claims jwt.Claims) endpoint.Middleware[O] {
	return func(next endpoint.Endpoint[O]) endpoint.Endpoint[O] {
		return func(ctx context.Context, request interface{}) (response O, err error) {
			token := jwt.NewWithClaims(method, claims)
			token.Header["kid"] = kid

			// Sign and get the complete encoded token as a string using the secret
			tokenString, err := token.SignedString(key)
			if err != nil {
				return response, err
			}
			ctx = context.WithValue(ctx, JWTContextKey, tokenString)

			return next(ctx, request)
		}
	}
}

// ClaimsFactory is a factory for jwt.Claims.
// Useful in NewParser middleware.
type ClaimsFactory func() jwt.Claims

// MapClaimsFactory is a ClaimsFactory that returns
// an empty jwt.MapClaims.
func MapClaimsFactory() jwt.Claims {
	return jwt.MapClaims{}
}

// StandardClaimsFactory is a ClaimsFactory that returns
// an empty jwt.StandardClaims.
func StandardClaimsFactory() jwt.Claims {
	return &jwt.RegisteredClaims{}
}

// NewParser creates a new JWT parsing middleware, specifying a
// jwt.Keyfunc interface, the signing method and the claims type to be used. NewParser
// adds the resulting claims to endpoint context or returns error on invalid token.
// Particularly useful for servers.
func NewParser[O interface{}](keyFunc jwt.Keyfunc, method jwt.SigningMethod, newClaims ClaimsFactory) endpoint.Middleware[O] {
	return func(next endpoint.Endpoint[O]) endpoint.Endpoint[O] {
		return func(ctx context.Context, request interface{}) (response O, err error) {
			// tokenString is stored in the context from the transport handlers.
			tokenString, ok := ctx.Value(JWTContextKey).(string)
			if !ok {
				return response, ErrTokenContextMissing
			}

			// Parse takes the token string and a function for looking up the
			// key. The latter is especially useful if you use multiple keys
			// for your application.  The standard is to use 'kid' in the head
			// of the token to identify which key to use, but the parsed token
			// (head and claims) is provided to the callback, providing
			// flexibility.
			token, err := jwt.ParseWithClaims(tokenString, newClaims(), func(token *jwt.Token) (interface{}, error) {
				// Don't forget to validate the alg is what you expect:
				if token.Method != method {
					return nil, ErrUnexpectedSigningMethod
				}

				return keyFunc(token)
			})
			if err != nil {
				if e, ok := err.(*jwt.ValidationError); ok {
					switch {
					case e.Errors&jwt.ValidationErrorMalformed != 0:
						// Token is malformed
						return response, ErrTokenMalformed
					case e.Errors&jwt.ValidationErrorExpired != 0:
						// Token is expired
						return response, ErrTokenExpired
					case e.Errors&jwt.ValidationErrorNotValidYet != 0:
						// Token is not active yet
						return response, ErrTokenNotActive
					case e.Inner != nil:
						// report e.Inner
						return response, e.Inner
					}
					// We have a ValidationError but have no specific Go kit error for it.
					// Fall through to return original error.
				}
				return response, err
			}

			if !token.Valid {
				return response, ErrTokenInvalid
			}

			ctx = context.WithValue(ctx, JWTClaimsContextKey, token.Claims)

			return next(ctx, request)
		}
	}
}
