package handler

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/api/internal/service"
)

// WhatsAppHandler handles HTTP requests for WhatsApp integration.
type WhatsAppHandler struct {
	svc service.WhatsAppService
}

// NewWhatsAppHandler registers WhatsApp routes.
func NewWhatsAppHandler(router fiber.Router, svc service.WhatsAppService) {
	h := &WhatsAppHandler{svc: svc}
	
	router.Get("/status", h.Status)
	router.Post("/connect", h.Connect)
	router.Post("/disconnect", h.Disconnect)
	router.Post("/send", h.Send)
}

// Status returns the current WhatsApp connection status.
func (h *WhatsAppHandler) Status(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "UNAUTHORIZED",
				"message": "unauthorized",
			},
		})
	}

	status, jid, err := h.svc.GetStatus(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": err.Error(),
			},
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"status": status,
			"jid":    jid,
		},
	})
}

// Connect initiates the WhatsApp connection process.
func (h *WhatsAppHandler) Connect(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "UNAUTHORIZED",
				"message": "unauthorized",
			},
		})
	}

	err := h.svc.Connect(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": err.Error(),
			},
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    fiber.Map{"message": "connection initiated"},
	})
}

// Disconnect closes the WhatsApp connection.
func (h *WhatsAppHandler) Disconnect(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "UNAUTHORIZED",
				"message": "unauthorized",
			},
		})
	}

	err := h.svc.Disconnect(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": err.Error(),
			},
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    fiber.Map{"message": "disconnected"},
	})
}

type sendReq struct {
	TargetJID string `json:"target_jid"`
	Message   string `json:"message"`
}

// Send sends a WhatsApp message.
func (h *WhatsAppHandler) Send(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "UNAUTHORIZED",
				"message": "unauthorized",
			},
		})
	}

	var req sendReq
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "BAD_REQUEST",
				"message": "invalid payload",
			},
		})
	}

	err := h.svc.SendMessage(c.Context(), userID, req.TargetJID, req.Message)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": err.Error(),
			},
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    fiber.Map{"message": "sent"},
	})
}
