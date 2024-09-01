package router

import (
	"api/internal/app/handler"
	"api/internal/config"
	"api/internal/repository"
	"api/pkg/requestid"
	"api/pkg/requestlog"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Router struct {
	config  *config.Config
	handler *handler.Handler
}

func New(c *config.Config, r *repository.Repository) *Router {
	h := handler.New(c, r)
	return &Router{
		config:  c,
		handler: h,
	}
}

func (r *Router) InitRoutes() *gin.Engine {
	router := gin.New()

	router.Use(gin.Recovery())

	router.Use(requestid.New)
	router.Use(requestlog.Handled)

	if r.config.Env == config.EnvLocal {
		router.GET("/coverage", func(c *gin.Context) {
			c.File("./coverage.html")
		})
	}

	api := router.Group("/api")
	{
		api.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

		api.GET("/healthcheck", r.handler.Healthcheck)

		api.POST("/auth/register", r.handler.Register)
		api.POST("/auth/login", r.handler.Login)

		api.GET("/me", r.handler.UserIdentity, r.handler.Me)

		api.POST("/workout", r.handler.UserIdentity, r.handler.CreateWorkout)
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
