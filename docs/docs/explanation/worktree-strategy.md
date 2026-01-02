---
sidebar_position: 4
title: Worktree Strategy
description: How git worktrees enable parallel work without branch switching
---

# Worktree Strategy

Headjack uses git worktrees to enable multiple agents to work on the same repository simultaneously, each on a different branch. This approach solves a fundamental problem: how do you run parallel development workflows without creating chaos?

## The Problem with Clones

The naive approach to parallel development is to clone the repository multiple times:

```
~/projects/
├── myapp/              # Clone 1: working on main
├── myapp-feature-a/    # Clone 2: feature A
├── myapp-feature-b/    # Clone 3: feature B
└── myapp-bugfix/       # Clone 4: bug fix
```

This works, but creates problems:

- **Disk space**: Each clone duplicates the entire git history (hundreds of MB or GB for large repos)
- **Synchronization**: Changes pushed to origin aren't visible in other clones until you fetch
- **Mental overhead**: Which clone has my latest changes? Where did I stash that work?
- **Tooling confusion**: IDEs, git GUIs, and scripts get confused by multiple clones

For human developers, these problems are manageable. For autonomous agents running in parallel, they become severe. An agent in one clone might not see work from another agent. Conflicting changes might be pushed without coordination.

## Git Worktrees

Git worktrees solve this elegantly. A worktree is an additional working directory linked to a single repository:

```
~/.local/share/headjack/git/
└── abc123/                    # Repository (identified by hash of remote URL)
    ├── feature-a/            # Worktree for feature-a branch
    │   └── .git              # (file pointing to main repo)
    ├── feature-b/            # Worktree for feature-b branch
    │   └── .git              # (file pointing to main repo)
    └── bugfix/               # Worktree for bugfix branch
        └── .git              # (file pointing to main repo)

~/projects/myapp/             # Your original clone (main branch)
└── .git/                     # The actual repository
    └── worktrees/            # Git's worktree metadata
        ├── feature-a
        ├── feature-b
        └── bugfix
```

All worktrees share:

- **Git history**: The object database (commits, trees, blobs) exists once
- **Remote configuration**: Push/pull targets the same remotes
- **Refs**: All branches, tags, and refs are shared

Each worktree has its own:

- **Working directory**: Files on disk for that specific branch
- **Index**: Staging area for that worktree
- **HEAD**: Currently checked-out commit/branch

## Why Headjack Manages Worktrees

Headjack creates and manages worktrees rather than asking users to set them up. This provides:

### Consistent Location

All worktrees live under `~/.local/share/headjack/git/<repo-id>/<branch>`. This predictable structure:

- Simplifies container mount configuration
- Enables cleanup without hunting for scattered directories
- Keeps your project directories clean

### Lifecycle Management

When you remove an instance, Headjack removes the worktree:

```go
// From manager.go
if entry.Worktree != "" {
    repo, err := m.git.Open(ctx, entry.Repo)
    if err == nil {
        repo.RemoveWorktree(ctx, entry.Worktree)
    }
}
```

Without automatic cleanup, abandoned worktrees would accumulate over time.

### Branch Tracking

Headjack tracks the relationship between worktrees, branches, and instances in its catalog. This enables commands like:

```bash
# Attach by branch name (not worktree path)
hjk attach feature-a

# See all instances across repositories
hjk list
```

## One Branch, One Worktree

Git enforces a constraint: a branch can only be checked out in one worktree at a time. Attempting to check out an already-checked-out branch fails:

```
fatal: 'feature-a' is already checked out at '/path/to/worktree'
```

Headjack embraces this constraint. Each instance gets exactly one branch. This means:

- **No confusion**: The agent working on `feature-a` is the only process with `feature-a` checked out
- **Clear ownership**: Changes to `feature-a` must have come from that instance's agent
- **Natural coordination**: Want two agents on the same branch? You can't. Create a new branch.

This constraint guides users toward good practices. If you need two agents working on related changes, create two feature branches and merge them later.

## How Worktree Creation Works

When you run `hjk run feature-branch`, Headjack's git integration:

1. **Checks if the branch exists** (locally or remotely)
2. **Creates the worktree** with the appropriate command:

```go
// If branch exists
args = []string{"worktree", "add", path, branch}

// If branch doesn't exist (create from HEAD)
args = []string{"worktree", "add", "-b", branch, path}
```

3. **Handles edge cases**:
   - Branch already checked out elsewhere (returns error)
   - Worktree path already exists (returns error)
   - Remote branch exists but not local (fetches and tracks)

## Disk Space Efficiency

For a large repository, worktrees dramatically reduce disk usage:

| Approach | 10 Active Branches |
|----------|-------------------|
| Full clones | 10x repository size |
| Worktrees | 1x repository size + 10x working tree size |

Working tree size is typically much smaller than the git history, especially for repositories with long histories or large binary assets tracked via LFS.

## Worktrees and Containers

The worktree becomes the bridge between host and container:

```go
// From manager.go
Mounts: []container.Mount{
    {Source: worktreePath, Target: "/workspace", ReadOnly: false},
}
```

The container sees the worktree mounted at `/workspace`. From the agent's perspective, it's working in a normal git repository. All standard git operations work:

```bash
# Inside container
cd /workspace
git status
git add .
git commit -m "Changes from agent"
git push
```

The mount is read-write, allowing the agent to modify files. Changes persist because the worktree exists on the host filesystem.

## Worktree Limitations

Some git operations have caveats with worktrees:

### Submodules

Git submodules can be tricky with worktrees. Each worktree needs its own submodule checkout, and synchronization can be confusing. For repositories with complex submodule setups, you may need to run `git submodule update` in each worktree.

### Stashes

`git stash` is per-worktree, not shared. A stash created in one worktree isn't visible in others. This is usually desirable for isolation but can surprise users expecting repository-wide stashes.

### Hooks

Git hooks (`.git/hooks/`) are shared across all worktrees since they live in the main repository. This means pre-commit hooks, etc., apply everywhere. Worktree-specific hook configuration isn't supported.

## Alternatives Considered

### Full Clones

Could have each instance clone the repository independently. Rejected due to disk space and synchronization concerns described above.

### Shallow Clones

Shallow clones (`git clone --depth=1`) reduce disk usage but break many git operations. Agents need full history for `git log`, `git blame`, and other context-gathering commands.

### Sparse Checkouts

Git sparse checkout allows checking out only specific directories. This is orthogonal to worktrees and could potentially be combined with them for very large monorepos. Not currently implemented but remains a future option.

### Working Directory Copies

Could copy the working directory without git history. Rejected because agents need git operations (status, diff, commit, push) to function effectively.

## Summary

Git worktrees enable Headjack's core value proposition: multiple agents working on the same repository simultaneously, each on its own branch, with efficient disk usage and clear isolation boundaries. The constraint of one branch per worktree becomes a feature, preventing the chaos that could arise from multiple agents modifying the same branch.

## Related

- [Architecture Overview](./architecture) - How worktrees fit into instances
- [Session Lifecycle](./session-lifecycle) - What happens inside the worktree's container
