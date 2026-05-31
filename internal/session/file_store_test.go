package session

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStoreCreateWritesAndReadsMetadata(t *testing.T) {
	store := NewFileStore(t.TempDir())
	metadata := testMetadata()

	if err := store.Create(context.Background(), metadata); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(store.root, metadata.ID, metadataFileName)); err != nil {
		t.Fatalf("metadata file was not created: %v", err)
	}

	decoded, err := store.ReadMetadata(context.Background(), metadata.ID)
	if err != nil {
		t.Fatalf("ReadMetadata returned error: %v", err)
	}

	assertMetadataEqual(t, decoded, metadata)
}

func TestFileStoreWriteMetadataUpdatesExistingFile(t *testing.T) {
	store := NewFileStore(t.TempDir())
	metadata := testMetadata()

	if err := store.Create(context.Background(), metadata); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	endedAt := metadata.CreatedAt.Add(3 * time.Second)
	metadata.EndedAt = &endedAt
	metadata.DurationMS = 3000
	metadata.Status = StatusCompleted

	if err := store.WriteMetadata(context.Background(), metadata); err != nil {
		t.Fatalf("WriteMetadata returned error: %v", err)
	}

	decoded, err := store.ReadMetadata(context.Background(), metadata.ID)
	if err != nil {
		t.Fatalf("ReadMetadata returned error: %v", err)
	}

	assertMetadataEqual(t, decoded, metadata)
}

func TestFileStoreReadMetadataMissingFile(t *testing.T) {
	store := NewFileStore(t.TempDir())

	_, err := store.ReadMetadata(context.Background(), "missing")
	if err == nil {
		t.Fatal("ReadMetadata returned nil error for missing metadata")
	}
}

func TestFileStoreRequiresSessionID(t *testing.T) {
	store := NewFileStore(t.TempDir())
	metadata := testMetadata()
	metadata.ID = ""

	if err := store.Create(context.Background(), metadata); err == nil {
		t.Fatal("Create returned nil error for empty id")
	}
	if err := store.WriteMetadata(context.Background(), metadata); err == nil {
		t.Fatal("WriteMetadata returned nil error for empty id")
	}
	if _, err := store.ReadMetadata(context.Background(), ""); err == nil {
		t.Fatal("ReadMetadata returned nil error for empty id")
	}
}

func TestFileStoreRejectsPathTraversalID(t *testing.T) {
	store := NewFileStore(t.TempDir())
	for _, id := range []string{"../escape", "foo/bar", `foo\bar`, "..", "./foo"} {
		metadata := testMetadata()
		metadata.ID = id
		if err := store.Create(context.Background(), metadata); err == nil {
			t.Fatalf("Create accepted unsafe id %q", id)
		}
		if _, err := store.ReadMetadata(context.Background(), id); err == nil {
			t.Fatalf("ReadMetadata accepted unsafe id %q", id)
		}
	}
}

func TestFileStoreWritesRestrictivePerms(t *testing.T) {
	store := NewFileStore(t.TempDir())
	metadata := testMetadata()

	if err := store.Create(context.Background(), metadata); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	dirInfo, err := os.Stat(filepath.Join(store.root, metadata.ID))
	if err != nil {
		t.Fatalf("stat session dir: %v", err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0o700 {
		t.Fatalf("session dir perm = %o, want 0700", perm)
	}

	fileInfo, err := os.Stat(filepath.Join(store.root, metadata.ID, metadataFileName))
	if err != nil {
		t.Fatalf("stat metadata file: %v", err)
	}
	if perm := fileInfo.Mode().Perm(); perm != 0o600 {
		t.Fatalf("metadata file perm = %o, want 0600", perm)
	}
}

func TestFileStoreListReturnsSortedByCreatedAtDesc(t *testing.T) {
	store := NewFileStore(t.TempDir())
	base := time.Date(2026, 5, 30, 8, 0, 0, 0, time.UTC)

	ids := []string{"alpha", "bravo", "charlie"}
	for i, id := range ids {
		m := testMetadata()
		m.ID = id
		m.Name = id
		m.CreatedAt = base.Add(time.Duration(i) * time.Minute)
		if err := store.Create(context.Background(), m); err != nil {
			t.Fatalf("Create %s: %v", id, err)
		}
	}

	got, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("count = %d, want 3", len(got))
	}
	wantOrder := []string{"charlie", "bravo", "alpha"}
	for i, id := range wantOrder {
		if got[i].ID != id {
			t.Fatalf("order[%d] = %q, want %q", i, got[i].ID, id)
		}
	}
}

func TestFileStoreListEmptyRoot(t *testing.T) {
	store := NewFileStore(t.TempDir())
	got, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
}

func TestFileStoreListMissingRoot(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "does-not-exist"))
	got, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
}

