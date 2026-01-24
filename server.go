package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	log "github.com/andriyg76/glog"

	"github.com/andriyg76/go-page-process/processor"
)

type serveOptions struct {
	Enabled bool
	Port    int
	Listen  string
}

func serveOutput(proc *processor.Processor, opts serveOptions) error {
	if proc == nil {
		return fmt.Errorf("processor is nil")
	}

	addr, err := resolveListenAddress(opts)
	if err != nil {
		return err
	}

	routes, err := proc.OutputRoutes()
	if err != nil {
		return err
	}

	routeMap := make(map[string]string, len(routes))
	for _, route := range routes {
		routeMap[route.URLPath] = route.FilePath
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		urlPath := r.URL.Path
		if urlPath == "" {
			urlPath = "/"
		}

		if filePath, ok := routeMap[urlPath]; ok {
			serveOutputFile(w, r, proc.Config.OutputPath, filePath)
			return
		}

		if urlPath != "/" && !strings.HasSuffix(urlPath, "/") {
			if _, ok := routeMap[urlPath+"/"]; ok {
				http.Redirect(w, r, urlPath+"/", http.StatusMovedPermanently)
				return
			}
		}

		if serveStaticFile(w, r, proc.Config.OutputPath, urlPath) {
			return
		}

		http.NotFound(w, r)
	})

	log.Info("Serving %s on %s\n", proc.Config.OutputPath, addr)
	return http.ListenAndServe(addr, handler)
}

func resolveListenAddress(opts serveOptions) (string, error) {
	if opts.Listen != "" {
		return opts.Listen, nil
	}
	if opts.Port < 0 || opts.Port > 65535 {
		return "", fmt.Errorf("invalid port %d", opts.Port)
	}
	return fmt.Sprintf(":%d", opts.Port), nil
}

func serveOutputFile(w http.ResponseWriter, r *http.Request, outputRoot, filePath string) {
	fullPath := filepath.Join(outputRoot, filepath.FromSlash(filePath))
	http.ServeFile(w, r, fullPath)
}

func serveStaticFile(w http.ResponseWriter, r *http.Request, outputRoot, urlPath string) bool {
	cleaned := path.Clean("/" + urlPath)
	rel := strings.TrimPrefix(cleaned, "/")
	fullPath := filepath.Join(outputRoot, filepath.FromSlash(rel))

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
