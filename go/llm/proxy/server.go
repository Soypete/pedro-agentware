package proxy

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
	config  *ProxyConfig
	mux     *http.ServeMux
	server  *http.Server
	handler *Handler
}

func NewServer(cfg *ProxyConfig, h *Handler) *Server {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	mux := http.NewServeMux()
	s := &Server{
		config:  cfg,
		mux:     mux,
		handler: h,
	}

	s.registerRoutes()

	s.server = &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      s.mux,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("POST /v1/chat/completions", s.handler.handleChatCompletions)
	s.mux.HandleFunc("GET /v1/models", s.handler.handleModels)
	s.mux.HandleFunc("GET /health", s.handler.handleHealth)
}

func (s *Server) Start() error {
	log.Printf("Starting proxy server on %s", s.config.ListenAddr)
	log.Printf("Backend: %s (%s)", s.config.BackendURL, s.config.BackendType)

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) StartAsync() error {
	go func() {
		if err := s.Start(); err != nil {
			log.Printf("Proxy server error: %v", err)
		}
	}()
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) ListenAddr() string {
	return s.config.ListenAddr
}

func WaitForSignal() os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	return <-sigChan
}
