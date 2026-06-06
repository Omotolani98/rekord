package memory

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	memDirPerm  = 0o700
	memFilePerm = 0o600
)

type FileStore struct {
	root string
}

func NewFileStore(root string) *FileStore {
	if root == "" {
		root = DefaultRoot()
	}
	return &FileStore{root: root}
}

func (s *FileStore) AddMemory(ctx context.Context, m Memory) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateMemory(m); err != nil {
		return err
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = m.CreatedAt
	}
	if m.Type == "" {
		m.Type = TypeNote
	}
	if m.Status == "" {
		m.Status = StatusOpen
	}
	return appendJSONL(s.memoriesPath(m.Project), m)
}

func (s *FileStore) ListMemories(ctx context.Context, f Filter) ([]Memory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	project, err := NormalizeProject(f.Project)
	if err != nil {
		return nil, err
	}
	items, err := readMemories(s.memoriesPath(project))
	if err != nil {
		return nil, err
	}
	items = filterMemories(items, f)
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return limitMemories(items, f.Limit), nil
}

func (s *FileStore) SearchMemories(ctx context.Context, query string, f Filter) ([]Memory, error) {
	items, err := s.ListMemories(ctx, Filter{Project: f.Project, Agent: f.Agent, FromAgent: f.FromAgent, Session: f.Session, Status: f.Status})
	if err != nil {
		return nil, err
	}
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return limitMemories(items, f.Limit), nil
	}
	type scored struct {
		m     Memory
		score int
	}
	var scoredItems []scored
	for _, m := range items {
		score := scoreMemory(m, query)
		if score > 0 {
			scoredItems = append(scoredItems, scored{m: m, score: score})
		}
	}
	sort.Slice(scoredItems, func(i, j int) bool {
		if scoredItems[i].score == scoredItems[j].score {
			return scoredItems[i].m.CreatedAt.After(scoredItems[j].m.CreatedAt)
		}
		return scoredItems[i].score > scoredItems[j].score
	})
	out := make([]Memory, len(scoredItems))
	for i, item := range scoredItems {
		out[i] = item.m
	}
	return limitMemories(out, f.Limit), nil
}

func (s *FileStore) GetMemory(ctx context.Context, project, id string) (Memory, error) {
	items, err := s.ListMemories(ctx, Filter{Project: project})
	if err != nil {
		return Memory{}, err
	}
	for _, m := range items {
		if m.ID == id {
			return m, nil
		}
	}
	return Memory{}, fmt.Errorf("memory not found: %q", id)
}

func (s *FileStore) UpdateMemory(ctx context.Context, updated Memory) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateMemory(updated); err != nil {
		return err
	}
	items, err := readMemories(s.memoriesPath(updated.Project))
	if err != nil {
		return err
	}
	found := false
	for i := range items {
		if items[i].ID == updated.ID {
			items[i] = updated
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("memory not found: %q", updated.ID)
	}
	return writeMemories(s.memoriesPath(updated.Project), items)
}

func (s *FileStore) CreateSnapshot(ctx context.Context, snap Snapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateSnapshot(snap); err != nil {
		return err
	}
	path := s.snapshotPath(snap.Project, snap.ID)
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("encode snapshot: %w", err)
	}
	data = append(data, '\n')
	return writeFileAtomic(path, data)
}

func (s *FileStore) ListSnapshots(ctx context.Context, f Filter) ([]Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	project, err := NormalizeProject(f.Project)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.snapshotsDir(project))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read snapshots: %w", err)
	}
	var out []Snapshot
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.snapshotsDir(project), entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read snapshot: %w", err)
		}
		var snap Snapshot
		if err := json.Unmarshal(data, &snap); err != nil {
			return nil, fmt.Errorf("decode snapshot: %w", err)
		}
		if matchSnapshot(snap, f) {
			out = append(out, snap)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}

