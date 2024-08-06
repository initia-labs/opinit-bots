package server

import "github.com/gofiber/fiber/v2"

type Server struct {
	*fiber.App
}

func NewServer() *Server {
	return &Server{
		fiber.New(),
	}
}

func (s *Server) Start(address string) error {
	return s.Listen(address)
}

func (s *Server) RegisterQuerier(path string, fn func(c *fiber.Ctx) error) {
	s.Get(path, fn)
}
