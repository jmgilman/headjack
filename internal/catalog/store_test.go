package catalog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStore(t *testing.T) {
	store := NewStore("/tmp/catalog.json")

	require.NotNil(t, store)
	assert.Equal(t, "/tmp/catalog.json", store.path)
}

func TestStore_Add(t *testing.T) {
	ctx := context.Background()

	t.Run("adds new entry", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), "catalog.json"))

		entry := Entry{
			ID:        "abc123",
			Repo:      "/path/to/repo",
			RepoID:    "myrepo-abc123",
			Branch:    "main",
			Worktree:  "/path/to/worktree",
			CreatedAt: time.Now(),
			Status:    StatusRunning,
		}
		err := store.Add(ctx, entry)

		require.NoError(t, err)

		got, err := store.Get(ctx, "abc123")
		require.NoError(t, err)
		assert.Equal(t, entry.ID, got.ID)
		assert.Equal(t, entry.RepoID, got.RepoID)
		assert.Equal(t, entry.Branch, got.Branch)
	})

	t.Run("returns ErrAlreadyExists for duplicate repo+branch", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), "catalog.json"))

		entry1 := Entry{
			ID:     "abc123",
			RepoID: "myrepo",
			Branch: "main",
		}
		entry2 := Entry{
			ID:     "def456",
			RepoID: "myrepo",
			Branch: "main",
		}

		err := store.Add(ctx, entry1)
		require.NoError(t, err)

		err = store.Add(ctx, entry2)
		assert.ErrorIs(t, err, ErrAlreadyExists)
	})

	t.Run("allows same branch in different repos", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), "catalog.json"))

		entry1 := Entry{ID: "abc123", RepoID: "repo1", Branch: "main"}
		entry2 := Entry{ID: "def456", RepoID: "repo2", Branch: "main"}

		require.NoError(t, store.Add(ctx, entry1))
		require.NoError(t, store.Add(ctx, entry2))

		entries, err := store.List(ctx, ListFilter{})
		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})
}

func TestStore_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("returns entry by ID", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), "catalog.json"))

		entry := Entry{ID: "abc123", RepoID: "myrepo", Branch: "main"}
		require.NoError(t, store.Add(ctx, entry))

		got, err := store.Get(ctx, "abc123")

		require.NoError(t, err)
		assert.Equal(t, "abc123", got.ID)
	})

	t.Run("returns ErrNotFound for missing ID", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), "catalog.json"))

		_, err := store.Get(ctx, "nonexistent")

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestStore_GetByRepoBranch(t *testing.T) {
	ctx := context.Background()

	t.Run("returns entry by repo+branch", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), "catalog.json"))

		entry := Entry{ID: "abc123", RepoID: "myrepo", Branch: "feat/auth"}
		require.NoError(t, store.Add(ctx, entry))

		got, err := store.GetByRepoBranch(ctx, "myrepo", "feat/auth")

		require.NoError(t, err)
		assert.Equal(t, "abc123", got.ID)
	})

	t.Run("returns ErrNotFound for missing repo+branch", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), "catalog.json"))

		_, err := store.GetByRepoBranch(ctx, "myrepo", "nonexistent")

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestStore_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("updates existing entry", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), "catalog.json"))

		entry := Entry{
			ID:     "abc123",
			RepoID: "myrepo",
			Branch: "main",
			Status: StatusCreating,
		}
		require.NoError(t, store.Add(ctx, entry))

		entry.Status = StatusRunning
		entry.ContainerID = "container-xyz"
		err := store.Update(ctx, entry)

		require.NoError(t, err)

		got, err := store.Get(ctx, "abc123")
		require.NoError(t, err)
		assert.Equal(t, StatusRunning, got.Status)
		assert.Equal(t, "container-xyz", got.ContainerID)
	})

	t.Run("returns ErrNotFound for missing entry", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), "catalog.json"))

		entry := Entry{ID: "nonexistent"}
		err := store.Update(ctx, entry)

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestStore_Remove(t *testing.T) {
	ctx := context.Background()

	t.Run("removes existing entry", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), "catalog.json"))

		entry := Entry{ID: "abc123", RepoID: "myrepo", Branch: "main"}
		require.NoError(t, store.Add(ctx, entry))

		err := store.Remove(ctx, "abc123")

		require.NoError(t, err)

		_, err = store.Get(ctx, "abc123")
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("returns ErrNotFound for missing entry", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), "catalog.json"))

		err := store.Remove(ctx, "nonexistent")

		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestStore_List(t *testing.T) {
	ctx := context.Background()

	t.Run("returns all entries when no filter", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), "catalog.json"))

		require.NoError(t, store.Add(ctx, Entry{ID: "a", RepoID: "repo1", Branch: "main"}))
		require.NoError(t, store.Add(ctx, Entry{ID: "b", RepoID: "repo2", Branch: "main"}))
		require.NoError(t, store.Add(ctx, Entry{ID: "c", RepoID: "repo1", Branch: "dev"}))

		entries, err := store.List(ctx, ListFilter{})

		require.NoError(t, err)
		assert.Len(t, entries, 3)
	})

	t.Run("filters by repo ID", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), "catalog.json"))

		require.NoError(t, store.Add(ctx, Entry{ID: "a", RepoID: "repo1", Branch: "main"}))
		require.NoError(t, store.Add(ctx, Entry{ID: "b", RepoID: "repo2", Branch: "main"}))
		require.NoError(t, store.Add(ctx, Entry{ID: "c", RepoID: "repo1", Branch: "dev"}))

		entries, err := store.List(ctx, ListFilter{RepoID: "repo1"})

		require.NoError(t, err)
		assert.Len(t, entries, 2)
		for _, e := range entries {
			assert.Equal(t, "repo1", e.RepoID)
		}
	})

	t.Run("filters by status", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), "catalog.json"))

		require.NoError(t, store.Add(ctx, Entry{ID: "a", RepoID: "repo1", Branch: "main", Status: StatusRunning}))
		require.NoError(t, store.Add(ctx, Entry{ID: "b", RepoID: "repo2", Branch: "main", Status: StatusStopped}))
		require.NoError(t, store.Add(ctx, Entry{ID: "c", RepoID: "repo1", Branch: "dev", Status: StatusRunning}))

		entries, err := store.List(ctx, ListFilter{Status: StatusRunning})

		require.NoError(t, err)
		assert.Len(t, entries, 2)
		for _, e := range entries {
			assert.Equal(t, StatusRunning, e.Status)
		}
	})

	t.Run("returns empty slice for no matches", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), "catalog.json"))

		require.NoError(t, store.Add(ctx, Entry{ID: "a", RepoID: "repo1"}))

		entries, err := store.List(ctx, ListFilter{RepoID: "nonexistent"})

		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