func (s *FileStore) LatestSnapshot(ctx context.Context, f Filter) (Snapshot, error) {
	f.Limit = 1
	snaps, err := s.ListSnapshots(ctx, f)
	if err != nil {
		return Snapshot{}, err
	}
	if len(snaps) == 0 {
		return Snapshot{}, fs.ErrNotExist
	}
	return snaps[0], nil
}

func (s *FileStore) ProjectDir(project string) string {
	project, _ = NormalizeProject(project)
	return filepath.Join(s.root, ProjectKey(project))
}

func (s *FileStore) PatchDir(project string) string {
	return filepath.Join(s.ProjectDir(project), "patches")
}

func (s *FileStore) snapshotsDir(project string) string {
	return filepath.Join(s.ProjectDir(project), "snapshots")
}

func (s *FileStore) memoriesPath(project string) string {
	return filepath.Join(s.ProjectDir(project), "memories.jsonl")
}

func (s *FileStore) snapshotPath(project, id string) string {
	return filepath.Join(s.snapshotsDir(project), id+".json")
}

func validateMemory(m Memory) error {
	if m.ID == "" {
		return fmt.Errorf("memory id is required")
	}
	if m.Project == "" {
		return fmt.Errorf("project is required")
	}
	if strings.TrimSpace(m.Body) == "" && strings.TrimSpace(m.Title) == "" {
		return fmt.Errorf("memory body is required")
	}
	return nil
}

func validateSnapshot(s Snapshot) error {
	if s.ID == "" {
		return fmt.Errorf("snapshot id is required")
	}
	if s.Project == "" {
		return fmt.Errorf("project is required")
	}
	return nil
}

func appendJSONL(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), memDirPerm); err != nil {
		return fmt.Errorf("create memory directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, memFilePerm)
	if err != nil {
		return fmt.Errorf("open memories: %w", err)
	}
	defer f.Close()
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("encode memory: %w", err)
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write memory: %w", err)
	}
	return nil
}

func readMemories(path string) ([]Memory, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("open memories: %w", err)
	}
	defer f.Close()
	var out []Memory
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		var m Memory
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			return nil, fmt.Errorf("decode memory: %w", err)
		}
		out = append(out, m)
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("read memories: %w", err)
	}
	return out, nil
}

func writeMemories(path string, items []Memory) error {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	for _, item := range items {
		if err := enc.Encode(item); err != nil {
			return fmt.Errorf("encode memories: %w", err)
		}
	}
	return writeFileAtomic(path, []byte(b.String()))
}

func writeFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), memDirPerm); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Chmod(memFilePerm); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

func WritePatch(path string, data []byte) (int64, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if err := writeFileAtomic(path, data); err != nil {
		return 0, err
	}
	return int64(len(data)), nil
}

func filterMemories(items []Memory, f Filter) []Memory {
	var out []Memory
	for _, m := range items {
		if !matchMemory(m, f) {
			continue
		}
		out = append(out, m)
	}
	return out
}

func matchMemory(m Memory, f Filter) bool {
	agent := f.Agent
	if agent == "" {
		agent = f.FromAgent
	}
	if agent != "" && m.Agent != agent {
		return false
	}
	if f.Status != "" && m.Status != f.Status {
		return false
	}
	if f.Session != "" && m.SessionID != f.Session && m.SessionName != f.Session {
		return false
	}
	return true
}

func matchSnapshot(s Snapshot, f Filter) bool {
	agent := f.Agent
	if agent == "" {
		agent = f.FromAgent
	}
	if agent != "" && s.Agent != agent {
		return false
	}
	if f.Session != "" && s.SessionID != f.Session && s.SessionName != f.Session {
		return false
	}
	return true
}

func limitMemories(items []Memory, limit int) []Memory {
	if limit > 0 && len(items) > limit {
		return items[:limit]
	}
	return items
}

func copyFile(dst string, src io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(dst), memDirPerm); err != nil {
		return err
	}
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, memFilePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, src)
	return err
}

var _ = copyFile
