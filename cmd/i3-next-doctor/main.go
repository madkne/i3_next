package main

import (
	"flag"
	"fmt"
	"os/exec"

	"i3_next/internal/version"
)

func main() {
	showVersion := flag.Bool("version", false, "Show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("i3-next-doctor %s (%s, %s)\n", version.Version, version.Commit, version.BuildDate)
		return
	}

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
