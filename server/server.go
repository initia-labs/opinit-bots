package server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/initia-labs/opinit-bots/server/types"
)

type Server struct {
	address string
	*fiber.App
}

func NewServer(cfg types.ServerConfig) *Server {
	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.AllowOrigins,
		AllowHeaders: cfg.AllowHeaders,
		AllowMethods: cfg.AllowMethods,
	}))

	return &Server{
		address: cfg.Address,
		App:     app,
	}
}

func (s *Server) Start() error {
	return s.Listen(s.address)
}

func (s *Server) RegisterQuerier(path string, fn func(c *fiber.Ctx) error) {
	s.Get(path, fn)
}
