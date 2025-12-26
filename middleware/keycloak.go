package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/tschuyebuhl/aids/userctx"
)

type KeycloakAuthenticator struct {
	handler     http.Handler
	provider    *oidc.Provider
	tokenMapper TokenMapper
}

type TokenMapper func(ctx context.Context, token *oidc.IDToken) (context.Context, error)

type KeycloakOption func(*KeycloakAuthenticator)

func WithTokenMapper(mapper TokenMapper) KeycloakOption {
	return func(a *KeycloakAuthenticator) {
		if mapper != nil {
			a.tokenMapper = mapper
		}
	}
}

func NewKeycloakAuthenticator(handler http.Handler, provider *oidc.Provider, opts ...KeycloakOption) *KeycloakAuthenticator {
	auth := &KeycloakAuthenticator{
		handler:     handler,
		provider:    provider,
		tokenMapper: defaultTokenMapper,
	}
	for _, opt := range opts {
		opt(auth)
	}
	if auth.tokenMapper == nil {
		auth.tokenMapper = defaultTokenMapper
	}
	return auth
}

func (a *KeycloakAuthenticator) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/api/") {
		a.handler.ServeHTTP(w, r)
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

	idToken, err := a.provider.Verifier(&oidc.Config{ClientID: "", SkipClientIDCheck: true}).Verify(r.Context(), tokenString)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error verifying token: %s", err), http.StatusUnauthorized)
		return
	}

	ctx := r.Context()
	if a.tokenMapper != nil {
		mappedCtx, err := a.tokenMapper(ctx, idToken)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error mapping token: %s", err), http.StatusUnauthorized)
			return
		}
		ctx = mappedCtx
	}

	a.handler.ServeHTTP(w, r.WithContext(ctx))
}

func defaultTokenMapper(ctx context.Context, token *oidc.IDToken) (context.Context, error) {
	return userctx.WithUserID(ctx, token.Subject), nil
}
