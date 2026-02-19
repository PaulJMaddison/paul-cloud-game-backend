package main

import (
	"log"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/platform"
)

func main() {
	if err := platform.RunBootService("gateway"); err != nil {
		log.Fatalf("gateway failed: %v", err)
	}
}
