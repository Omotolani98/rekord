package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	metadataFileName = "metadata.json"
	sessionDirPerm   = 0o700
	sessionFilePerm  = 0o600
)

type FileStore struct {
	root string
}

func NewFileStore(root string) *FileStore {
	return &FileStore{root: root}
}

func (s *FileStore) Create(ctx context.Context, metadata Metadata) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateSessionID(metadata.ID); err != nil {
		return err
	}

	return s.WriteMetadata(ctx, metadata)
}

func (s *FileStore) ReadMetadata(ctx context.Context, sessionID string) (Metadata, error) {
	if err := ctx.Err(); err != nil {
		return Metadata{}, err
	}
	if err := validateSessionID(sessionID); err != nil {
		return Metadata{}, err
	}

	data, err := os.ReadFile(s.metadataPath(sessionID))
	if err != nil {
		return Metadata{}, fmt.Errorf("read metadata: %w", err)
	}

	var metadata Metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return Metadata{}, fmt.Errorf("decode metadata: %w", err)
	}

	return metadata, nil
}

func (s *FileStore) WriteMetadata(ctx context.Context, metadata Metadata) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateSessionID(metadata.ID); err != nil {
		return err
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("encode metadata: %w", err)
	}
	data = append(data, '\n')

	dir := s.sessionPath(metadata.ID)
	if err := os.MkdirAll(dir, sessionDirPerm); err != nil {
		return fmt.Errorf("create session directory: %w", err)
	}

	final := s.metadataPath(metadata.ID)
	tmp, err := os.CreateTemp(dir, metadataFileName+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp metadata: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write metadata: %w", err)
	}
	if err := tmp.Chmod(sessionFilePerm); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("chmod metadata: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("sync metadata: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close metadata: %w", err)
	}
	if err := os.Rename(tmpName, final); err != nil {
		cleanup()
		return fmt.Errorf("rename metadata: %w", err)
	}

	return nil
}

func (s *FileStore) sessionPath(sessionID string) string {
	return filepath.Join(s.root, sessionID)
}

func (s *FileStore) metadataPath(sessionID string) string {
	return filepath.Join(s.sessionPath(sessionID), metadataFileName)
}

func validateSessionID(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	if strings.ContainsAny(sessionID, `/\`) || strings.Contains(sessionID, "..") {
		return fmt.Errorf("invalid session id %q", sessionID)
	}
	if sessionID != filepath.Clean(sessionID) {
		return fmt.Errorf("invalid session id %q", sessionID)
	}
	return nil
}
