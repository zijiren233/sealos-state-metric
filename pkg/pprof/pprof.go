// Package pprof provides pprof server for profiling
package pprof

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	//nolint:gosec
	_ "net/http/pprof"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var pprofMux *http.ServeMux

func init() {
	pprofMux = http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()
}

// Server manages the pprof HTTP server
type Server struct {
	port     int
	server   *http.Server
	listener net.Listener
	//nolint:containedctx // Context stored for server lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	mu     sync.Mutex
}

// NewServer creates a new pprof server
func NewServer(port int) *Server {
	return &Server{
		port: port,
	}
}

// Start starts the pprof server
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server != nil {
		return errors.New("pprof server already running")
	}

	addr := fmt.Sprintf("127.0.0.1:%d", s.port)

	//nolint:noctx
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to create pprof listener on %s: %w", addr, err)
	}

	s.listener = listener
	s.server = &http.Server{
		Addr:              addr,
		Handler:           pprofMux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.done = make(chan struct{})

	log.WithField("address", addr).Info("Starting pprof server")

	go func() {
		defer close(s.done)

		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Error("Pprof server error")
		}
	}()

	return nil
}

// Stop stops the pprof server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server == nil {
		return nil
	}

	log.Info("Stopping pprof server")

	// Cancel context
	if s.cancel != nil {
		s.cancel()
	}

	// Shutdown server gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		log.WithError(err).Warn("Failed to shutdown pprof server gracefully, forcing close")
		// Force close listener
		if s.listener != nil {
			s.listener.Close()
		}
	}

	// Wait for server goroutine to exit
	if s.done != nil {
		<-s.done
	}

	// Clean up
	s.server = nil
	s.listener = nil
	s.ctx = nil
	s.cancel = nil
	s.done = nil

	log.Info("Pprof server stopped")

	return nil
}

// Reload reloads the pprof server with new port
func (s *Server) Reload(ctx context.Context, port int) error {
	// If port hasn't changed and server is running, do nothing
	s.mu.Lock()
	portChanged := s.port != port
	wasRunning := s.server != nil
	s.mu.Unlock()

	if !portChanged && wasRunning {
		log.Debug("Pprof server port unchanged, skipping reload")
		return nil
	}

	// Stop existing server
	if err := s.Stop(); err != nil {
		return fmt.Errorf("failed to stop pprof server: %w", err)
	}

	// Update port
	s.port = port

	// Start new server
	if err := s.Start(ctx); err != nil {
		return fmt.Errorf("failed to start pprof server: %w", err)
	}

	log.Info("Pprof server reloaded successfully")

	return nil
}
