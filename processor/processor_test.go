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
