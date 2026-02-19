package main

import (
	"log"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/platform"
)

func main() {
	if err := platform.RunBootService("router"); err != nil {
		log.Fatalf("router failed: %v", err)
	}
}
