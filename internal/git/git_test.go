package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jmgilman/headjack/internal/exec"
)

// resolvePath resolves symlinks in a path (handles macOS /var -> /private/var).
func resolvePath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return resolved
}

// testRepo creates a git repository in a temp directory for testing.
// Returns the repo path and a cleanup function.
func testRepo(t *testing.T) string {
	t.Helper()

	dir := resolvePath(t, t.TempDir())
	e := exec.New()
	ctx := context.Background()

	// Initialize repo
	_, err := e.Run(ctx, exec.RunOptions{
		Name: "git",
		Args: []string{"init"},
		Dir:  dir,
	})
	require.NoError(t, err, "git init")

	// Configure git user (required for commits)
	_, err = e.Run(ctx, exec.RunOptions{
		Name: "git",
		Args: []string{"config", "user.email", "test@test.com"},
		Dir:  dir,
	})
	require.NoError(t, err, "git config email")

	_, err = e.Run(ctx, exec.RunOptions{
		Name: "git",
		Args: []string{"config", "user.name", "Test User"},
		Dir:  dir,
	})
	require.NoError(t, err, "git config name")

	// Create initial commit
	testFile := filepath.Join(dir, "README.md")
	err = os.WriteFile(testFile, []byte("# Test Repo\n"), 0644)
	require.NoError(t, err, "create test file")

	_, err = e.Run(ctx, exec.RunOptions{
		Name: "git",
		Args: []string{"add", "."},
		Dir:  dir,
	})
	require.NoError(t, err, "git add")

	_, err = e.Run(ctx, exec.RunOptions{
		Name: "git",
		Args: []string{"commit", "-m", "initial commit"},
		Dir:  dir,
	})
	require.NoError(t, err, "git commit")

	return dir
}

// createBranch creates a branch in the given repo.
func createBranch(t *testing.T, repoDir, branch string) {
	t.Helper()

	e := exec.New()
	_, err := e.Run(context.Background(), exec.RunOptions{
		Name: "git",
		Args: []string{"branch", branch},
		Dir:  repoDir,
	})
	require.NoError(t, err, "create branch %s", branch)
}

func TestOpener_Open(t *testing.T) {
	e := exec.New()
	opener := NewOpener(e)
	ctx := context.Background()

	t.Run("opens valid repository", func(t *testing.T) {
		repoDir := testRepo(t)

		repo, err := opener.Open(ctx, repoDir)

		require.NoError(t, err)
		assert.Equal(t, repoDir, repo.Root())
		assert.NotEmpty(t, repo.Identifier())
	})

	t.Run("opens repository from subdirectory", func(t *testing.T) {
		repoDir := testRepo(t)
		subDir := filepath.Join(repoDir, "subdir")
		require.NoError(t, os.MkdirAll(subDir, 0755))

		repo, err := opener.Open(ctx, subDir)

		require.NoError(t, err)
		// Both paths should resolve to the same location
		assert.Equal(t, resolvePath(t, repoDir), resolvePath(t, repo.Root()))
	})

	t.Run("returns error for non-repository", func(t *testing.T) {
		nonRepoDir := t.TempDir()

		_, err := opener.Open(ctx, nonRepoDir)

		assert.ErrorIs(t, err, ErrNotRepository)
	})

	t.Run("identifier has correct format", func(t *testing.T) {
		repoDir := testRepo(t)

		repo, err := opener.Open(ctx, repoDir)

		require.NoError(t, err)
		id := repo.Identifier()
		// Should be "<dirname>-<7char hash>"
		assert.Regexp(t, `^[^-]+-[a-f0-9]{7}$`, id)
		// Should start with the directory name
		assert.True(t, len(id) > 8, "identifier should be longer than 8 chars")
	})
}

