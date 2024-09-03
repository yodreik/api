package main

import (
	_ "api/docs"
	"api/pkg/random"
	"fmt"
)

// @title       Welnex API
// @version     0.1
// @description API server for Welnex application
// @host        localhost:6969
// @BasePath    /api
//
// @securityDefinitions.apikey AccessToken
// @in                         header
// @name                       Authorization
func main() {
	// c := config.MustLoad()
	// a := app.New(c)

	// a.Run()

	for i := 0; i < 5; i++ {
		fmt.Println(random.String(20))
	}
}
