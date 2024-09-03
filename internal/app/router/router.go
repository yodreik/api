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

	switch r.config.Env {
	case config.EnvLocal, config.EnvDevelopment:
		router.Use(func(c *gin.Context) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH")

			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}

			c.Next()
		})

		router.GET("/coverage", func(c *gin.Context) {
			c.File("./coverage.html")
		})

		router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	api := router.Group("/api")
	{

		api.GET("/healthcheck", r.handler.Healthcheck)

		api.POST("/auth/register", r.handler.Register)
		api.POST("/auth/login", r.handler.Login)

		api.POST("/auth/password/reset", r.handler.ResetPassword)
		api.PATCH("/auth/password/update", r.handler.UpdatePassword)

		api.GET("/me", r.handler.UserIdentity, r.handler.Me)

		api.POST("/workout", r.handler.UserIdentity, r.handler.CreateWorkout)
	}

	r.log(router.Routes())

	return router
}

func (r *Router) log(routes gin.RoutesInfo) {
	for _, route := range routes {
		switch r.config.Env {
		case config.EnvLocal, config.EnvDevelopment:
			record := fmt.Sprintf("Registered handler for %s %s --> %s", route.Method, route.Path, route.Handler)
			slog.Info(record)
		}
	}
}
