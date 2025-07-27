// Package nojs provides a framework for building web applications without JavaScript
package nojs

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Server represents a NoJS web server
type Server struct {
	mux         *http.ServeMux
	middlewares []Middleware
	config      ServerConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	StreamingEnabled  bool
	AutoRefreshPeriod time.Duration
}

// DefaultServerConfig returns sensible defaults
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
		StreamingEnabled:  true,
		AutoRefreshPeriod: 5 * time.Second,
	}
}

// NewServer creates a new NoJS server
func NewServer(config ...ServerConfig) *Server {
	cfg := DefaultServerConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return &Server{
		mux:    http.NewServeMux(),
		config: cfg,
	}
}

// Route registers a route handler
func (s *Server) Route(pattern string, handler Handler) {
	s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		ctx := &Context{
			Request:        r,
			ResponseWriter: w,
			server:         s,
		}

		// Apply middlewares
		finalHandler := handler
		for i := len(s.middlewares) - 1; i >= 0; i-- {
			finalHandler = s.middlewares[i](finalHandler)
		}

		// Execute handler
		if err := finalHandler(ctx); err != nil {
			s.handleError(ctx, err)
		}
	})
}

// Use adds middleware to the server
func (s *Server) Use(middleware Middleware) {
	s.middlewares = append(s.middlewares, middleware)
}

// Static serves static files from a directory
func (s *Server) Static(pattern string, dir string) {
	s.mux.Handle(pattern, http.StripPrefix(pattern, http.FileServer(http.Dir(dir))))
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	srv := &http.Server{
		Addr:           addr,
		Handler:        s.mux,
		ReadTimeout:    s.config.ReadTimeout,
		WriteTimeout:   s.config.WriteTimeout,
		IdleTimeout:    s.config.IdleTimeout,
		MaxHeaderBytes: s.config.MaxHeaderBytes,
	}

	fmt.Printf("NoJS server starting on %s\n", addr)
	return srv.ListenAndServe()
}

// StartWithContext starts the server with context for graceful shutdown
func (s *Server) StartWithContext(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:           addr,
		Handler:        s.mux,
		ReadTimeout:    s.config.ReadTimeout,
		WriteTimeout:   s.config.WriteTimeout,
		IdleTimeout:    s.config.IdleTimeout,
		MaxHeaderBytes: s.config.MaxHeaderBytes,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	fmt.Printf("NoJS server starting on %s\n", addr)
	return srv.ListenAndServe()
}

// handleError handles errors in a consistent way
func (s *Server) handleError(ctx *Context, err error) {
	if httpErr, ok := err.(*HTTPError); ok {
		http.Error(ctx.ResponseWriter, httpErr.Message, httpErr.Code)
	} else {
		http.Error(ctx.ResponseWriter, "Internal Server Error", http.StatusInternalServerError)
	}
}