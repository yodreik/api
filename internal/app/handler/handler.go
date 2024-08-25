package handler

import (
	"api/internal/config"
)

type Handler struct {
	config *config.Config
}

func New(cfg *config.Config) *Handler {
	return &Handler{
		config: cfg,
	}
}
