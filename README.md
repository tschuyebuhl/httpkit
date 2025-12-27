# httpkit

Small helpers for building Go HTTP APIs: request logging, SPA fallbacks, Keycloak auth, query param parsing, and a few data/query utilities.

## Install

```sh
go get github.com/tschuyebuhl/httpkit
```

## HTTP middleware

Query params parsing with filters, sorting, and pagination:

```go
mux := http.NewServeMux()
handler := middleware.QueryParams(mux)

func list(w http.ResponseWriter, r *http.Request) {
    params := middleware.QueryParamsFromContext(r.Context())
    // params.Filter, params.Sort, params.Pagination
    _ = params
}
```

Query params URL example:

```
GET /api/habits?habit_id_exact="f86b053f-94ce-4c6f-b13a-e1208979218a"
```

Keycloak auth as middleware:

```go
auth := middleware.NewKeycloak(provider)
secured := auth.Handler(handler)
// or: httpx.Chain(handler, auth.Middleware())
```

Custom token mapping with extra JWT claims:

```go
type userEmailKey struct{}
type userRolesKey struct{}

var UserEmailKey = userEmailKey{}
var UserRolesKey = userRolesKey{}

auth := middleware.NewKeycloak(provider, middleware.WithTokenMapper(
    func(ctx context.Context, token *oidc.IDToken) (context.Context, error) {
        var claims struct {
            Email       string `json:"email"`
            RealmAccess struct {
                Roles []string `json:"roles"`
            } `json:"realm_access"`
        }

        if err := token.Claims(&claims); err != nil {
            return ctx, err
        }

        ctx = userctx.WithUserID(ctx, token.Subject)
        ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
        ctx = context.WithValue(ctx, UserRolesKey, claims.RealmAccess.Roles)
        return ctx, nil
    },
))
```

Per-route middleware:

```go
routes := []httpx.Route{
    {Pattern: "GET /health", Handler: health},
    {Pattern: "GET /health/secure", Handler: health, Use: []httpx.Middleware{auth.Middleware()}},
}
```

Route groups:

```go
api := httpx.Use(habits, auth.Middleware())
httpx.Register(mux, api)
```

## HTTP helpers

Request logging with panic recovery:

```go
logged := httpx.NewLogger(handler)
```

This suffices for basic logging. This will also inject a request ID into the context. One can extract it like so:
```go
package config

import (
	"context"
	"log/slog"
	"os"

	"github.com/tschuyebuhl/httpkit/httpx"
)

type RequestIDHandler struct {
	next slog.Handler
}

func (h RequestIDHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return h.next.Enabled(ctx, lvl)
}

func (h RequestIDHandler) Handle(ctx context.Context, r slog.Record) error {
	if rid, ok := httpx.RequestIDString(ctx); ok {
		r.AddAttrs(slog.String("request_id", rid))
	}
	return h.next.Handle(ctx, r)
}

func (h RequestIDHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return RequestIDHandler{next: h.next.WithAttrs(attrs)}
}

func (h RequestIDHandler) WithGroup(name string) slog.Handler {
	return RequestIDHandler{next: h.next.WithGroup(name)}
}

func LoggerSetup(cfg Log) {
	var base slog.Handler
	var logLevel slog.Level
	switch cfg.Level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		slog.Error("unknown log level", "level", cfg.Level)
		logLevel = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	}
	switch cfg.OutputFormat {
	case "json":
		base = slog.NewJSONHandler(os.Stdout, opts)
	case "text":
		base = slog.NewTextHandler(os.Stdout, opts)
	}
	logger := slog.New(RequestIDHandler{next: base})
	slog.SetDefault(logger)
}
```

SPA fallback for embedded or static file servers:

```go
fs := http.FS(embedded)
fileServer := http.FileServer(fs)
serveIndex := httpx.ServeFileContents("index.html", fs)

mux.Handle("/", httpx.Intercept404(fileServer, serveIndex))
```

## Data/query helpers

Slugify and query helpers:

```go
slug := data.Slugify("Daily Focus")
_ = slug
```

Apply user scoping in bob queries:

```go
habit, err := models.Habits.Query(models.SelectWhere.Habits.Code.EQ(domainModel.HabitCode), query.UserIDModifier(ctx),
		sm.Columns(dbinfo.Habits.Columns.ID.Name)).One(ctx, p.db)
```

## Full server wiring example

```go
var (
    //go:embed frontend/dist
    embeddedFS embed.FS
)

func NewHTTPServer(appConfig config.App, httpConfig config.Web, auth httpx.Middleware,
	habits *routers.HabitRouter, groups *routers.HabitGroupRouter,
	logs *routers.HabitLogRouter, streaks *routers.HabitStreakRouter,
	attributes *routers.HabitAttributeDefinitionRouter,
	health *routers.HealthRouter) *HTTPServer {
	slog.Info("creating new server instance", "listening addr", httpConfig.Addr)

	baseMux := http.NewServeMux()
	if appConfig.DevMode {
		baseMux.Handle("/", httpx.DevProxy(appConfig.FrontendAddress))
	} else {
		httpx.RunEmbeddedApp("/", embeddedFS, baseMux)
	}
	httpx.Register(baseMux,
		httpx.Use(habits, auth),
		httpx.Use(groups, auth),
		httpx.Use(logs, auth),
		httpx.Use(streaks, auth),
		httpx.Use(attributes, auth),
		health,
	)
	logger := httpx.NewLogger(baseMux)
	handler := httpx.Chain(logger, middleware.QueryParams)

	return &HTTPServer{
		Server: &http.Server{
			Addr:         httpConfig.Addr,
			Handler:      handler,
			TLSConfig:    nil,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 20 * time.Second,
		},
	}
}

```

## Tests

```sh
go test ./...
```

## License
This project is licensed under the [MIT License](./LICENSE)
