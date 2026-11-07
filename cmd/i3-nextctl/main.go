package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"i3_next/internal/core/config"
	"i3_next/internal/version"
)

func main() {
	showVersion := flag.Bool("version", false, "Show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("i3-nextctl %s (%s, %s)\n", version.Version, version.Commit, version.BuildDate)
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
	case "config":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: i3-nextctl config <validate|show|backup|restore> [file]")
			os.Exit(2)
		}
		runConfig(args[1:])
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		os.Exit(1)
	}
}

func runConfig(args []string) {
	switch args[0] {
	case "validate":
		path := ""
		if len(args) > 1 {
			path = args[1]
		} else {
			path = filepath.Join(config.ConfigDir(), "desktop.yaml")
		}
		if err := config.ValidateDesktopFile(path); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("valid")
	case "show":
		opts := config.LoadOptions{Env: true}
		cfg, _, err := config.LoadDesktopMerged(opts)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("%+v\n", cfg)
	case "backup":
		dir := config.ConfigDir()
		if len(args) > 1 {
			dir = args[1]
		}
		fmt.Println("backup dir:", dir)
	case "restore":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "restore requires a backup path")
			os.Exit(2)
		}
		fmt.Println("restore from:", args[1])
	default:
		fmt.Fprintf(os.Stderr, "unknown config subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`i3-nextctl - i3_next Desktop Control

Usage:
  i3-nextctl [options] <command>

Commands:
  status       Show desktop status
  config      validate|show|backup|restore
  help        Show help

Options:
  --version    Show version
`)
}
