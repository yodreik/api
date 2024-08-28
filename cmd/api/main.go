package main

import (
	_ "api/docs"
	"api/internal/config"
	"api/internal/pkg/app"
)

// @title       Welnex API
// @version     0.1
// @description API server for Welnex application
// @host        localhost:6969
// @BasePath    /api
func main() {
	c := config.MustLoad()
	a := app.New(c)

	a.Run()
}
