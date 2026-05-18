package dash

import "github.com/gofiber/fiber/v3"

var dashApp *fiber.App

// Bootstrap initialises the Dash static server.
func Bootstrap(cfg Config) {
	dashApp = NewServer()
}

// GetServer returns the initialised Dash Fiber app.
func GetServer() *fiber.App {
	if dashApp == nil {
		panic("dash: Bootstrap has not been called")
	}
	return dashApp
}
