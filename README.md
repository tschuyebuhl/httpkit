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
	handler := httpx.Chain(baseMux, middleware.QueryParams, httpx.LoggerMiddleware())

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
