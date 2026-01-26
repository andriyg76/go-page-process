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

// OutputRoutes scans the data directory and returns URL routes to serve.
// Routes are derived from _page.output and require _page.template.
// It includes aliases for index.html files ("/" and "/section/").
func (p *Processor) OutputRoutes() ([]OutputRoute, error) {
	if p == nil {
		return nil, fmt.Errorf("processor is nil")
	}
	return collectOutputRoutesFromData(p, p.Config.DataPath)
}

func collectOutputRoutesFromData(p *Processor, dataPath string) ([]OutputRoute, error) {
	if dataPath == "" {
		return nil, fmt.Errorf("data path is empty")
	}

	routes := make(map[string]OutputRoute)
	err := filepath.Walk(dataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		data, err := p.loadFile(path)
		if err != nil {
			return err
		}
		if data == nil {
			return nil
		}

		page, ok := data["_page"].(map[string]interface{})
		if !ok {
			return nil
		}

		output, ok := pageOutputPath(path, page)
		if !ok {
			return nil
		}

		urlPath := "/" + output
		addRoute(routes, urlPath, output)

		if strings.HasSuffix(urlPath, "/index.html") {
			alias := strings.TrimSuffix(urlPath, "index.html")
			addRoute(routes, alias, output)
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

func pageOutputPath(filePath string, page map[string]interface{}) (string, bool) {
	templateName, ok := page["template"].(string)
	if !ok || templateName == "" {
		return "", false
	}

	output, ok := page["output"].(string)
	if !ok || output == "" {
		output = filepath.Base(filePath)
	}

	output = filepath.ToSlash(output)
	if strings.HasPrefix(output, "/") {
		output = output[1:]
	}
	if output == "" {
		return "", false
	}

	return output, true
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
