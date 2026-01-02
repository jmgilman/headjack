---
sidebar_position: 2
title: Your First Coding Task
description: Walk through giving an agent a real coding task, observing its work, and reviewing the results
---

# Your First Coding Task

In this tutorial, we will give a Claude agent a real coding task, observe how it works through the problem, and review the results. By the end, you will understand how to effectively delegate work to agents and evaluate their output.

This tutorial takes approximately 20-30 minutes to complete.

## Prerequisites

Before starting, ensure you have:

- Completed the [Getting Started](./getting-started) tutorial
- A git repository with some existing code to work on
- Claude Code authentication configured (`hjk auth claude`)

## Step 1: Choose a Task

We need a concrete task for our agent. Good first tasks are:

- Adding a small feature with clear requirements
- Writing tests for existing code
- Refactoring a specific function or module
- Fixing a bug with a known location

For this tutorial, we will ask Claude to add input validation to a function. Navigate to your project:

```bash
cd ~/projects/my-app
```

## Step 2: Write an Effective Prompt

The quality of your prompt directly affects the agent's output. A good prompt includes:

- **Context**: What part of the codebase to work in
- **Goal**: What you want to achieve
- **Constraints**: Any requirements or limitations

Here is our example prompt:

```
Add input validation to the createUser function in src/users.js.
Validate that email is a valid format and name is non-empty.
Throw descriptive errors for invalid input.
Add unit tests for the validation logic.
```

## Step 3: Launch the Agent

Create an instance and start the agent with your task:

```bash
hjk run feat/user-validation --agent claude "Add input validation to the createUser function in src/users.js. Validate that email is a valid format and name is non-empty. Throw descriptive errors for invalid input. Add unit tests for the validation logic."
```

You will see output indicating the instance was created:

```
Created instance abc123 for branch feat/user-validation
```

Then your terminal attaches to the Claude session. Watch as Claude begins working.

## Step 4: Observe the Agent Working

Claude will start by analyzing your codebase. You will see it:

1. **Read the target file** to understand the existing code
2. **Explore related files** to understand the project structure
3. **Plan its approach** before making changes
4. **Implement the changes** incrementally
5. **Write tests** to verify the implementation

As Claude works, it may ask clarifying questions. For example:

```
I see there's an existing validation utility in src/utils/validate.js.
Should I use that for consistency, or create new validation logic
specific to user creation?
```

Answer questions to guide the agent toward your preferred solution.

## Step 5: Detach and Monitor

If Claude is working independently and you want to do other work, detach from the session:

```
Ctrl+B, then d
```

You return to your host terminal. The agent continues working in the background.

To monitor progress, first get the session name:

```bash
hjk ps feat/user-validation
```

Output:

```
SESSION       TYPE    STATUS    CREATED   ACCESSED
happy-panda   claude  detached  5m ago    just now
```

Then view logs for that session:

```bash
hjk logs feat/user-validation happy-panda
```

To follow the output in real-time:

```bash
hjk logs feat/user-validation happy-panda -f
```

Press `Ctrl+C` to stop following.

## Step 6: Reattach and Review

When you are ready to check on Claude directly, reattach:

```bash
hjk attach feat/user-validation
```

If Claude has finished, you will see a summary of what it accomplished. If it is still working, you can watch it continue or provide additional guidance.

## Step 7: Review the Changes

Once Claude indicates it has completed the task, review what it produced. Detach from the session if attached:

```
Ctrl+B, then d
```

The changes exist in the git worktree for the feature branch. You can review them using standard git tools:

```bash
# Switch to the feature branch in your main repository
git checkout feat/user-validation

# See what files changed
git status

# Review the diff
git diff HEAD~1

# Run the tests Claude wrote
npm test
```

Alternatively, start a shell session in the same instance to explore:

```bash
hjk run feat/user-validation --name review-shell
```

This opens a shell in the same container where the agent worked. You can run commands, inspect files, and verify the changes.

## Step 8: Iterate if Needed

If the changes need adjustments, you have several options:

**Option A: Give Claude follow-up instructions**

Reattach and provide feedback:

```bash
hjk attach feat/user-validation
```

Then type your feedback directly:

```
The email validation looks good, but please also check for maximum
length (255 characters) and add a test case for that.
```

**Option B: Start a new session with corrections**

```bash
hjk run feat/user-validation --agent claude "The email validation in createUser needs one adjustment: also validate maximum length of 255 characters. Add a test for this case."
```

## Step 9: Commit the Work

When you are satisfied with the changes, commit them. You can do this from your host terminal after checking out the branch, or ask Claude to commit:

```bash
hjk attach feat/user-validation
```

Then:

```
Please commit these changes with an appropriate commit message.
```

Or commit manually:

```bash
git checkout feat/user-validation
git add -A
git commit -m "feat(users): add input validation to createUser"
```

## Step 10: Clean Up

Stop the instance when you are finished:

```bash
hjk stop feat/user-validation
```

The worktree and branch remain for future work. If you want to remove everything:

```bash
hjk rm feat/user-validation
```

## What We Learned

In this tutorial, we:

- Wrote an effective prompt with context, goals, and constraints
- Launched an agent with a real coding task
- Observed how Claude analyzes code and implements changes
- Monitored background work using logs
- Reviewed changes in the git worktree
- Iterated on the results with follow-up instructions

The key insight is that agents work best with clear, specific instructions. Vague prompts produce vague results. The more context you provide upfront, the less back-and-forth you will need.

## Next Steps

Now that you can delegate individual tasks, explore these resources:

**How-To Guides**
- [Manage Sessions](../how-to/manage-sessions) - Advanced session management techniques
- [Stop and Clean Up Instances](../how-to/stop-cleanup) - Instance lifecycle management
