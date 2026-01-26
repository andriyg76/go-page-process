package processor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestProcessRendersOutput(t *testing.T) {
	root := t.TempDir()
	config := DefaultConfig()
	config.RootPath = root

	templatePath := filepath.Join(root, config.TemplatesPath, "page.hbs")
	writeFile(t, templatePath, "Hello {{title}} from {{_shared.site.name}}")

	sharedPath := filepath.Join(root, config.SharedPath, "site.json")
	writeFile(t, sharedPath, `{"name":"Docs"}`)

	dataPath := filepath.Join(root, config.DataPath, "index.json")
	writeFile(t, dataPath, `{"_page":{"template":"page","output":"index.html"},"title":"Welcome"}`)

	proc := New(config)
	if err := proc.Process(); err != nil {
		t.Fatalf("process: %v", err)
	}

	outputPath := filepath.Join(root, config.OutputPath, "index.html")
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	got := strings.TrimSpace(string(content))
	want := "Hello Welcome from Docs"
	if got != want {
		t.Fatalf("unexpected output: got %q want %q", got, want)
	}
}

func TestProcessReturnsErrorOnMissingTemplate(t *testing.T) {
	root := t.TempDir()
	config := DefaultConfig()
	config.RootPath = root

	templateDir := filepath.Join(root, config.TemplatesPath)
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}

	dataPath := filepath.Join(root, config.DataPath, "index.json")
	writeFile(t, dataPath, `{"_page":{"template":"missing","output":"index.html"},"title":"Welcome"}`)

	proc := New(config)
	err := proc.Process()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "template missing is not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOutputRoutes(t *testing.T) {
	root := t.TempDir()
	config := DefaultConfig()
	config.RootPath = root

	proc := New(config)
	dataDir := filepath.Join(root, config.DataPath)

	writeFile(t, filepath.Join(dataDir, "index.json"), `{"_page":{"template":"page","output":"index.html"}}`)
	writeFile(t, filepath.Join(dataDir, "assets", "app.json"), `{"_page":{"template":"page","output":"assets/app.js"}}`)
	writeFile(t, filepath.Join(dataDir, "blog", "index.json"), `{"_page":{"template":"page","output":"blog/index.html"}}`)
	writeFile(t, filepath.Join(dataDir, "blog", "post.json"), `{"_page":{"template":"page","output":"blog/post.html"}}`)
	writeFile(t, filepath.Join(dataDir, "robots.json"), `{"_page":{"template":"page","output":"robots.txt"}}`)
	writeFile(t, filepath.Join(dataDir, "draft.json"), `{"_page":{"template":"page"}}`)
	writeFile(t, filepath.Join(dataDir, "docs.json"), `{"_page":{"template":"page","output":"/docs/index.html"}}`)
	writeFile(t, filepath.Join(dataDir, "skip.json"), `{"_page":{"output":"skip.html"}}`)

	routes, err := proc.OutputRoutes()
	if err != nil {
		t.Fatalf("output routes: %v", err)
	}

	got := make(map[string]string, len(routes))
	for _, route := range routes {
		got[route.URLPath] = route.FilePath
	}
	if len(routes) != len(got) {
		t.Fatalf("expected unique routes, got %d entries with %d unique", len(routes), len(got))
	}

	want := map[string]string{
		"/":                "index.html",
		"/index.html":      "index.html",
		"/assets/app.js":   "assets/app.js",
		"/blog/":           "blog/index.html",
		"/blog/index.html": "blog/index.html",
		"/blog/post.html":  "blog/post.html",
		"/docs/":           "docs/index.html",
		"/docs/index.html": "docs/index.html",
		"/draft.json":      "draft.json",
		"/robots.txt":      "robots.txt",
	}

	if len(got) != len(want) {
		t.Fatalf("unexpected route count: got %d want %d", len(got), len(want))
	}

	for urlPath, filePath := range want {
		if got[urlPath] != filePath {
			t.Fatalf("unexpected route %s: got %q want %q", urlPath, got[urlPath], filePath)
		}
	}
}
