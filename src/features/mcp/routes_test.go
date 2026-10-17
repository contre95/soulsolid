package mcp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/gofiber/fiber/v2"
)

const toolsListBody = `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`

// newTestApp mounts the MCP endpoint on an in-memory Fiber app. Tool handlers
// are never invoked by these tests, so an empty Service is sufficient.
func newTestApp(cfg ...config.MCP) *fiber.App {
	app := fiber.New()
	var c config.MCP
	if len(cfg) > 0 {
		c = cfg[0]
	}
	RegisterRoutes(app, &Service{}, c)
	return app
}

func doMCP(t *testing.T, app *fiber.App, body string, headers map[string]string) *http.Response {
	t.Helper()
	req := httptest.NewRequest("POST", "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

func postMCP(t *testing.T, app *fiber.App, body string, headers ...map[string]string) (int, map[string]any) {
	t.Helper()
	var h map[string]string
	if len(headers) > 0 {
		h = headers[0]
	}
	resp := doMCP(t, app, body, h)
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

// listTools returns the registered tools keyed by name.
func listTools(t *testing.T, app *fiber.App) map[string]map[string]any {
	t.Helper()
	status, resp := postMCP(t, app, toolsListBody)
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
	byName := make(map[string]map[string]any, len(tools))
	for _, tool := range tools {
		if m, ok := tool.(map[string]any); ok {
			if name, ok := m["name"].(string); ok {
				byName[name] = m
			}
		}
	}
	return byName
}

func TestMCPToolsList(t *testing.T) {
	tools := listTools(t, newTestApp())
	for _, expected := range []string{"search_library", "get_track", "update_track_tags", "list_jobs", "download_album", "create_playlist", "get_config", "find_merge_groups", "merge_tracks"} {
		if tools[expected] == nil {
			t.Errorf("expected tool %q to be registered", expected)
		}
	}
}

func TestMCPToolAnnotations(t *testing.T) {
	tools := listTools(t, newTestApp())
	for name, tool := range tools {
		ann, _ := tool["annotations"].(map[string]any)
		if _, ok := ann["readOnlyHint"].(bool); !ok {
			t.Errorf("tool %q has no readOnlyHint annotation", name)
		}
	}
	for _, name := range []string{"delete_track", "merge_tracks", "update_settings"} {
		ann, _ := tools[name]["annotations"].(map[string]any)
		if ann["destructiveHint"] != true {
			t.Errorf("expected %q to be annotated destructive, got %v", name, ann)
		}
	}
}

func TestMCPReadOnlyMode(t *testing.T) {
	tools := listTools(t, newTestApp(config.MCP{ReadOnly: true}))
	for name, tool := range tools {
		ann, _ := tool["annotations"].(map[string]any)
		if ann["readOnlyHint"] != true {
			t.Errorf("read-only mode exposed non-read-only tool %q", name)
		}
	}
	if tools["search_library"] == nil {
		t.Error("expected read tools to remain available in read-only mode")
	}
}

func TestMCPToken(t *testing.T) {
	app := newTestApp(config.MCP{Token: "s3cret"})
	for _, tc := range []struct {
		name, auth string
		want       int
	}{
		{"missing", "", http.StatusUnauthorized},
		{"wrong", "Bearer nope", http.StatusUnauthorized},
		{"no scheme", "s3cret", http.StatusUnauthorized},
		{"valid", "Bearer s3cret", http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := doMCP(t, app, toolsListBody, map[string]string{"Authorization": tc.auth})
			resp.Body.Close()
			if resp.StatusCode != tc.want {
				t.Errorf("expected %d, got %d", tc.want, resp.StatusCode)
			}
		})
	}
}

func TestMCPOrigin(t *testing.T) {
	app := newTestApp(config.MCP{AllowedOrigins: []string{"https://agent.example"}})
	for _, tc := range []struct {
		name, origin string
		want         int
	}{
		{"no origin (non-browser client)", "", http.StatusOK},
		{"allowed", "https://agent.example", http.StatusOK},
		{"foreign site / DNS rebinding", "http://evil.example:3535", http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := doMCP(t, app, toolsListBody, map[string]string{"Origin": tc.origin})
			resp.Body.Close()
			if resp.StatusCode != tc.want {
				t.Errorf("expected %d, got %d", tc.want, resp.StatusCode)
			}
		})
	}
}

func TestMCPBodyLimit(t *testing.T) {
	resp := doMCP(t, newTestApp(), strings.Repeat(" ", maxRequestBody+1), nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", resp.StatusCode)
	}
}
