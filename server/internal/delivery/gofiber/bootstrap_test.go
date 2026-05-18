package gofiber_test

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/server/internal/delivery/gofiber"
	"github.com/kilip/opus/server/internal/shared/logger"
	"testing"
)

func TestBootstrap_Panic(t *testing.T) {
	app := fiber.New()
	log := &logger.NoopLogger{}
	cfg := gofiber.Config{}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected bootstrap to panic due to missing domain services")
		}
	}()
	gofiber.Bootstrap(app, log, cfg)
}
