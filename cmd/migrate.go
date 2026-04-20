package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/svandragt/park/internal/db"
)

func RunMigrate(srcPath string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: park migrate <destination-dir>")
	}
	destDir := args[0]
	if err := os.MkdirAll(destDir, 0700); err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	destPath := filepath.Join(destDir, filepath.Base(srcPath))

	if err := copyFile(srcPath, destPath); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	conn, err := db.Open(destPath)
	if err != nil {
		os.Remove(destPath)
		return fmt.Errorf("verify: %w", err)
	}
	conn.Close()

	fmt.Printf("Database migrated to: %s\n", destPath)
	fmt.Printf("Add to your shell profile:\n  export PARK_DB=%s\n", destPath)
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
