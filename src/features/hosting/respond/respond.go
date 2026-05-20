package respond

import "github.com/gofiber/fiber/v2"

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
