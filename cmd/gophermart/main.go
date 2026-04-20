package main

import (
	"gophermart/internal/router"
	"log"
)

func main() {
	if err := router.Run(); err != nil {
		log.Fatal(err)
	}
}
