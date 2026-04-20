package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/svandragt/park/cmd"
	"github.com/svandragt/park/internal/db"
	"github.com/svandragt/park/internal/park"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "park:", err)
		os.Exit(1)
	}
}

func run() error {
	dbPath := os.Getenv("PARK_DB")
	if dbPath == "" {
		dataHome := os.Getenv("XDG_DATA_HOME")
		if dataHome == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			dataHome = filepath.Join(home, ".local", "share")
		}
		dir := filepath.Join(dataHome, "park")
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		dbPath = filepath.Join(dir, "park.db")
	}

	database, err := db.Open(dbPath)
	if err != nil {
		return err
	}
	defer database.Close()

	store := park.New(database)

	if len(os.Args) < 2 {
		return fmt.Errorf("usage: park <add|list|show|done|archive> [flags]")
	}

	sub := os.Args[1]
	args := os.Args[2:]

	switch sub {
	case "add":
		return cmd.RunAdd(store, args)
	case "list", "ls":
		return cmd.RunList(store, args)
	case "show":
		return cmd.RunShow(store, args)
	case "done":
		return cmd.RunDone(store, args)
	case "archive":
		return cmd.RunArchive(store, args)
	default:
		return fmt.Errorf("unknown command: %s", sub)
	}
}
