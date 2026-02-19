package main

import (
	"log"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/platform"
)

func main() {
	if err := platform.RunBootService("sessions"); err != nil {
		log.Fatalf("sessions failed: %v", err)
	}
}
