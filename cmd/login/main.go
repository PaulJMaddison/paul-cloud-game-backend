package main

import (
	"log"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/platform"
)

func main() {
	if err := platform.RunBootService("login"); err != nil {
		log.Fatalf("login failed: %v", err)
	}
}
