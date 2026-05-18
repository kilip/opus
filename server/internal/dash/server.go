package dash

import (
	"io/fs"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

// NewServer returns a Fiber app configured to serve the embedded Dash PWA assets.
func NewServer() *fiber.App {
	app := fiber.New()

	// Sub-select the 'dist' folder from the embedded FS
	distFS, _ := fs.Sub(FS, "dist")

	app.Use(static.New("", static.Config{
		FS:         distFS,
		IndexNames: []string{"index.html"},
		NotFoundHandler: func(c fiber.Ctx) error {
			return c.Status(200).SendFile("index.html", fiber.SendFile{
				FS: distFS,
			})
		},
	}))

	return app
}