func TestStore_Persistence(t *testing.T) {
	ctx := context.Background()

	t.Run("persists entries across store instances", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "catalog.json")

		// First store instance
		store1 := NewStore(path)
		require.NoError(t, store1.Add(ctx, Entry{ID: "abc123", RepoID: "myrepo", Branch: "main"}))

		// Second store instance (same path)
		store2 := NewStore(path)
		got, err := store2.Get(ctx, "abc123")

		require.NoError(t, err)
		assert.Equal(t, "abc123", got.ID)
	})
}

func TestStore_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()

	t.Run("handles concurrent reads", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), "catalog.json"))
		require.NoError(t, store.Add(ctx, Entry{ID: "abc123", RepoID: "myrepo", Branch: "main"}))

		var wg sync.WaitGroup
		errs := make(chan error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := store.Get(ctx, "abc123")
				if err != nil {
					errs <- err
				}
			}()
		}

		wg.Wait()
		close(errs)

		for err := range errs {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("handles concurrent writes", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), "catalog.json"))

		var wg sync.WaitGroup
		successCount := 0
		var mu sync.Mutex

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				entry := Entry{
					ID:     fmt.Sprintf("entry-%d", idx),
					RepoID: fmt.Sprintf("repo-%d", idx),
					Branch: "main",
				}
				if err := store.Add(ctx, entry); err == nil {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}(i)
		}

		wg.Wait()

		// All writes should succeed (different repo IDs)
		assert.Equal(t, 10, successCount)

		entries, err := store.List(ctx, ListFilter{})
		require.NoError(t, err)
		assert.Len(t, entries, 10)
	})
}

func TestStore_ContextCancellation(t *testing.T) {
	t.Run("respects context cancellation during lock acquisition", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), "catalog.json"))

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := store.Add(ctx, Entry{ID: "abc123"})

		assert.Error(t, err)
	})
}

