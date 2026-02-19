package main

import (
	"log"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/platform"
)

func main() {
	if err := platform.RunBootService("matchmaking"); err != nil {
		log.Fatalf("matchmaking failed: %v", err)
	}
}
