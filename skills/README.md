# Skills Catalog

This repository is part of the The Synthetic Engineer system.

This index reflects the current folders under `skills/`.
Use it to pick the narrowest matching skill, then open that skill's `SKILL.md`.

## How to use this index

1. Start with the closest domain match.
2. Prefer narrow skills over broad fallback skills.
3. Combine skills when a task spans architecture, implementation, testing, and security.
4. Treat each skill's `SKILL.md` and frontmatter as the source of truth.

## Remote Skill Retrieval

If a needed skill or reference file is not present in the current workspace but exists in a GitHub repository such as `JOHNEPPILLAR/the-synth-eng-skills` or `JOHNEPPILLAR/the-synth-eng-code-ref`, fetch it through the Pandora's Box MCP server instead of browsing raw GitHub pages.

Priority:

1. use local workspace files when the repo is checked out locally
2. otherwise use Pandora's Box `get_github_file_content` for exact files and `search_github_repo_patterns` for pattern discovery
3. reserve `context7` for library/framework docs rather than repo-owned skill or example content

## Current Skills

| Skill | Use for | Common triggers |
| --- | --- | --- |
| `git-worktree` | Parallel isolated work using Git worktrees | worktree, parallel branch, isolated refactor |
| `memory-management` | Durable repo-memory boundaries and update policies | memory update, durable notes, session vs repo memory |
| `multi-model-review` | Consensus-based consolidation of multi-review findings | consensus scoring, merge reviewer outputs, conflict triage |
| `planning-structure` | Planning-track selection, decomposition, and readiness gates | quick change, feature track, system track, plan delta |
| `research-discovery` | Fast broad-to-narrow read-only discovery | codebase scouting, owner mapping, discovery passes |
| `review-core` | Shared independent-review output contract | blocker/warning/suggestion format, actionable findings |
| `review-orchestration` | Review routing policy and post-implementation gates | single vs multi review, skip rules, fix loop |

## Quick Routing Hints

- Planning and decomposition: `planning-structure`
- Discovery and owner/file-scope scouting: `research-discovery`
- Durable vs session memory governance: `memory-management`
- Independent review contract: `review-core`
- Review routing and fix loops: `review-orchestration`
- Multi-model consensus consolidation: `multi-model-review`
- Parallel isolated change streams: `git-worktree`

## Selection Rules

1. Use the most specific skill that matches the task.
2. Combine planning + discovery skills for ambiguous requests.
3. Combine review skills (`review-core`, `review-orchestration`, `multi-model-review`) when review depth increases.
4. If uncertain, start narrow, then layer in one broad fallback.

## New Skill File Routing Block (Required)

For every new `skills/<name>/SKILL.md`, include a short routing block near the top.
This gives agents a fast, explicit invoke signal.

Example:

```md
## When to invoke this skill

Invoke when:
- You are implementing or reviewing <domain/task type>.
- You need <specific constraints or architecture focus>.

Do not invoke when:
- The task is primarily <unrelated domain>.
- A narrower skill already fully covers the request.

Quick triggers: keyword1, keyword2, keyword3
```
