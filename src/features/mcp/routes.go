package mcp

import (
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"slices"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/gofiber/fiber/v2"
	"github.com/mark3labs/mcp-go/server"
)

// maxRequestBody caps JSON-RPC request bodies handed to mcp-go. Fiber has
// already buffered the body under the app-wide limit (sized for file uploads),
// so this bounds JSON parsing work, not memory.
const maxRequestBody = 1 << 20 // 1 MiB

// RegisterRoutes mounts the MCP Streamable HTTP endpoint on the Fiber app at /mcp.
func RegisterRoutes(app *fiber.App, s *Service, cfg config.MCP) {
	version := os.Getenv("IMAGE_TAG")
	if version == "" {
		version = "dev"
	}

	s.readOnly = cfg.ReadOnly

	mcpServer := server.NewMCPServer("soulsolid", version)
	s.registerLibraryTools(mcpServer)
	s.registerMetadataTools(mcpServer)
	s.registerLyricsTools(mcpServer)
	s.registerJobTools(mcpServer)
	s.registerImportingTools(mcpServer)
	s.registerDownloadTools(mcpServer)
	s.registerPlaylistTools(mcpServer)
	s.registerConfigTools(mcpServer)
	s.registerMergeTools(mcpServer)

	// Stateless: every tool call is a plain request/response, no sessions or
	// SSE streams. Fiber buffers responses (CanStream() == false), so mcp-go
	// answers POSTs with a single application/json reply and rejects GET
	// (SSE listening) with 405, which is spec-compliant for stateless servers.
	httpServer := server.NewStreamableHTTPServer(mcpServer,
		server.WithEndpointPath("/mcp"),
		server.WithStateLess(true),
	)

	app.All("/mcp", guard(cfg), func(c *fiber.Ctx) error {
		handleMCP(c, httpServer)
		return nil
	})
}

// guard rejects requests that are too large or come from a disallowed browser
// origin. The endpoint has no authentication: anyone who can reach the port
// can use it, so expose it only on a trusted network.
func guard(cfg config.MCP) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if len(c.Body()) > maxRequestBody {
			return c.Status(fiber.StatusRequestEntityTooLarge).SendString("request body too large")
		}
		// MCP spec: servers must validate Origin to prevent DNS rebinding.
		// Same-origin checks don't help there (the rebound Host matches the
		// attacker's Origin), so only an explicit allowlist is accepted.
		if origin := c.Get(fiber.HeaderOrigin); origin != "" && !slices.Contains(cfg.AllowedOrigins, origin) {
			slog.Warn("MCP request rejected: origin not allowed", "origin", origin, "ip", c.IP())
			return c.Status(fiber.StatusForbidden).SendString("origin not allowed")
		}
		return c.Next()
	}
}

// handleMCP bridges a Fiber request into mcp-go's transport-agnostic Handle
// entry point (the integration path mcp-go provides for non-net/http frameworks).
func handleMCP(c *fiber.Ctx, srv *server.StreamableHTTPServer) {
	u, err := url.Parse(c.OriginalURL())
	if err != nil {
		u = &url.URL{Path: c.Path()}
	}

	header := make(http.Header)
	c.Request().Header.VisitAll(func(k, v []byte) {
		header.Add(string(k), string(v))
	})

	req := &server.HTTPRequest{
		Method:  c.Method(),
		URL:     u,
		Header:  header,
		Body:    c.Body(),
		Context: c.Context(), // *fasthttp.RequestCtx implements context.Context
	}
	srv.Handle(&fiberResponseWriter{c: c, header: make(http.Header)}, req)
}

// fiberResponseWriter adapts Fiber's buffered response to mcp-go's
// HTTPResponseWriter. It cannot stream, which mcp-go detects via CanStream.
type fiberResponseWriter struct {
	c           *fiber.Ctx
	header      http.Header
	wroteHeader bool
}

func (w *fiberResponseWriter) Header() http.Header { return w.header }

func (w *fiberResponseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	for k, vs := range w.header {
		for i, v := range vs {
			if i == 0 {
				w.c.Response().Header.Set(k, v)
			} else {
				w.c.Response().Header.Add(k, v)
			}
		}
	}
	w.c.Status(status)
}

func (w *fiberResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.c.Write(p)
}

func (w *fiberResponseWriter) Flush()          {}
func (w *fiberResponseWriter) CanStream() bool { return false }
