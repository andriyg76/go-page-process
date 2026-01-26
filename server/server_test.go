package server

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/andriyg76/go-page-process/processor"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestServerStartReloadAndStop(t *testing.T) {
	root := t.TempDir()
	config := processor.DefaultConfig()
	config.RootPath = root

	proc := processor.New(config)
	dataDir := filepath.Join(root, config.DataPath)
	outputDir := filepath.Join(root, config.OutputPath)

	writeFile(t, filepath.Join(dataDir, "index.json"), `{"_page":{"template":"page","output":"index.html"}}`)
	writeFile(t, filepath.Join(outputDir, "index.html"), "home")

	srv, err := New(proc, Options{Listen: "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := srv.Stop(ctx); err != nil {
			t.Fatalf("stop: %v", err)
		}
	})

	client := &http.Client{}
	resp, body := getWithRetry(t, client, "http://"+srv.Addr()+"/")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if strings.TrimSpace(string(body)) != "home" {
		t.Fatalf("unexpected body: %q", strings.TrimSpace(string(body)))
	}

	writeFile(t, filepath.Join(dataDir, "docs.json"), `{"_page":{"template":"page","output":"docs/index.html"}}`)
	writeFile(t, filepath.Join(outputDir, "docs", "index.html"), "docs")
	if err := srv.Reload(); err != nil {
		t.Fatalf("reload: %v", err)
	}

	resp, body = getWithRetry(t, client, "http://"+srv.Addr()+"/docs/")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if strings.TrimSpace(string(body)) != "docs" {
		t.Fatalf("unexpected body: %q", strings.TrimSpace(string(body)))
	}

	redirectClient := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, _ = getWithRetry(t, redirectClient, "http://"+srv.Addr()+"/docs")
	if resp.StatusCode != http.StatusMovedPermanently {
		t.Fatalf("expected redirect, got %d", resp.StatusCode)
	}
	if location := resp.Header.Get("Location"); location != "/docs/" {
		t.Fatalf("unexpected redirect location: %q", location)
	}
}

func getWithRetry(t *testing.T, client *http.Client, url string) (*http.Response, []byte) {
	t.Helper()
	var lastErr error
	for i := 0; i < 10; i++ {
		resp, err := client.Get(url)
		if err == nil {
			defer resp.Body.Close()
			body, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				t.Fatalf("read body: %v", readErr)
			}
			return resp, body
		}
		lastErr = err
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("request failed: %v", lastErr)
	return nil, nil
}
