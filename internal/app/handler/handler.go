package handler

import (
	"api/internal/config"
	"api/internal/repository"
)

type Handler struct {
	config     *config.Config
	repository *repository.Repository
}

func New(c *config.Config, r *repository.Repository) *Handler {
	return &Handler{
		config:     c,
		repository: r,
	}
}
