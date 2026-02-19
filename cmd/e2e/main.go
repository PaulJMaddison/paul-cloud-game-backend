package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	if _, err := exec.LookPath("docker"); err != nil {
		fmt.Println("SKIP: docker not found")
		return
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		fmt.Printf("SKIP: docker unavailable: %v\n", err)
		return
	}
	fmt.Println("E2E runner placeholder: prerequisites OK")
	os.Exit(0)
}
