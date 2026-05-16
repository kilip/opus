package fiber

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/api/internal/config"
	"github.com/kilip/opus/api/internal/delivery/fiber/handler"
	"github.com/kilip/opus/api/internal/delivery/fiber/middleware"
	"github.com/kilip/opus/api/internal/repository/queue"
	"github.com/kilip/opus/api/internal/service"
)

// Server encapsulates the Fiber application and its dependencies
type Server struct {
	app         *fiber.App
	cfg         *config.Config
	authService service.AuthServiceInterface
	userService service.UserServiceInterface
	queueDriver queue.QueueDriver
	sseHub      service.SSEHub
	waService   service.WhatsAppService
}

// NewServer creates a new Fiber server instance
func NewServer(
	cfg *config.Config,
	authService service.AuthServiceInterface,
	userService service.UserServiceInterface,
	waService service.WhatsAppService,
	queueDriver queue.QueueDriver,
	sseHub service.SSEHub,
) *Server {
	app := fiber.New(fiber.Config{
		AppName:      "Opus API",
		ErrorHandler: middleware.ErrorHandler,
	})

	s := &Server{
		app:         app,
		cfg:         cfg,
		authService: authService,
		userService: userService,
		queueDriver: queueDriver,
		sseHub:      sseHub,
		waService:   waService,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

func (s *Server) setupMiddleware() {
	s.app.Use(middleware.Logger())
	s.app.Use(middleware.Recovery())
}

func (s *Server) setupRoutes() {
	// Handlers
	authHandler := handler.NewAuthHandler(s.authService, s.userService, s.cfg)
	userHandler := handler.NewUserHandler(s.userService)
	healthHandler := handler.NewHealthHandler()
	sseHandler := handler.NewSSEHandler(s.sseHub)

	// Public Routes
	s.app.Get("/health", healthHandler.Check)

	auth := s.app.Group("/auth")
	auth.Post("/login", authHandler.Login)
	auth.Post("/refresh", authHandler.Refresh)
	auth.Post("/logout", authHandler.Logout)
	auth.Get("/google", authHandler.GoogleLogin)
	auth.Get("/google/callback", authHandler.GoogleCallback)

	// Protected Routes
	s.app.Get("/user/me", userHandler.Me, middleware.Auth(s.authService))
	s.app.Get("/stream", sseHandler.Stream, middleware.Auth(s.authService))

	// Queue/DLQ Routes
	queueHandler := handler.NewQueueHandler(s.queueDriver)
	q := s.app.Group("/queue", middleware.Auth(s.authService))
	q.Get("/dead", queueHandler.ListDeadLetters)
	q.Post("/dead/:id/retry", queueHandler.RetryDeadLetter)
	q.Delete("/dead/:id", queueHandler.DeleteDeadLetter)

	// WhatsApp Routes
	wa := s.app.Group("/whatsapp", middleware.Auth(s.authService))
	handler.NewWhatsAppHandler(wa, s.waService)
}

// Start starts the Fiber server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	return s.app.Listen(addr)
}

// Stop gracefully shuts down the Fiber server
func (s *Server) Stop() error {
	return s.app.Shutdown()
}
