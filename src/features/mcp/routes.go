package mcp

import (
	"net/http"
	"net/url"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterRoutes mounts the MCP Streamable HTTP endpoint on the Fiber app at /mcp.
func RegisterRoutes(app *fiber.App, s *Service) {
	version := os.Getenv("IMAGE_TAG")
	if version == "" {
		version = "dev"
	}

	mcpServer := server.NewMCPServer("soulsolid", version)
	s.registerLibraryTools(mcpServer)
	s.registerMetadataTools(mcpServer)
	s.registerLyricsTools(mcpServer)
	s.registerJobTools(mcpServer)
	s.registerImportingTools(mcpServer)
	s.registerDownloadTools(mcpServer)
	s.registerPlaylistTools(mcpServer)
	s.registerConfigTools(mcpServer)

	// Stateless: every tool call is a plain request/response, no sessions or
	// SSE streams. Fiber buffers responses (CanStream() == false), so mcp-go
	// answers POSTs with a single application/json reply and rejects GET
	// (SSE listening) with 405, which is spec-compliant for stateless servers.
	httpServer := server.NewStreamableHTTPServer(mcpServer,
		server.WithEndpointPath("/mcp"),
		server.WithStateLess(true),
	)

	app.All("/mcp", func(c *fiber.Ctx) error {
		handleMCP(c, httpServer)
		return nil
	})
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
