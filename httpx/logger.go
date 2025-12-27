package httpx

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/gofrs/uuid/v5"
)

type ctxKeyRequestID struct{}

type PanicHandler func(w http.ResponseWriter, r *http.Request, recovered any, stack []byte)

type LoggerOption func(*Logger)

func WithLogger(logger *slog.Logger) LoggerOption {
	return func(l *Logger) {
		if logger != nil {
			l.logger = logger
		}
	}
}

func WithPanicHandler(handler PanicHandler) LoggerOption {
	return func(l *Logger) {
		if handler != nil {
			l.panicHandler = handler
		}
	}
}

func LoggerMiddleware(opts ...LoggerOption) Middleware {
	return func(next http.Handler) http.Handler {
		return NewLogger(next, opts...)
	}
}

type Logger struct {
	handler      http.Handler
	logger       *slog.Logger
	panicHandler PanicHandler
}

func NewLogger(handler http.Handler, opts ...LoggerOption) *Logger {
	l := &Logger{
		handler: handler,
		logger:  slog.Default(),
	}
	l.panicHandler = defaultPanicHandler(l.logger)
	for _, opt := range opts {
		opt(l)
	}
	if l.panicHandler == nil {
		l.panicHandler = defaultPanicHandler(l.logger)
	}
	return l
}

func (l *Logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	i := &Interceptor{ResponseWriter: w}
	start := time.Now()
	id := uuid.Must(uuid.NewV4())
	l.logger.Info("handling http request", "method", r.Method, "path", r.URL.Path, "id", id.String())
	ctx := context.WithValue(r.Context(), ctxKeyRequestID{}, id)
	r = r.WithContext(ctx)

	defer func() {
		if rec := recover(); rec != nil {
			stack := debug.Stack()
			l.panicHandler(i, r, rec, stack)
		}
		status := i.Status()
		l.logger.Info("handled http request", "method", r.Method, "path", r.URL.Path, "time", time.Since(start), "response code", status, "id", id.String())
	}()

	l.handler.ServeHTTP(i, r)
}

func defaultPanicHandler(logger *slog.Logger) PanicHandler {
	return func(w http.ResponseWriter, r *http.Request, recovered any, stack []byte) {
		fmt.Fprintf(os.Stderr, "panic serving request %s %s\n%s\n", r.Method, r.URL.Path, stack)
		logger.Error("panic serving request", "method", r.Method, "path", r.URL.Path, "error", recovered)
		if iw, ok := w.(*Interceptor); ok {
			if !iw.HasWritten() {
				http.Error(iw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

type Interceptor struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (l *Interceptor) WriteHeader(statusCode int) {
	if l.wroteHeader {
		l.ResponseWriter.WriteHeader(statusCode)
		return
	}
	l.statusCode = statusCode
	l.wroteHeader = true
	l.ResponseWriter.WriteHeader(statusCode)
}

func (l *Interceptor) Write(p []byte) (int, error) {
	if !l.wroteHeader {
		l.WriteHeader(http.StatusOK)
	}
	return l.ResponseWriter.Write(p)
}

func (l *Interceptor) Status() int {
	if l.statusCode == 0 {
		return http.StatusOK
	}
	return l.statusCode
}

func (l *Interceptor) HasWritten() bool {
	return l.wroteHeader
}

func RequestID(ctx context.Context) (uuid.UUID, bool) {
	v := ctx.Value(ctxKeyRequestID{})
	if v == nil {
		return uuid.UUID{}, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

func RequestIDString(ctx context.Context) (string, bool) {
	id, ok := RequestID(ctx)
	if !ok {
		return "", false
	}
	return id.String(), true
}
