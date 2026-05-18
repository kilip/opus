//go:build integration

package gofiber_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/server/ent/enttest"
	"github.com/kilip/opus/server/internal/adapter/entgo"
	"github.com/kilip/opus/server/internal/auth"
	"github.com/kilip/opus/server/internal/delivery/gofiber/handler"
	_ "github.com/mattn/go-sqlite3"
)

func TestRouterWiring(t *testing.T) {
	app := fiber.New()
	_ = context.Background()
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	repo := entgo.NewAuthRepo(client)
	reg := auth.NewProviderRegistry()
	svc := auth.NewService(repo, reg, nil, auth.Config{
		JWTSecret:       "sec",
		AccessTokenTTL:  "15m",
		RefreshTokenTTL: "168h",
	}, nil)

	h := handler.NewAuthHandler(svc, repo)

	// Test if route register call executes cleanly without panic
	handler.RegisterAuthRoutes(app, h, repo, auth.Config{
		JWTSecret: "sec",
	})

	req, _ := http.NewRequest("POST", "/auth/register", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed test request: %v", err)
	}

	// Should respond with BadRequest envelope instead of 404 (indicating route was matched)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 BadRequest, got %d", resp.StatusCode)
	}
}
