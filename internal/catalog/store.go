package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

const (
	lockTimeout    = 5 * time.Second
	fileMode       = 0o644
	dirMode        = 0o755
	currentVersion = 2 // Bump when schema changes
)

// catalogFile represents the on-disk catalog format.
type catalogFile struct {
	Version int     `json:"version"`
	Entries []Entry `json:"entries"`
}

type jsonStore struct {
	path string
	mu   sync.RWMutex
}

// NewStore creates a new JSON-backed catalog store.
func NewStore(path string) Store {
	return &jsonStore{path: path}
}

func (s *jsonStore) Add(ctx context.Context, entry *Entry) error {
	return s.withExclusiveLock(ctx, func(cf *catalogFile) error {
		// Check for duplicate
		for i := range cf.Entries {
			if cf.Entries[i].RepoID == entry.RepoID && cf.Entries[i].Branch == entry.Branch {
				return ErrAlreadyExists
			}
		}

		cf.Entries = append(cf.Entries, *entry)
		return nil
	})
}

func (s *jsonStore) Get(ctx context.Context, id string) (*Entry, error) {
	var result *Entry

	err := s.withSharedLock(ctx, func(cf *catalogFile) error {
		for i := range cf.Entries {
			if cf.Entries[i].ID == id {
				entry := cf.Entries[i]
				result = &entry
				return nil
			}
		}
		return ErrNotFound
	})

	return result, err
}

func (s *jsonStore) GetByRepoBranch(ctx context.Context, repoID, branch string) (*Entry, error) {
	var result *Entry

	err := s.withSharedLock(ctx, func(cf *catalogFile) error {
		for i := range cf.Entries {
			if cf.Entries[i].RepoID == repoID && cf.Entries[i].Branch == branch {
				entry := cf.Entries[i]
				result = &entry
				return nil
			}
		}
		return ErrNotFound
	})

	return result, err
}

func (s *jsonStore) Update(ctx context.Context, entry *Entry) error {
	return s.withExclusiveLock(ctx, func(cf *catalogFile) error {
		for i := range cf.Entries {
			if cf.Entries[i].ID == entry.ID {
				cf.Entries[i] = *entry
				return nil
			}
		}
		return ErrNotFound
	})
}

func (s *jsonStore) Remove(ctx context.Context, id string) error {
	return s.withExclusiveLock(ctx, func(cf *catalogFile) error {
		for i := range cf.Entries {
			if cf.Entries[i].ID == id {
				cf.Entries = append(cf.Entries[:i], cf.Entries[i+1:]...)
				return nil
			}
		}
		return ErrNotFound
	})
}

func (s *jsonStore) List(ctx context.Context, filter ListFilter) ([]Entry, error) {
	var result []Entry

	err := s.withSharedLock(ctx, func(cf *catalogFile) error {
		for i := range cf.Entries {
			if filter.RepoID != "" && cf.Entries[i].RepoID != filter.RepoID {
				continue
			}
			if filter.Status != "" && cf.Entries[i].Status != filter.Status {
				continue
			}
			result = append(result, cf.Entries[i])
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// withSharedLock executes fn with a shared (read) lock.
func (s *jsonStore) withSharedLock(ctx context.Context, fn func(*catalogFile) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cf, file, err := s.openAndLock(ctx, false)
	if err != nil {
		return err
	}
	defer s.unlockAndClose(file)

	return fn(cf)
}

// withExclusiveLock executes fn with an exclusive (write) lock.
// Changes made by fn are persisted to disk.
func (s *jsonStore) withExclusiveLock(ctx context.Context, fn func(*catalogFile) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cf, file, err := s.openAndLock(ctx, true)
	if err != nil {
		return err
	}
	defer s.unlockAndClose(file)

	if err := fn(cf); err != nil {
		return err
	}

	return s.save(cf)
}

// openAndLock opens the catalog file and acquires a lock.
func (s *jsonStore) openAndLock(ctx context.Context, exclusive bool) (*catalogFile, *os.File, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.path), dirMode); err != nil {
		return nil, nil, fmt.Errorf("create catalog directory: %w", err)
	}

	// Open or create file
	file, err := os.OpenFile(s.path, os.O_RDWR|os.O_CREATE, fileMode)
	if err != nil {
		return nil, nil, fmt.Errorf("open catalog file: %w", err)
	}

	// Acquire lock with timeout
	lockType := syscall.LOCK_SH
	if exclusive {
		lockType = syscall.LOCK_EX
	}

	if lockErr := s.acquireLock(ctx, file, lockType); lockErr != nil {
		file.Close()
		return nil, nil, lockErr
	}

	// Load catalog
	cf, err := s.load(file)
	if err != nil {
		s.unlockAndClose(file)
		return nil, nil, err
	}

	return cf, file, nil
}

// acquireLock attempts to acquire a file lock with timeout.
func (s *jsonStore) acquireLock(ctx context.Context, file *os.File, lockType int) error {
	deadline := time.Now().Add(lockTimeout)

	for {
		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Try non-blocking lock
		err := syscall.Flock(int(file.Fd()), lockType|syscall.LOCK_NB)
		if err == nil {
			return nil
		}

		if err != syscall.EWOULDBLOCK {
			return fmt.Errorf("acquire file lock: %w", err)
		}

		// Check timeout
		if time.Now().After(deadline) {
			return ErrLockTimeout
		}

		// Wait and retry
		time.Sleep(10 * time.Millisecond)
	}
}

// unlockAndClose releases the lock and closes the file.
func (s *jsonStore) unlockAndClose(file *os.File) {
	//nolint:errcheck // Unlock errors are not actionable during cleanup
	syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	file.Close()
}

// load reads and parses the catalog file.
func (s *jsonStore) load(file *os.File) (*catalogFile, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat catalog file: %w", err)
	}

	// Empty file - return default
	if info.Size() == 0 {
		return &catalogFile{Version: currentVersion, Entries: []Entry{}}, nil
	}

	// Seek to beginning
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("seek catalog file: %w", err)
	}

	var cf catalogFile
	if err := json.NewDecoder(file).Decode(&cf); err != nil {
		return nil, fmt.Errorf("decode catalog file: %w", err)
	}

	// Migrate from older versions
	if cf.Version < currentVersion {
		s.migrate(&cf)
	}

	return &cf, nil
}

// migrate upgrades catalog data from older versions to the current version.
func (s *jsonStore) migrate(cf *catalogFile) {
	// Migration from v1 to v2: initialize Sessions field
	if cf.Version < 2 {
		for i := range cf.Entries {
			if cf.Entries[i].Sessions == nil {
				cf.Entries[i].Sessions = []Session{}
			}
		}
	}

	cf.Version = currentVersion
}

// save writes the catalog to disk atomically.
func (s *jsonStore) save(cf *catalogFile) error {
	cf.Version = currentVersion

	// Write to temp file
	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, "catalog-*.json.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	// Clean up on error
	defer func() {
		if tmpPath != "" {
			//nolint:errcheck // Cleanup errors are not actionable
			os.Remove(tmpPath)
		}
	}()

	encoder := json.NewEncoder(tmp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cf); err != nil {
		tmp.Close()
		return fmt.Errorf("encode catalog: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("rename catalog file: %w", err)
	}

	tmpPath = "" // Prevent cleanup
	return nil
}
