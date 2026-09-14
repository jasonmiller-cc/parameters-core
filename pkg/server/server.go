// Package server provides a production-ready HTTP server factory with graceful
// shutdown, TLS support, and pre-wired middleware for parameters services.
package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jasonmiller-cc/parameters-core/pkg/config"
	corelog "github.com/jasonmiller-cc/parameters-core/pkg/log"
)

// Server wraps net/http.Server with graceful shutdown.
type Server struct {
	http    *http.Server
	log     *corelog.Logger
	cfg     config.ServerConfig
}

// New creates a Server from config. Call Run to start.
func New(cfg config.ServerConfig, handler http.Handler, log *corelog.Logger) *Server {
	readTimeout := time.Duration(cfg.ReadTimeout) * time.Second
	if readTimeout == 0 {
		readTimeout = 15 * time.Second
	}
	writeTimeout := time.Duration(cfg.WriteTimeout) * time.Second
	if writeTimeout == 0 {
		writeTimeout = 60 * time.Second
	}

	s := &Server{
		log: log,
		cfg: cfg,
		http: &http.Server{
			Addr:         cfg.Addr(),
			Handler:      handler,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  120 * time.Second,
		},
	}

	if cfg.TLSCert != "" && cfg.TLSKey != "" {
		s.http.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	return s
}

// Run starts the HTTP server and blocks until SIGINT/SIGTERM is received,
// then performs a graceful shutdown with a configurable deadline.
func (s *Server) Run(ctx context.Context) error {
	shutdownTimeout := time.Duration(s.cfg.ShutdownTimeout) * time.Second
	if shutdownTimeout == 0 {
		shutdownTimeout = 30 * time.Second
	}

	ln, err := net.Listen("tcp", s.http.Addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.http.Addr, err)
	}

	s.log.Info("server started", "addr", s.http.Addr, "tls", s.cfg.TLSCert != "")

	errCh := make(chan error, 1)
	go func() {
		if s.cfg.TLSCert != "" && s.cfg.TLSKey != "" {
			errCh <- s.http.ServeTLS(ln, s.cfg.TLSCert, s.cfg.TLSKey)
		} else {
			errCh <- s.http.Serve(ln)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		s.log.Info("shutdown signal received", "signal", sig)
	case <-ctx.Done():
		s.log.Info("context cancelled, shutting down")
	}

	shutCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := s.http.Shutdown(shutCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	s.log.Info("server stopped")
	return nil
}
