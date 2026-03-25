package processor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestProcessFilesConcurrentlyCollectsEntriesAndErrors(t *testing.T) {
	dir := t.TempDir()
	fileA := filepath.Join(dir, "a.log")
	fileB := filepath.Join(dir, "b.log")
	missing := filepath.Join(dir, "missing.log")

	contentA := "2023-12-25T14:30:15Z [INFO] user-service: request_id=req_1 ok\n"
	contentB := "2023-12-25T14:30:16Z [ERROR] auth-service: request_id=req_2 failed\n"
	if err := os.WriteFile(fileA, []byte(contentA), 0o600); err != nil {
		t.Fatalf("WriteFile(fileA) returned error: %v", err)
	}
	if err := os.WriteFile(fileB, []byte(contentB), 0o600); err != nil {
		t.Fatalf("WriteFile(fileB) returned error: %v", err)
	}

	entries, err := ProcessFilesConcurrently(
		context.Background(),
		[]string{fileA, missing, fileB},
		2,
	)
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want %d", len(entries), 2)
	}
	if err == nil {
		t.Fatal("ProcessFilesConcurrently error = nil, want non-nil")
	}
}

func TestProcessFilesConcurrentlyContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	entries, err := ProcessFilesConcurrently(ctx, []string{"unused.log"}, 1)
	if len(entries) != 0 {
		t.Fatalf("len(entries) = %d, want %d", len(entries), 0)
	}
	if err == nil {
		t.Fatal("ProcessFilesConcurrently error = nil, want non-nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
