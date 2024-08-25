package router

import (
	"api/internal/app/handler"
	"api/internal/config"
	"api/pkg/requestid"

	"github.com/gin-gonic/gin"
)

type Router struct {
	config  *config.Config
	handler *handler.Handler
}

func New(cfg *config.Config) *Router {
	h := handler.New(cfg)
	return &Router{
		config:  cfg,
		handler: h,
	}
}

func (r *Router) InitRoutes() *gin.Engine {
	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(requestid.New)

	return router
}
