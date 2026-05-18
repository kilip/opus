package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/server/internal/auth"
	"github.com/kilip/opus/server/internal/delivery/gofiber/handler"
	"go.uber.org/mock/gomock"
)

func TestAuthHandler_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := auth.NewMockRepository(ctrl)
	mockPolicy := auth.NewMockPolicyService(ctrl)
	mockRegistry := auth.NewProviderRegistry()

	svc := auth.NewService(mockRepo, mockRegistry, mockPolicy, auth.Config{
		JWTSecret:       "jwt-secret",
		AccessTokenTTL:  "15m",
		RefreshTokenTTL: "168h",
	}, nil)

	h := handler.NewAuthHandler(svc, mockRepo)
	app := fiber.New()
	app.Post("/auth/register", h.Register)

	// Expectations
	mockRepo.EXPECT().FindUserByEmail(gomock.Any(), "new@email.com").Return(nil, auth.ErrUserNotFound)
	mockRepo.EXPECT().CreateUserWithWorkspace(gomock.Any(), gomock.Any(), nil, "New User's Workspace").Return(&auth.User{
		ID:          "u1",
		Email:       "new@email.com",
		WorkspaceID: "ws1",
	}, nil)
	mockPolicy.EXPECT().AssignRole(gomock.Any(), "u1", "ws1", "admin").Return(true, nil)
	mockRepo.EXPECT().CreateSession(gomock.Any(), gomock.Any()).Return(nil)
	mockRepo.EXPECT().CreateToken(gomock.Any(), gomock.Any()).Times(2).Return(nil)

	payload := map[string]string{
		"email":    "new@email.com",
		"password": "secretpass123",
		"name":     "New User",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed test request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
}
