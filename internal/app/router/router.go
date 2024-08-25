package router

import (
	"api/internal/app/handler"
	"api/internal/config"
	"api/pkg/requestid"
	"fmt"
	"log/slog"

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

	api := router.Group("/api")
	{
		api.POST("/auth/register", r.handler.Register)
		api.POST("/auth/login", r.handler.Login)
	}

	r.log(router.Routes())

	return router
}

func (r *Router) log(routes gin.RoutesInfo) {
	for _, route := range routes {
		if r.config.Env == config.EnvLocal {
			record := fmt.Sprintf("Registered handler for %s %s --> %s", route.Method, route.Path, route.Handler)
			slog.Info(record)
		}
	}
}
