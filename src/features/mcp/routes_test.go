package mcp

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// newTestApp mounts the MCP endpoint on an in-memory Fiber app. Tool handlers
// are never invoked by these tests, so an empty Service is sufficient.
func newTestApp() *fiber.App {
	app := fiber.New()
	RegisterRoutes(app, &Service{})
	return app
}

func postMCP(t *testing.T, app *fiber.App, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest("POST", "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("response is not JSON (status %d): %s", resp.StatusCode, raw)
	}
	return resp.StatusCode, parsed
}

func TestMCPInitialize(t *testing.T) {
	app := newTestApp()
	status, resp := postMCP(t, app, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`)
	if status != 200 {
		t.Fatalf("expected status 200, got %d: %v", status, resp)
	}
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected initialize result, got: %v", resp)
	}
	serverInfo, ok := result["serverInfo"].(map[string]any)
	if !ok || serverInfo["name"] != "soulsolid" {
		t.Fatalf("expected serverInfo.name 'soulsolid', got: %v", result)
	}
}

func TestMCPToolsList(t *testing.T) {
	app := newTestApp()
	status, resp := postMCP(t, app, `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
	if status != 200 {
		t.Fatalf("expected status 200, got %d: %v", status, resp)
	}
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected tools/list result, got: %v", resp)
	}
	tools, ok := result["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatalf("expected a non-empty tool list, got: %v", result)
	}
	names := make(map[string]bool, len(tools))
	for _, tool := range tools {
		if m, ok := tool.(map[string]any); ok {
			if name, ok := m["name"].(string); ok {
				names[name] = true
			}
		}
	}
	for _, expected := range []string{"search_library", "get_track", "update_track_tags", "list_jobs", "download_album", "create_playlist", "get_config"} {
		if !names[expected] {
			t.Errorf("expected tool %q to be registered, got tools: %v", expected, names)
		}
	}
}