func TestStore_Sessions(t *testing.T) {
	ctx := context.Background()

	t.Run("stores entry with sessions", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), "catalog.json"))

		now := time.Now()
		entry := Entry{
			ID:        "abc123",
			RepoID:    "myrepo",
			Branch:    "main",
			CreatedAt: now,
			Status:    StatusRunning,
			Sessions: []Session{
				{
					ID:            "sess-1",
					Name:          "happy-panda",
					Type:          SessionTypeClaude,
					MuxSessionID: "hjk-abc123-sess-1",
					CreatedAt:     now,
					LastAccessed:  now,
				},
			},
		}
		require.NoError(t, store.Add(ctx, entry))

		got, err := store.Get(ctx, "abc123")
		require.NoError(t, err)
		require.Len(t, got.Sessions, 1)
		assert.Equal(t, "sess-1", got.Sessions[0].ID)
		assert.Equal(t, "happy-panda", got.Sessions[0].Name)
		assert.Equal(t, SessionTypeClaude, got.Sessions[0].Type)
		assert.Equal(t, "hjk-abc123-sess-1", got.Sessions[0].MuxSessionID)
	})

	t.Run("updates entry with modified sessions", func(t *testing.T) {
		store := NewStore(filepath.Join(t.TempDir(), "catalog.json"))

		now := time.Now()
		entry := Entry{
			ID:       "abc123",
			RepoID:   "myrepo",
			Branch:   "main",
			Sessions: []Session{},
		}
		require.NoError(t, store.Add(ctx, entry))

		// Add a session
		entry.Sessions = append(entry.Sessions, Session{
			ID:            "sess-1",
			Name:          "clever-wolf",
			Type:          SessionTypeShell,
			MuxSessionID: "hjk-abc123-sess-1",
			CreatedAt:     now,
			LastAccessed:  now,
		})
		require.NoError(t, store.Update(ctx, entry))

		got, err := store.Get(ctx, "abc123")
		require.NoError(t, err)
		require.Len(t, got.Sessions, 1)
		assert.Equal(t, "clever-wolf", got.Sessions[0].Name)
		assert.Equal(t, SessionTypeShell, got.Sessions[0].Type)
	})

	t.Run("persists multiple sessions", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "catalog.json")
		store := NewStore(path)

		now := time.Now()
		entry := Entry{
			ID:     "abc123",
			RepoID: "myrepo",
			Branch: "main",
			Sessions: []Session{
				{ID: "sess-1", Name: "happy-panda", Type: SessionTypeClaude, CreatedAt: now, LastAccessed: now},
				{ID: "sess-2", Name: "clever-wolf", Type: SessionTypeShell, CreatedAt: now, LastAccessed: now},
				{ID: "sess-3", Name: "swift-eagle", Type: SessionTypeGemini, CreatedAt: now, LastAccessed: now},
			},
		}
		require.NoError(t, store.Add(ctx, entry))

		// Reload from disk
		store2 := NewStore(path)
		got, err := store2.Get(ctx, "abc123")
		require.NoError(t, err)
		require.Len(t, got.Sessions, 3)
	})
}

func TestStore_Migration(t *testing.T) {
	ctx := context.Background()

	t.Run("migrates v1 catalog to v2", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "catalog.json")

		// Write a v1 catalog directly (without sessions field)
		v1Catalog := `{
			"version": 1,
			"entries": [
				{
					"id": "abc123",
					"repo": "/path/to/repo",
					"repo_id": "myrepo-abc123",
					"branch": "main",
					"worktree": "/path/to/worktree",
					"container_id": "container-xyz",
					"created_at": "2025-01-01T00:00:00Z",
					"status": "running"
				}
			]
		}`
		require.NoError(t, os.WriteFile(path, []byte(v1Catalog), 0644))

		// Load with the store - should migrate automatically
		store := NewStore(path)
		got, err := store.Get(ctx, "abc123")
		require.NoError(t, err)

		// Sessions should be initialized to empty slice after migration
		assert.NotNil(t, got.Sessions)
		assert.Empty(t, got.Sessions)
	})

	t.Run("persists migrated catalog with new version", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "catalog.json")

		// Write a v1 catalog
		v1Catalog := `{
			"version": 1,
			"entries": [{"id": "abc123", "repo_id": "myrepo", "branch": "main", "status": "running"}]
		}`
		require.NoError(t, os.WriteFile(path, []byte(v1Catalog), 0644))

		// Load and modify to trigger save
		store := NewStore(path)
		entry, err := store.Get(ctx, "abc123")
		require.NoError(t, err)

		entry.Status = StatusStopped
		require.NoError(t, store.Update(ctx, *entry))

		// Read raw file to check version
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"version": 2`)
	})

	t.Run("handles empty v1 sessions gracefully", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "catalog.json")

		// V1 catalog with null sessions (which JSON would produce for missing field)
		v1Catalog := `{
			"version": 1,
			"entries": [
				{"id": "a", "repo_id": "repo1", "branch": "main"},
				{"id": "b", "repo_id": "repo2", "branch": "dev"}
			]
		}`
		require.NoError(t, os.WriteFile(path, []byte(v1Catalog), 0644))

		store := NewStore(path)
		entries, err := store.List(ctx, ListFilter{})
		require.NoError(t, err)
		require.Len(t, entries, 2)

		for _, entry := range entries {
			assert.NotNil(t, entry.Sessions, "Sessions should not be nil after migration")
		}
	})
}

func TestSessionType_Values(t *testing.T) {
	// Verify session type constants have expected values
	assert.Equal(t, SessionType("shell"), SessionTypeShell)
	assert.Equal(t, SessionType("claude"), SessionTypeClaude)
	assert.Equal(t, SessionType("gemini"), SessionTypeGemini)
	assert.Equal(t, SessionType("codex"), SessionTypeCodex)
}

var _ = fmt.Sprintf // use fmt package
