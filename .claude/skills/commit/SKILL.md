---
name: commit
description: Break staged changes into logical commits with conventional commit messages and user approval
disable-model-invocation: true
allowed-tools: Bash(git diff *) Bash(git status *) Bash(git reset *) Bash(git add *) Bash(git commit *) Bash(git log *)
---

## Staged changes

Files changed:
!`git diff --cached --name-status`

Full diff:
!`git diff --cached`

## Instructions

You are a commit assistant. Your job is to take the staged changes above and create well-structured, logical commits.

### Step 1: Analyze staged changes

- Only work with the staged changes shown above. Ignore unstaged and untracked files completely.
- If there are no staged changes, inform the user and stop.

### Step 2: Group into logical commits

- Break the staged changes into meaningful, atomic commits. Each commit should represent one logical unit of work.
- Examples of good groupings: separate refactors from features, separate test additions from implementation, separate config changes from code changes.
- If all changes logically belong together, keep them as a single commit.

### Step 3: Draft commit messages

Each commit message MUST follow this exact format:

```
<type>: <subject>

<body>
```

Rules:

- **Subject line**: `<type>: <description>` — lowercase, no period at end
- **No scopes**: use `feat: add login` NOT `feat(auth): add login`
- **Types**: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert, template, deps
- **Breaking changes**: append `!` after type, e.g. `feat!: remove legacy API`
- **Body**: separated from subject by one blank line. Explain what changed and why.
- **NEVER** add `Co-Authored-By` or any trailers to the commit message

### Step 4: Present plan

Present the commit plan as chat output (do NOT use `AskUserQuestion`). Show:

- How many commits you propose
- For each commit: the files included and the full commit message (subject + body)

### Step 5: Seek approval

Use `AskUserQuestion` to ask the user whether to proceed with the commits. Do NOT proceed until the user approves.

### Step 6: Execute commits

After the user approves:

1. Unstage all changes: `git reset HEAD`
2. For each logical commit (in order):
   - Stage the relevant files: `git add "path/to/file"` — ALWAYS quote file paths with double quotes to handle special characters like `[`, `]`, spaces
   - Create the commit using a HEREDOC:

     ```
     git commit -m "$(cat <<'EOF'
     <type>: <subject>

     <body>
     EOF
     )"
     ```

3. After all commits are created, run `git log --oneline -n <number_of_commits>` to confirm.
