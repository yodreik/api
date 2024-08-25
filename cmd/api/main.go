package main

import (
	"api/internal/config"
	"api/internal/pkg/app"
)

func main() {
	c := config.MustLoad()
	a := app.New(c)

	a.Run()
}
