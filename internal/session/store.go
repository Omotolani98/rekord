package session

import "context"

type Store interface {
	Create(ctx context.Context, metadata Metadata) error
	ReadMetadata(ctx context.Context, sessionID string) (Metadata, error)
	WriteMetadata(ctx context.Context, metadata Metadata) error
}
