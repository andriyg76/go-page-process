package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// OutputRoute maps a request path to a generated file.
// FilePath is output-relative and uses '/' separators.
type OutputRoute struct {
	URLPath  string
	FilePath string
}

// OutputRoutes scans the output directory and returns URL routes to serve.
// It includes aliases for index.html files ("/" and "/section/").
func (p *Processor) OutputRoutes() ([]OutputRoute, error) {
	if p == nil {
		return nil, fmt.Errorf("processor is nil")
	}
	return collectOutputRoutes(p.Config.OutputPath)
}

func collectOutputRoutes(outputPath string) ([]OutputRoute, error) {
	if outputPath == "" {
		return nil, fmt.Errorf("output path is empty")
	}

	routes := make(map[string]OutputRoute)
	err := filepath.Walk(outputPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(outputPath, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		urlPath := "/" + rel
		addRoute(routes, urlPath, rel)

		if strings.HasSuffix(urlPath, "/index.html") {
			alias := strings.TrimSuffix(urlPath, "index.html")
			addRoute(routes, alias, rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	output := make([]OutputRoute, 0, len(routes))
	for _, route := range routes {
		output = append(output, route)
	}
	sort.Slice(output, func(i, j int) bool {
		return output[i].URLPath < output[j].URLPath
	})
	return output, nil
}

func addRoute(routes map[string]OutputRoute, urlPath, filePath string) {
	if urlPath == "" {
		return
	}
	if _, exists := routes[urlPath]; exists {
		return
	}
	routes[urlPath] = OutputRoute{URLPath: urlPath, FilePath: filePath}
}
