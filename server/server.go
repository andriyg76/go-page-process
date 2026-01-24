package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	log "github.com/andriyg76/glog"

	"github.com/andriyg76/go-page-process/processor"
)

const defaultPort = 8080

// Options configures the embedded HTTP server.
type Options struct {
	Port   int
	Listen string
}

// Server exposes rendered output over HTTP.
type Server struct {
	proc       *processor.Processor
	opts       Options
	addr       string
	outputRoot string

	routesMu sync.RWMutex
	routes   map[string]string

	stateMu  sync.Mutex
	server   *http.Server
	listener net.Listener
}

// New creates a server bound to the processor output.
func New(proc *processor.Processor, opts Options) (*Server, error) {
	if proc == nil {
		return nil, fmt.Errorf("processor is nil")
	}
	if proc.Config.OutputPath == "" {
		return nil, fmt.Errorf("output path is empty")
	}

	opts = normalizeOptions(opts)
	addr, err := resolveListenAddress(opts)
	if err != nil {
		return nil, err
	}

	srv := &Server{
		proc:       proc,
		opts:       opts,
		addr:       addr,
		outputRoot: proc.Config.OutputPath,
	}
	if err := srv.Reload(); err != nil {
		return nil, err
	}
	return srv, nil
}

// Addr returns the configured address. If running, it returns the listener address.
func (s *Server) Addr() string {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.addr
}

// Start launches the server in a background goroutine.
func (s *Server) Start() error {
	if err := s.Reload(); err != nil {
		return err
	}

	s.stateMu.Lock()
	if s.server != nil {
		s.stateMu.Unlock()
		return fmt.Errorf("server is already running")
	}

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		s.stateMu.Unlock()
		return err
	}

	srv := &http.Server{
		Addr:    s.addr,
		Handler: s,
	}
	s.server = srv
	s.listener = listener
	s.stateMu.Unlock()

	go func() {
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("HTTP server error: %v", err)
		}
		s.clearState(srv)
	}()

	return nil
}

// Serve starts the server and blocks until it stops.
func (s *Server) Serve() error {
	if err := s.Reload(); err != nil {
		return err
	}

	s.stateMu.Lock()
	if s.server != nil {
		s.stateMu.Unlock()
		return fmt.Errorf("server is already running")
	}

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		s.stateMu.Unlock()
		return err
	}

	srv := &http.Server{
		Addr:    s.addr,
		Handler: s,
	}
	s.server = srv
	s.listener = listener
	s.stateMu.Unlock()

	err = srv.Serve(listener)
	s.clearState(srv)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	s.stateMu.Lock()
	srv := s.server
	s.stateMu.Unlock()
	if srv == nil {
		return nil
	}

	err := srv.Shutdown(ctx)
	s.clearState(srv)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Restart stops the server (if running) and starts it again.
func (s *Server) Restart(ctx context.Context) error {
	if err := s.Stop(ctx); err != nil {
		return err
	}
	return s.Start()
}

// Reload rebuilds route mappings from the data directory.
func (s *Server) Reload() error {
	routes, err := s.proc.OutputRoutes()
	if err != nil {
		return err
	}

	routeMap := make(map[string]string, len(routes))
	for _, route := range routes {
		routeMap[route.URLPath] = route.FilePath
	}

	s.routesMu.Lock()
	s.routes = routeMap
	s.routesMu.Unlock()
	return nil
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path
	if urlPath == "" {
		urlPath = "/"
	}

	if filePath, ok := s.routeFor(urlPath); ok {
		s.serveOutputFile(w, r, filePath)
		return
	}

	if urlPath != "/" && !strings.HasSuffix(urlPath, "/") {
		if s.hasRoute(urlPath + "/") {
			http.Redirect(w, r, urlPath+"/", http.StatusMovedPermanently)
			return
		}
	}

	if s.serveStaticFile(w, r, urlPath) {
		return
	}

	http.NotFound(w, r)
}

func normalizeOptions(opts Options) Options {
	if opts.Listen == "" && opts.Port == 0 {
		opts.Port = defaultPort
	}
	return opts
}

func resolveListenAddress(opts Options) (string, error) {
	if opts.Listen != "" {
		return opts.Listen, nil
	}
	if opts.Port <= 0 || opts.Port > 65535 {
		return "", fmt.Errorf("invalid port %d", opts.Port)
	}
	return fmt.Sprintf(":%d", opts.Port), nil
}

func (s *Server) routeFor(urlPath string) (string, bool) {
	s.routesMu.RLock()
	defer s.routesMu.RUnlock()
	filePath, ok := s.routes[urlPath]
	return filePath, ok
}

func (s *Server) hasRoute(urlPath string) bool {
	s.routesMu.RLock()
	defer s.routesMu.RUnlock()
	_, ok := s.routes[urlPath]
	return ok
}

func (s *Server) serveOutputFile(w http.ResponseWriter, r *http.Request, filePath string) {
	fullPath := filepath.Join(s.outputRoot, filepath.FromSlash(filePath))
	http.ServeFile(w, r, fullPath)
}

func (s *Server) serveStaticFile(w http.ResponseWriter, r *http.Request, urlPath string) bool {
	cleaned := path.Clean("/" + urlPath)
	rel := strings.TrimPrefix(cleaned, "/")
	fullPath := filepath.Join(s.outputRoot, filepath.FromSlash(rel))

	info, err := os.Stat(fullPath)
	if err != nil {
		return false
	}

	if info.IsDir() {
		indexPath := filepath.Join(fullPath, "index.html")
		if _, err := os.Stat(indexPath); err != nil {
			return false
		}
		http.ServeFile(w, r, indexPath)
		return true
	}

	http.ServeFile(w, r, fullPath)
	return true
}

func (s *Server) clearState(srv *http.Server) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.server != srv {
		return
	}
	s.server = nil
	if s.listener != nil {
		_ = s.listener.Close()
		s.listener = nil
	}
}
