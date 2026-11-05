package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	showVersion := flag.Bool("version", false, "Show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("i3-nextctl %s (%s)\n", VERSION, commit)
		return
	}

	args := flag.Args()

	if len(args) == 0 {
		printUsage()
		return
	}

	switch args[0] {
	case "status":
		fmt.Println("i3_next: not connected")
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`i3-nextctl - i3_next Desktop Control

Usage:
  i3-nextctl [options] <command>

Commands:
  status       Show desktop status
  help         Show help

Options:
  --version    Show version
`)
}
