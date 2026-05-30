package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const metadataFileName = "metadata.json"

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
	if metadata.ID == "" {
		return fmt.Errorf("session id is required")
	}

	if err := os.MkdirAll(s.sessionPath(metadata.ID), 0o755); err != nil {
		return fmt.Errorf("create session directory: %w", err)
	}

	return s.WriteMetadata(ctx, metadata)
}

func (s *FileStore) ReadMetadata(ctx context.Context, sessionID string) (Metadata, error) {
	if err := ctx.Err(); err != nil {
		return Metadata{}, err
	}
	if sessionID == "" {
		return Metadata{}, fmt.Errorf("session id is required")
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
	if metadata.ID == "" {
		return fmt.Errorf("session id is required")
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("encode metadata: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(s.sessionPath(metadata.ID), 0o755); err != nil {
		return fmt.Errorf("create session directory: %w", err)
	}
	if err := os.WriteFile(s.metadataPath(metadata.ID), data, 0o644); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}

	return nil
}

func (s *FileStore) sessionPath(sessionID string) string {
	return filepath.Join(s.root, sessionID)
}

func (s *FileStore) metadataPath(sessionID string) string {
	return filepath.Join(s.sessionPath(sessionID), metadataFileName)
}
