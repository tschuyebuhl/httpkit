package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/coreos/go-oidc/v3/oidc/oidctest"
	"github.com/tschuyebuhl/httpkit/userctx"
)

func TestKeycloakRequiresAuthorization(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called without authorization")
	})

	auth := NewKeycloak(nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	auth.Handler(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestKeycloakSetsUserID(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	oidcServer := &oidctest.Server{
		PublicKeys: []oidctest.PublicKey{
			{
				PublicKey: priv.Public(),
				KeyID:     "test-key",
				Algorithm: oidc.RS256,
			},
		},
	}
	srv := httptest.NewServer(oidcServer)
	defer srv.Close()
	oidcServer.SetIssuer(srv.URL)

	provider, err := oidc.NewProvider(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	rawClaims := fmt.Sprintf(`{"iss":"%s","aud":"test","sub":"user-1","exp":%d}`,
		srv.URL, time.Now().Add(time.Hour).Unix(),
	)
	token := oidctest.SignIDToken(priv, "test-key", oidc.RS256, rawClaims)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userctx.UserIDFromContext(r.Context())
		if !ok {
			t.Fatal("expected user_id in context")
		}
		_, _ = w.Write([]byte(userID))
	})

	auth := NewKeycloak(provider)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	auth.Handler(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "user-1" {
		t.Fatalf("expected body user-1, got %q", rec.Body.String())
	}
}
