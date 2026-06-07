package respond

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Section renders the section partial for HTMX requests or the full main layout otherwise.
// Assumes templates follow the "sections/<section>" naming convention.
// data must not be nil — a nil map causes a panic when "Section" is injected for non-HTMX requests.
func Section(c *fiber.Ctx, section string, data fiber.Map) error {
	if c.Get("HX-Request") != "true" {
		data["Section"] = section
		return c.Render("main", data)
	}
	return c.Render("sections/"+section, data)
}

// ToastErr responds with an error toast for HTMX requests or a JSON error body otherwise.
func ToastErr(c *fiber.Ctx, status int, msg string) error {
	if c.Get("HX-Request") == "true" {
		return c.Status(status).Render("toast/toastErr", fiber.Map{"Msg": msg})
	}
	return c.Status(status).JSON(fiber.Map{"error": msg})
}

// ToastOk responds with a success toast for HTMX requests or a JSON message otherwise.
func ToastOk(c *fiber.Ctx, msg string) error {
	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{"Msg": msg})
	}
	return c.JSON(fiber.Map{"message": msg})
}

// ToastJob responds with a success toast for HTMX requests or a 202 JSON body with the job_id otherwise.
func ToastJob(c *fiber.Ctx, jobID string, msg string) error {
	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{"Msg": msg})
	}
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"job_id": jobID})
}

// Partial renders the given template for HTMX requests or returns the same data as JSON otherwise.
func Partial(c *fiber.Ctx, template string, data fiber.Map) error {
	if c.Get("HX-Request") == "true" {
		return c.Render(template, data)
	}
	return c.JSON(data)
}

// Text sends a plain string for HTMX requests or {"key": key, "value": value} JSON otherwise.
// An optional htmxText overrides the formatted string sent to HTMX; without it, value is stringified.
func Text(c *fiber.Ctx, key string, value any, htmxText ...string) error {
	if c.Get("HX-Request") == "true" {
		if len(htmxText) > 0 {
			return c.SendString(htmxText[0])
		}
		return c.SendString(fmt.Sprint(value))
	}
	return c.JSON(fiber.Map{"key": key, "value": value})
}

// Resource serves binary or file content by default, or returns {"type": mimeType, "url": url} JSON
// when the client sends Accept: application/json.
// Unlike other helpers, this uses Accept-header negotiation rather than HX-Request, because browser
// resource tags (<img>, <a href>) never send HX-Request but still need the binary response.
func Resource(c *fiber.Ctx, mimeType, url string, serve func() error) error {
	if strings.Contains(c.Get("Accept"), "application/json") {
		return c.JSON(fiber.Map{"type": mimeType, "url": url})
	}
	return serve()
}

// HTMX renders the given template only for HTMX requests.
// Non-HTMX clients receive 406 Not Acceptable with a JSON error body.
//
// Use this for endpoints that are intrinsically tied to the HTMX interaction
// model and have no meaningful alternative representation — for example,
// endpoints that return raw HTML fragments whose rendering logic depends on
// in-flight HTMX state (polling, event triggers, multi-step UI flows).
// Do NOT use it just because a JSON response is inconvenient; prefer Partial
// if the data can stand alone as JSON.
func HTMX(c *fiber.Ctx, template string, data fiber.Map) error {
	if c.Get("HX-Request") != "true" {
		return c.Status(fiber.StatusNotAcceptable).JSON(fiber.Map{
			"error": "this endpoint is only available via HTMX",
		})
	}
	return c.Render(template, data)
}
