package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
)

// NewMux returns an HTTP mux with shared diagnostics endpoints.
func NewMux(serviceName string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	mux.HandleFunc("/metrics", metricsHandler(serviceName))
	return mux
}

// Run starts the HTTP server and blocks until the context is canceled.
func Run(ctx context.Context, logger zerolog.Logger, port int, handler http.Handler, shutdownTimeout time.Duration) error {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           withObservability(handler, logger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info().Int("port", port).Msg("starting HTTP server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		logger.Info().Msg("shutting down HTTP server")
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func withObservability(next http.Handler, logger zerolog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := strings.TrimSpace(r.Header.Get("X-Request-Id"))
		if requestID == "" {
			requestID = newRequestID()
		}
		correlationID := strings.TrimSpace(r.Header.Get("X-Correlation-Id"))
		w.Header().Set("X-Request-Id", requestID)

		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		incRequestCounter(r.Method, r.URL.Path, rw.status)
		evt := logger.Info().
			Str("request_id", requestID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", rw.status).
			Dur("duration", time.Since(start))
		if correlationID != "" {
			evt = evt.Str("correlation_id", correlationID)
		}
		evt.Msg("http request")
	})
}

func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(b)
}

type metricsKey struct {
	method string
	path   string
	status int
}

var (
	startedAt     = time.Now()
	requestTotals sync.Map
	totalRequests atomic.Int64
)

func incRequestCounter(method, path string, status int) {
	key := metricsKey{method: method, path: path, status: status}
	counter, _ := requestTotals.LoadOrStore(key, &atomic.Int64{})
	counter.(*atomic.Int64).Add(1)
	totalRequests.Add(1)
}

func metricsHandler(serviceName string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.WriteHeader(http.StatusOK)

		_, _ = fmt.Fprintf(w, "# HELP pcgb_http_requests_total Total HTTP requests handled.\n")
		_, _ = fmt.Fprintf(w, "# TYPE pcgb_http_requests_total counter\n")
		requestTotals.Range(func(k, v any) bool {
			key := k.(metricsKey)
			val := v.(*atomic.Int64).Load()
			_, _ = fmt.Fprintf(w,
				"pcgb_http_requests_total{service=%q,method=%q,path=%q,status=%q} %d\n",
				serviceName,
				key.method,
				key.path,
				strconv.Itoa(key.status),
				val,
			)
			return true
		})
		_, _ = fmt.Fprintf(w, "# HELP pcgb_http_requests_all_total Total HTTP requests across all paths.\n")
		_, _ = fmt.Fprintf(w, "# TYPE pcgb_http_requests_all_total counter\n")
		_, _ = fmt.Fprintf(w, "pcgb_http_requests_all_total{service=%q} %d\n", serviceName, totalRequests.Load())
		_, _ = fmt.Fprintf(w, "# HELP pcgb_process_uptime_seconds Process uptime in seconds.\n")
		_, _ = fmt.Fprintf(w, "# TYPE pcgb_process_uptime_seconds gauge\n")
		_, _ = fmt.Fprintf(w, "pcgb_process_uptime_seconds{service=%q} %.0f\n", serviceName, time.Since(startedAt).Seconds())
	}
}
