package respond

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestRevalidated(t *testing.T) {
	cover := []byte("first cover")
	app := fiber.New()
	app.Get("/artwork", func(c fiber.Ctx) error { return Revalidated(c, "image/jpeg", cover) })

	resp, err := app.Test(httptest.NewRequest("GET", "/artwork", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	etag := resp.Header.Get("ETag")
	if resp.StatusCode != 200 || string(body) != "first cover" || etag == "" || resp.Header.Get("Cache-Control") != "no-cache" {
		t.Fatalf("expected the cover with an ETag to revalidate, got %d %q, ETag %q, Cache-Control %q", resp.StatusCode, body, etag, resp.Header.Get("Cache-Control"))
	}

	// The ETag as sent, in a list, weakened by a proxy, or as "*"
	for _, ifNoneMatch := range []string{etag, `"other", ` + etag, "W/" + etag, "*"} {
		req := httptest.NewRequest("GET", "/artwork", nil)
		req.Header.Set("If-None-Match", ifNoneMatch)
		if resp, _ := app.Test(req); resp.StatusCode != 304 {
			t.Errorf("expected 304 for an unchanged cover with If-None-Match %s, got %d", ifNoneMatch, resp.StatusCode)
		}
	}

	// A replaced cover is sent again under the same URL
	cover = []byte("second cover")
	req := httptest.NewRequest("GET", "/artwork", nil)
	req.Header.Set("If-None-Match", etag)
	resp, _ = app.Test(req)
	body, _ = io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || string(body) != "second cover" {
		t.Errorf("expected the replaced cover, got %d %q", resp.StatusCode, body)
	}
}