func TestFileStoreListSkipsInvalidEntries(t *testing.T) {
	root := t.TempDir()
	store := NewFileStore(root)

	if err := store.Create(context.Background(), testMetadata()); err != nil {
		t.Fatalf("Create valid: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "stray-file"), []byte("noise"), 0o600); err != nil {
		t.Fatalf("stray file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "empty-dir"), 0o700); err != nil {
		t.Fatalf("empty dir: %v", err)
	}

	got, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].ID != testMetadata().ID {
		t.Fatalf("ID = %q, want %q", got[0].ID, testMetadata().ID)
	}
}

func TestFileStoreListRespectsCanceledContext(t *testing.T) {
	store := NewFileStore(t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.List(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("List error = %v, want context.Canceled", err)
	}
}

func TestFileStoreRespectsCanceledContext(t *testing.T) {
	store := NewFileStore(t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := store.Create(ctx, testMetadata()); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create error = %v, want context.Canceled", err)
	}
}

func testMetadata() Metadata {
	return Metadata{
		ID:            "20260530-080000-monocron-demo",
		Name:          "monocron-demo",
		CreatedAt:     time.Date(2026, 5, 30, 8, 0, 0, 0, time.UTC),
		Shell:         "/bin/zsh",
		CWD:           "/Users/tolani/projects/monocron",
		Cols:          120,
		Rows:          40,
		Status:        StatusRecording,
		RekordVersion: "0.1.0",
	}
}

func assertMetadataEqual(t *testing.T, got, want Metadata) {
	t.Helper()

	if got.ID != want.ID || got.Name != want.Name {
		t.Fatalf("metadata identity = (%q, %q), want (%q, %q)", got.ID, got.Name, want.ID, want.Name)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("CreatedAt = %s, want %s", got.CreatedAt, want.CreatedAt)
	}
	if (got.EndedAt == nil) != (want.EndedAt == nil) {
		t.Fatalf("EndedAt nil = %v, want %v", got.EndedAt == nil, want.EndedAt == nil)
	}
	if got.EndedAt != nil && !got.EndedAt.Equal(*want.EndedAt) {
		t.Fatalf("EndedAt = %s, want %s", *got.EndedAt, *want.EndedAt)
	}
	if got.DurationMS != want.DurationMS || got.Status != want.Status {
		t.Fatalf("metadata duration/status = (%d, %q), want (%d, %q)", got.DurationMS, got.Status, want.DurationMS, want.Status)
	}
	if got.Shell != want.Shell || got.CWD != want.CWD {
		t.Fatalf("metadata shell/cwd = (%q, %q), want (%q, %q)", got.Shell, got.CWD, want.Shell, want.CWD)
	}
	if got.Cols != want.Cols || got.Rows != want.Rows {
		t.Fatalf("metadata size = (%d, %d), want (%d, %d)", got.Cols, got.Rows, want.Cols, want.Rows)
	}
	if got.RekordVersion != want.RekordVersion {
		t.Fatalf("RekordVersion = %q, want %q", got.RekordVersion, want.RekordVersion)
	}
}
