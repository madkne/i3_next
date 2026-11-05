package main

import (
	"fmt"
	"os"

	"i3_next/internal/version"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("i3-next-session %s (%s, %s)\n", version.Version, version.Commit, version.BuildDate)
		return
	}

	fmt.Println("i3_next session manager")
	fmt.Println("Session bootstrap is not implemented yet.")
}
