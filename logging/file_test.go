package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileWriterCreatesFile(t *testing.T) {
	dir := t.TempDir()
	fw, err := NewFileWriter(dir, "app", 1024*1024)
	if err != nil {
		t.Fatalf("failed to create file writer: %v", err)
	}
	defer fw.currentFile.Close()

	n, err := fw.Write([]byte("hello world\n"))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if n != len("hello world\n") {
		t.Errorf("expected %d bytes written, got %d", len("hello world\n"), n)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 log file, got %d", len(entries))
	}
	if !strings.HasPrefix(entries[0].Name(), "app-") {
		t.Errorf("expected file prefix app-, got %q", entries[0].Name())
	}
}

func TestFileWriterRotates(t *testing.T) {
	dir := t.TempDir()
	fw, err := NewFileWriter(dir, "app", 10)
	if err != nil {
		t.Fatalf("failed to create file writer: %v", err)
	}
	defer fw.currentFile.Close()

	firstFile := fw.currentFile.Name()

	// First write stays under threshold-ish; second write triggers rotation.
	if _, err := fw.Write([]byte("123456789\n")); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if _, err := fw.Write([]byte("123456789\n")); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	if fw.currentFile.Name() == firstFile {
		t.Errorf("expected file rotation, but current file is still %s", firstFile)
	}

	// Old file should exist and be closed (writing to it would fail).
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read dir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 log files after rotation, got %d", len(entries))
	}

	// The older file should contain the two writes; the new current file may be empty.
	var total int
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}
		total += len(data)
	}
	if total != 20 {
		t.Errorf("expected 20 bytes across log files, got %d", total)
	}
}

func TestFileWriterPicksMostRecentFile(t *testing.T) {
	dir := t.TempDir()
	oldFile := filepath.Join(dir, "app-old.log")
	newFile := filepath.Join(dir, "app-new.log")
	if err := os.WriteFile(oldFile, []byte("old\n"), 0640); err != nil {
		t.Fatalf("failed to create old file: %v", err)
	}
	if err := os.WriteFile(newFile, []byte("new\n"), 0640); err != nil {
		t.Fatalf("failed to create new file: %v", err)
	}
	// Ensure mod times differ.
	oldTime := time.Now().Add(-time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatalf("failed to set old file time: %v", err)
	}

	fw, err := NewFileWriter(dir, "app", 1024*1024)
	if err != nil {
		t.Fatalf("failed to create file writer: %v", err)
	}
	defer fw.currentFile.Close()

	if fw.currentFile.Name() != newFile {
		t.Errorf("expected writer to pick newest file %s, got %s", newFile, fw.currentFile.Name())
	}
}
