package diags

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/paragor/faraway-edge/pkg/log"
)

type HTTPServer struct {
	port  int
	ready atomic.Bool
}

func NewHTTPServer(port int) *HTTPServer {
	server := &HTTPServer{
		port: port,
	}
	server.ready.Store(false)
	return server
}

func (s *HTTPServer) SetReady(ready bool) {
	s.ready.Store(ready)
}

func (s *HTTPServer) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if s.ready.Load() {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("not ready"))
	}
}

func (s *HTTPServer) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if s.ready.Load() {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("not ready"))
	}
}

func (s *HTTPServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement actual metrics
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("# No metrics implemented yet\n"))
}

func (s *HTTPServer) Run(ctx context.Context) error {
	logger := log.FromContext(ctx)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)
	mux.HandleFunc("/metrics", s.handleMetrics)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpServer.Shutdown(shutdownCtx)
	}()

	logger.Info("diags HTTP server started", slog.Int("port", s.port))
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server failed: %w", err)
	}
	return nil
}
