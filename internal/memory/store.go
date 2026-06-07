package memory

import "context"

type Store interface {
	AddMemory(ctx context.Context, m Memory) error
	ListMemories(ctx context.Context, f Filter) ([]Memory, error)
	SearchMemories(ctx context.Context, query string, f Filter) ([]Memory, error)
	GetMemory(ctx context.Context, project, id string) (Memory, error)
	UpdateMemory(ctx context.Context, m Memory) error
	CreateSnapshot(ctx context.Context, s Snapshot) error
	ListSnapshots(ctx context.Context, f Filter) ([]Snapshot, error)
	LatestSnapshot(ctx context.Context, f Filter) (Snapshot, error)
	ProjectDir(project string) string
	PatchDir(project string) string
}
