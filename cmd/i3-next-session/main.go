package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("i3-next-session %s (%s)\n", VERSION, commit)
		return
	}

	fmt.Println("i3_next session manager")
	fmt.Println("Session bootstrap is not implemented yet.")
}
