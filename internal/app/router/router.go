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

		router.GET("/docs/*any", swaggin.WrapHandler(files.Handler))
	}

	api := router.Group("/api")
	{
		api.GET("/healthcheck", r.handler.Healthcheck)

		auth := api.Group("/auth")
		{
			auth.POST("/register", r.handler.Register)
			auth.POST("/login", r.handler.Login)

			auth.POST("/password/reset", r.handler.ResetPassword)
			auth.PATCH("/password/update", r.handler.UpdatePassword)

			auth.POST("/confirm", r.handler.ConfirmEmail)
		}

		api.GET("/me", r.handler.UserIdentity, r.handler.Me)

		api.GET("/me/workouts", r.handler.UserIdentity, r.handler.GetWorkouts)

		api.POST("/workout", r.handler.UserIdentity, r.handler.CreateWorkout)
	}

	return router
}
