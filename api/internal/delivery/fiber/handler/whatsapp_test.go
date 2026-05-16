package handler

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/mock/gomock"
	"github.com/kilip/opus/api/mocks"
)

func TestWhatsAppHandler_Status(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockWhatsAppService(ctrl)
	app := fiber.New()
	
	// Mock auth middleware - assuming it sets userID in Locals
	app.Use(func(c fiber.Ctx) error {
		c.Locals("userID", "user-123")
		return c.Next()
	})

	NewWhatsAppHandler(app.Group("/whatsapp"), mockSvc)

	mockSvc.EXPECT().GetStatus(gomock.Any(), "user-123").Return("CONNECTED", "62812@s.whatsapp.net", nil)

	req := httptest.NewRequest("GET", "/whatsapp/status", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestWhatsAppHandler_Send(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockWhatsAppService(ctrl)
	app := fiber.New()
	
	app.Use(func(c fiber.Ctx) error {
		c.Locals("userID", "user-123")
		return c.Next()
	})

	NewWhatsAppHandler(app.Group("/whatsapp"), mockSvc)

	mockSvc.EXPECT().SendMessage(gomock.Any(), "user-123", "target@s.whatsapp.net", "hello").Return(nil)

	body, _ := json.Marshal(map[string]string{
		"target_jid": "target@s.whatsapp.net",
		"message":    "hello",
	})
	req := httptest.NewRequest("POST", "/whatsapp/send", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
