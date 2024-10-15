package main

import (
	_ "api/docs"
	"api/internal/config"
	"api/internal/pkg/app"
)

// @title       yodreik API
// @version     0.1
// @description API server for yodreik application
// @host        dreik.d.qarwe.online
// @BasePath    /api
// @schemes     https
//
// @securityDefinitions.apikey AccessToken
// @in                         header
// @name                       Authorization
func main() {
	c := config.MustLoad()
	a := app.New(c)

	a.Run()
}
