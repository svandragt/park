package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/svandragt/park/internal/db"
)

func TestRunMigrate_CopiesDB(t *testing.T) {
	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "park.db")
	conn, err := db.Open(srcPath)
	if err != nil {
		t.Fatalf("create src db: %v", err)
	}
	conn.Close()

	destDir := t.TempDir()
	if err := RunMigrate(srcPath, []string{destDir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	destPath := filepath.Join(destDir, "park.db")
	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("expected DB at %s: %v", destPath, err)
	}
}

func TestRunMigrate_NoArgs(t *testing.T) {
	if err := RunMigrate("/some/path.db", nil); err == nil {
		t.Error("expected error for missing dest arg")
	}
}
