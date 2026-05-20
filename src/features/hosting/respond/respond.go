package respond

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// Section renders the section partial for HTMX requests or the full main layout otherwise.
// Assumes templates follow the "sections/<section>" naming convention.
func Section(c *fiber.Ctx, section string, data fiber.Map) error {
	if c.Get("HX-Request") != "true" {
		data["Section"] = section
		return c.Render("main", data)
	}
	return c.Render("sections/"+section, data)
}

// Err responds with an error toast for HTMX requests or a JSON error body otherwise.
func Err(c *fiber.Ctx, status int, msg string) error {
	if c.Get("HX-Request") == "true" {
		return c.Status(status).Render("toast/toastErr", fiber.Map{"Msg": msg})
	}
	return c.Status(status).JSON(fiber.Map{"error": msg})
}

// Ok responds with a success toast for HTMX requests or a JSON message otherwise.
func Ok(c *fiber.Ctx, msg string) error {
	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{"Msg": msg})
	}
	return c.JSON(fiber.Map{"message": msg})
}

// Job responds with a success toast for HTMX requests or a 202 JSON body with the job_id otherwise.
func Job(c *fiber.Ctx, jobID string, msg string) error {
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

// Resource serves binary or file content for HTMX requests or returns {"type": mimeType, "url": url} JSON otherwise.
// The serve func is only called for HTMX requests and is responsible for setting headers and writing the body.
func Resource(c *fiber.Ctx, mimeType, url string, serve func() error) error {
	if c.Get("HX-Request") != "true" {
		return c.JSON(fiber.Map{"type": mimeType, "url": url})
	}
	return serve()
}
