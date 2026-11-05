package main

import (
	"fmt"
	"os/exec"
)

var version = "dev"

func main() {
	if _, err := exec.LookPath("sway"); err != nil {
		fmt.Println("[FAIL] sway not found")
	} else {
		fmt.Println("[OK] sway installed")
	}

	if _, err := exec.LookPath("quickshell"); err != nil {
		fmt.Println("[WARN] quickshell not found")
	} else {
		fmt.Println("[OK] quickshell installed")
	}

	if _, err := exec.LookPath("go"); err != nil {
		fmt.Println("[WARN] go not found")
	} else {
		fmt.Println("[OK] go installed")
	}
}