// Package middleware provides 3rd party plugins to use with http.Handler types
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/tschuyebuhl/httpkit/userctx"
)

type Keycloak struct {
	verifier    *oidc.IDTokenVerifier
	tokenMapper TokenMapper
}

type TokenMapper func(ctx context.Context, token *oidc.IDToken) (context.Context, error)

type KeycloakOption func(*Keycloak)

func WithTokenMapper(mapper TokenMapper) KeycloakOption {
	return func(a *Keycloak) {
		if mapper != nil {
			a.tokenMapper = mapper
		}
	}
}

func NewKeycloak(provider *oidc.Provider, opts ...KeycloakOption) *Keycloak {
	cfg := &Keycloak{
		tokenMapper: defaultTokenMapper,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	var verifier *oidc.IDTokenVerifier
	if provider != nil {
		verifier = provider.Verifier(&oidc.Config{ClientID: "", SkipClientIDCheck: true})
	}

	return &Keycloak{
		verifier:    verifier,
		tokenMapper: cfg.tokenMapper,
	}
}

func (k *Keycloak) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		k.serve(w, r, next)
	})
}

func (k *Keycloak) Middleware() func(http.Handler) http.Handler {
	return k.Handler
}

func KeycloakMiddleware(provider *oidc.Provider, opts ...KeycloakOption) *Keycloak {
	return NewKeycloak(provider, opts...)
}

func (k *Keycloak) serve(w http.ResponseWriter, r *http.Request, next http.Handler) {
	if k.verifier == nil {
		http.Error(w, "OIDC provider is required", http.StatusUnauthorized)
		return
	}
	if next == nil {
		http.Error(w, "Next handler is required", http.StatusInternalServerError)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authorization header is required", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
		return
	}

	idToken, err := k.verifier.Verify(r.Context(), tokenString)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error verifying token: %s", err), http.StatusUnauthorized)
		return
	}

	ctx := r.Context()
	if k.tokenMapper != nil {
		mappedCtx, err := k.tokenMapper(ctx, idToken)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error mapping token: %s", err), http.StatusUnauthorized)
			return
		}
		ctx = mappedCtx
	}

	next.ServeHTTP(w, r.WithContext(ctx))
}

func defaultTokenMapper(ctx context.Context, token *oidc.IDToken) (context.Context, error) {
	return userctx.WithUserID(ctx, token.Subject), nil
}
