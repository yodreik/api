package router

import (
	"api/internal/app/handler"
	"api/internal/config"
	"api/internal/mailer"
	"api/internal/repository"
	"api/internal/token"
	"api/pkg/requestid"
	"api/pkg/requestlog"

	"github.com/gin-gonic/gin"
	files "github.com/swaggo/files"
	swaggin "github.com/swaggo/gin-swagger"
)

type Router struct {
	config  *config.Config
	handler *handler.Handler
}

func New(c *config.Config, r *repository.Repository, m mailer.Mailer, t token.Manager) *Router {
	h := handler.New(c, r, m, t)
	return &Router{
		config:  c,
		handler: h,
	}
}

func (r *Router) InitRoutes() *gin.Engine {
	router := gin.New()

	router.Use(gin.Recovery())

	router.Use(requestid.New)
	router.Use(requestlog.Completed)

	switch r.config.Env {
	case config.EnvLocal, config.EnvDevelopment:
		router.Use(func(c *gin.Context) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}

			c.Next()
		})
	}

	api := router.Group("/api")
	{
		switch r.config.Env {
		case config.EnvLocal, config.EnvDevelopment:
			api.GET("/coverage", func(c *gin.Context) {
				c.File("./coverage.html")
			})

			api.GET("/docs/*any", swaggin.WrapHandler(files.Handler))
		}

		api.GET("/healthcheck", r.handler.Healthcheck)

		api.POST("/auth/session", r.handler.CreateSession)
		api.POST("/auth/account", r.handler.CreateAccount)

		api.Static("/avatar", ".database/avatars")
		api.PATCH("/account/avatar", r.handler.UserIdentity, r.handler.UploadAvatar)
		api.DELETE("/account/avatar", r.handler.UserIdentity, r.handler.DeleteAvatar)

		api.GET("/account", r.handler.UserIdentity, r.handler.GetCurrentAccount)
		api.PATCH("/account", r.handler.UserIdentity, r.handler.UpdateAccount)

		api.POST("/account/confirm", r.handler.ConfirmAccount)

		api.POST("/account/reset-password/request", r.handler.ResetPassword)
		api.PATCH("/account/reset-password", r.handler.UpdatePassword)

		api.POST("/workout", r.handler.UserIdentity, r.handler.CreateWorkout)
		api.DELETE("/workout/:id", r.handler.UserIdentity, r.handler.DeleteWorkout)

		api.GET("/activity", r.handler.UserIdentity, r.handler.GetActivityHistory)
		api.GET("/statistics", r.handler.UserIdentity, r.handler.GetStatistics)

		api.GET("/user/:username", r.handler.GetUserByUsername)
	}

	return router
}