func TestRepository_BranchExists(t *testing.T) {
	e := exec.New()
	opener := NewOpener(e)
	ctx := context.Background()

	t.Run("returns true for existing local branch", func(t *testing.T) {
		repoDir := testRepo(t)
		createBranch(t, repoDir, "feature-branch")
		repo, err := opener.Open(ctx, repoDir)
		require.NoError(t, err)

		exists, err := repo.BranchExists(ctx, "feature-branch")

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns true for default branch", func(t *testing.T) {
		repoDir := testRepo(t)
		repo, err := opener.Open(ctx, repoDir)
		require.NoError(t, err)

		// Check for master or main (depends on git version/config)
		existsMaster, _ := repo.BranchExists(ctx, "master")
		existsMain, _ := repo.BranchExists(ctx, "main")

		assert.True(t, existsMaster || existsMain, "default branch should exist")
	})

	t.Run("returns false for non-existent branch", func(t *testing.T) {
		repoDir := testRepo(t)
		repo, err := opener.Open(ctx, repoDir)
		require.NoError(t, err)

		exists, err := repo.BranchExists(ctx, "nonexistent-branch")

		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestRepository_CreateWorktree(t *testing.T) {
	e := exec.New()
	opener := NewOpener(e)
	ctx := context.Background()

	t.Run("creates worktree for existing branch", func(t *testing.T) {
		repoDir := testRepo(t)
		createBranch(t, repoDir, "existing-branch")
		repo, err := opener.Open(ctx, repoDir)
		require.NoError(t, err)

		worktreePath := filepath.Join(resolvePath(t, t.TempDir()), "worktree")
		err = repo.CreateWorktree(ctx, worktreePath, "existing-branch")

		require.NoError(t, err)
		assert.DirExists(t, worktreePath)
		assert.FileExists(t, filepath.Join(worktreePath, "README.md"))
	})

	t.Run("creates worktree with new branch", func(t *testing.T) {
		repoDir := testRepo(t)
		repo, err := opener.Open(ctx, repoDir)
		require.NoError(t, err)

		worktreePath := filepath.Join(resolvePath(t, t.TempDir()), "worktree")
		err = repo.CreateWorktree(ctx, worktreePath, "new-branch")

		require.NoError(t, err)
		assert.DirExists(t, worktreePath)

		// Verify the new branch was created
		exists, err := repo.BranchExists(ctx, "new-branch")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns error for existing worktree path", func(t *testing.T) {
		repoDir := testRepo(t)
		repo, err := opener.Open(ctx, repoDir)
		require.NoError(t, err)

		worktreePath := filepath.Join(resolvePath(t, t.TempDir()), "worktree")
		err = repo.CreateWorktree(ctx, worktreePath, "branch1")
		require.NoError(t, err)

		// Try to create another worktree at the same path
		err = repo.CreateWorktree(ctx, worktreePath, "branch2")

		assert.Error(t, err)
	})
}

func TestRepository_RemoveWorktree(t *testing.T) {
	e := exec.New()
	opener := NewOpener(e)
	ctx := context.Background()

	t.Run("removes existing worktree", func(t *testing.T) {
		repoDir := testRepo(t)
		repo, err := opener.Open(ctx, repoDir)
		require.NoError(t, err)

		worktreePath := filepath.Join(resolvePath(t, t.TempDir()), "worktree")
		err = repo.CreateWorktree(ctx, worktreePath, "test-branch")
		require.NoError(t, err)
		require.DirExists(t, worktreePath)

		err = repo.RemoveWorktree(ctx, worktreePath)

		require.NoError(t, err)
		assert.NoDirExists(t, worktreePath)
	})

	t.Run("returns error for non-existent worktree", func(t *testing.T) {
		repoDir := testRepo(t)
		repo, err := opener.Open(ctx, repoDir)
		require.NoError(t, err)

		err = repo.RemoveWorktree(ctx, "/nonexistent/path")

		assert.ErrorIs(t, err, ErrWorktreeNotFound)
	})
}

func TestRepository_ListWorktrees(t *testing.T) {
	e := exec.New()
	opener := NewOpener(e)
	ctx := context.Background()

	t.Run("lists main worktree", func(t *testing.T) {
		repoDir := testRepo(t)
		repo, err := opener.Open(ctx, repoDir)
		require.NoError(t, err)

		worktrees, err := repo.ListWorktrees(ctx)

		require.NoError(t, err)
		require.Len(t, worktrees, 1)
		// Compare resolved paths to handle symlinks
		assert.Equal(t, resolvePath(t, repoDir), resolvePath(t, worktrees[0].Path))
	})

	t.Run("lists multiple worktrees", func(t *testing.T) {
		repoDir := testRepo(t)
		repo, err := opener.Open(ctx, repoDir)
		require.NoError(t, err)

		worktree1 := filepath.Join(resolvePath(t, t.TempDir()), "wt1")
		worktree2 := filepath.Join(resolvePath(t, t.TempDir()), "wt2")
		require.NoError(t, repo.CreateWorktree(ctx, worktree1, "branch1"))
		require.NoError(t, repo.CreateWorktree(ctx, worktree2, "branch2"))

		worktrees, err := repo.ListWorktrees(ctx)

		require.NoError(t, err)
		assert.Len(t, worktrees, 3) // main + 2 worktrees

		// Check that our worktrees are in the list
		paths := make([]string, len(worktrees))
		for i, wt := range worktrees {
			paths[i] = wt.Path
		}
		assert.Contains(t, paths, worktree1)
		assert.Contains(t, paths, worktree2)
	})
}

func TestRepository_WorktreeForBranch(t *testing.T) {
	e := exec.New()
	opener := NewOpener(e)
	ctx := context.Background()

	t.Run("returns path for branch with worktree", func(t *testing.T) {
		repoDir := testRepo(t)
		repo, err := opener.Open(ctx, repoDir)
		require.NoError(t, err)

		worktreePath := filepath.Join(resolvePath(t, t.TempDir()), "worktree")
		err = repo.CreateWorktree(ctx, worktreePath, "test-branch")
		require.NoError(t, err)

		path, err := repo.WorktreeForBranch(ctx, "test-branch")

		require.NoError(t, err)
		assert.Equal(t, worktreePath, path)
	})

	t.Run("returns empty string for branch without worktree", func(t *testing.T) {
		repoDir := testRepo(t)
		createBranch(t, repoDir, "no-worktree-branch")
		repo, err := opener.Open(ctx, repoDir)
		require.NoError(t, err)

		path, err := repo.WorktreeForBranch(ctx, "no-worktree-branch")

		require.NoError(t, err)
		assert.Empty(t, path)
	})
}

func TestParseWorktreeList(t *testing.T) {
	t.Run("parses single worktree", func(t *testing.T) {
		input := `worktree /path/to/repo
HEAD abc123def456
branch refs/heads/main

`
		worktrees, err := parseWorktreeList(input)

		require.NoError(t, err)
		require.Len(t, worktrees, 1)
		assert.Equal(t, "/path/to/repo", worktrees[0].Path)
		assert.Equal(t, "main", worktrees[0].Branch)
		assert.False(t, worktrees[0].Bare)
	})

	t.Run("parses multiple worktrees", func(t *testing.T) {
		input := `worktree /path/to/main
HEAD abc123
branch refs/heads/main

worktree /path/to/feature
HEAD def456
branch refs/heads/feature

`
		worktrees, err := parseWorktreeList(input)

		require.NoError(t, err)
		require.Len(t, worktrees, 2)
		assert.Equal(t, "main", worktrees[0].Branch)
		assert.Equal(t, "feature", worktrees[1].Branch)
	})

	t.Run("parses bare worktree", func(t *testing.T) {
		input := `worktree /path/to/bare
HEAD abc123
bare

`
		worktrees, err := parseWorktreeList(input)

		require.NoError(t, err)
		require.Len(t, worktrees, 1)
		assert.True(t, worktrees[0].Bare)
		assert.Empty(t, worktrees[0].Branch)
	})

	t.Run("handles empty input", func(t *testing.T) {
		worktrees, err := parseWorktreeList("")

		require.NoError(t, err)
		assert.Empty(t, worktrees)
	})
}
