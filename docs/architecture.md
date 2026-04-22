# Architecture

This document explains what the The Synthetic Engineer core repo does, how the main parts fit together, and how work moves through the system.

## Purpose

The repository is a control plane for VS Code agent workflows. It provides:

- user-facing coordination through `Orchestrator` and `Planner`
- specialised hidden execution agents for coding, review, debugging, and verification
- reusable operational skills that define planning, review, memory, and worktree policy
- durable repo memory templates under `.agent-memory/`
- the `Pandora's Box` MCP server for external engineering context from Confluence and GitHub

The system is designed to keep responsibilities explicit:

- coordination is separated from implementation
- review is separated from authorship
- verification is separated from review
- durable memory is separated from transient session context

## System Context

```mermaid
flowchart LR
    User["User"] --> VSCode["VS Code + Copilot Chat"]
    VSCode --> Orchestrator["Orchestrator\nuser-facing control plane"]
    VSCode --> Planner["Planner\nuser-facing planning gate"]

    Orchestrator --> InternalAgents["Hidden Internal Agents\nExplore, SoftwareEngineer, PrincipalEngineer,\nDesigner, Debugger, Reviewer(s), MultiReviewer, Verifier"]
    Planner --> Explore["Explore\nread-only discovery"]

    InternalAgents --> Skills["skills/\nplanning, review, memory, worktree, discovery"]
    InternalAgents --> Memory[".agent-memory/\ndurable repo knowledge templates"]
    InternalAgents --> SessionMemory["VS Code session memory\ntransient plans and breadcrumbs"]

    InternalAgents --> PandorasBox["Pandora's Box MCP Server\nexternal engineering context"]
    PandorasBox --> GitHub["GitHub repos\nskills and code references"]
    PandorasBox --> Confluence["Confluence\nstandards and platform docs"]
```

## Core Components

### User-facing agents

- `Orchestrator`: the main manager. It triages requests, routes work, controls review and verification, and decides when memory or worktrees are needed.
- `Planner`: the planning gatekeeper. It handles ambiguity, discovery, decomposition, and execution readiness.

### Hidden internal agents

- `Explore`: read-only discovery for codebase mapping and pattern lookup.
- `SoftwareEngineer`: execution agent for smaller implementation tasks.
- `PrincipalEngineer`: execution agent for larger or riskier implementation tasks.
- `Designer`: UI and UX-only execution path.
- `Debugger`: minimal root-cause bug fixing for reproducible failures.
- `Reviewer`, `ReviewerGPT`, `ReviewerGemini`: independent review producers.
- `MultiReviewer`: consolidation layer for multi-model review.
- `Verifier`: objective acceptance gate based on commands, tests, builds, and smoke checks.

### Skills

The `skills/` directory holds operational policy and reusable workflow guidance rather than user-facing features. Current core skills cover:

- planning structure
- research and discovery
- review contracts and orchestration
- multi-model review consolidation
- durable memory governance
- git worktree strategy

### Durable memory

The repo commits `.agent-memory/` as reusable templates. The files define the durable memory shape, but downstream projects populate the project-specific entries.

### Pandora's Box MCP server

`mcp/pandoras-box` is a Go MCP server that exposes external engineering context to agents. It currently supports:

- environment inspection
- Confluence search and page retrieval
- GitHub repo listing
- GitHub file-content retrieval
- GitHub repository pattern search

This gives agents a structured alternative to raw web browsing for repo-owned skills, reference examples, and platform guidance.

## Repository Structure

```mermaid
flowchart TD
    Root["the-synth-eng-core/"] --> Agents["agents/\nagent role definitions"]
    Root --> Skills["skills/\noperational workflow skills"]
    Root --> MCP["mcp/pandoras-box/\nGo MCP server"]
    Root --> Memory[".agent-memory/\ndurable memory templates"]
    Root --> VSCode[".vscode/\nworkspace MCP config and local settings"]
    Root --> Docs["docs/\narchitecture and reference docs"]
    Root --> Readme["README.md\nproject overview"]

    Agents --> UserFacing["Orchestrator, Planner"]
    Agents --> Hidden["Explore, execution, review, debug, verify"]
    Skills --> Policy["planning, review, memory, worktree, discovery"]
    MCP --> PB["Pandora's Box tools"]
```

## Request Lifecycle

The system uses a staged flow rather than allowing every agent to do everything.

```mermaid
sequenceDiagram
    participant U as User
    participant O as Orchestrator
    participant P as Planner
    participant E as Explore
    participant X as Execution Agent
    participant R as Reviewer Path
    participant V as Verifier
    participant M as Memory

    U->>O: request
    O->>O: classify intent and risk
    alt clarification or architecture needed
        O->>P: route for planning
        P->>E: optional discovery
        E-->>P: findings
        P-->>O: execution-ready plan or blocked status
    else localised execution-ready task
        O->>X: direct execution routing
    end

    O->>X: implementation or debugging delegation
    X-->>O: code changes + verification notes
    O->>R: independent review
    R-->>O: findings or approval
    O->>V: objective verification
    V-->>O: pass or blocked
    O->>M: durable memory update decision when needed
    O-->>U: final outcome and risks
```

## Routing and Decision Model

The control plane is intentionally asymmetric:

- `Orchestrator` governs but does not write code
- `Planner` clarifies and decomposes but does not implement
- execution agents write code but should not own product ambiguity
- reviewers inspect but do not implement
- `Verifier` validates closure based on executed evidence, not opinion

This separation prevents a single prompt from quietly becoming planner, implementer, reviewer, and verifier all at once.

## Planning and Readiness

Planning has three explicit tracks:

- `Quick Change`: small, localised work
- `Feature Track`: medium complexity work with a few moving parts
- `System Track`: architecture, integration, or multi-surface work

Each plan is expected to define:

- objective
- scope and exclusions
- ordered implementation steps
- verification
- gaps/defaults
- multi-hive decision
- implementation readiness

Execution should not begin when readiness is blocked.

## Review and Verification Pipeline

Review and verification are separate gates.

```mermaid
flowchart TD
    Change["Implementation change"] --> ReviewDecision{"Independent review required?"}

    ReviewDecision -- "Skip by policy" --> Verify
    ReviewDecision -- "Single-model" --> Reviewer["Reviewer"]
    ReviewDecision -- "Multi-model" --> Reviewers["ReviewerGPT + ReviewerGemini + Reviewer"]

    Reviewers --> MultiReviewer["MultiReviewer"]
    Reviewer --> FixDecision
    MultiReviewer --> FixDecision

    FixDecision{"Findings require changes?"}
    FixDecision -- "Yes" --> Rework["Route focused follow-up fix"]
    Rework --> ReviewDecision
    FixDecision -- "No" --> Verify["Verifier"]

    Verify --> Verdict{"Verification Verdict"}
    Verdict -- "PASS" --> Close["Close task"]
    Verdict -- "BLOCKED" --> Rework
```

Key consequence:

- passing review is not enough to close a task
- passing verification is the default closure gate for non-trivial work

## Memory Model

The system keeps durable and transient context separate.

```mermaid
flowchart LR
    Task["Current task"] --> Session["Session memory\ntransient context"]
    Task --> Durable[".agent-memory/\ndurable repo knowledge"]

    Session --> Notes["draft plans\nrouting hints\ntemporary breadcrumbs"]
    Durable --> Decisions["project_decisions.md"]
    Durable --> Patterns["error_patterns.md"]
    Durable --> Archive["archive/"]

    Notes -. not canonical .-> Discard["do not treat as durable truth"]
    Decisions --> Future["future agent context"]
    Patterns --> Future
    Archive --> Future
```

Rules:

- durable repo knowledge belongs only in `.agent-memory/`
- session memory is useful for continuity but not canonical truth
- repo memory is kept git-tracked and portable instead of using expiring workspace-only memory

## Pandora's Box in the Architecture

Pandora's Box is the external context adapter for this repo.

```mermaid
flowchart LR
    Agent["Agent or skill needing external context"] --> Need{"What content is needed?"}

    Need -- "Repo-owned skills or examples" --> PBFile["Pandora's Box\nget_github_file_content"]
    Need -- "Pattern discovery across reference repo" --> PBSearch["Pandora's Box\nsearch_github_repo_patterns"]
    Need -- "Repo inventory" --> PBRepos["Pandora's Box\nlist_github_repos"]
    Need -- "Standards or internal docs" --> PBConfluence["Pandora's Box\nConfluence tools"]

    PBFile --> GitHub["GitHub API"]
    PBSearch --> GitHub
    PBRepos --> GitHub
    PBConfluence --> Confluence["Confluence API"]
```

Preferred retrieval order:

1. local workspace file when the repo is already checked out
2. Pandora's Box when the content lives in GitHub or Confluence
3. `context7` for external library and framework documentation
4. generic web fetch only for non-repository, non-library pages

## What the System Optimises For

The design optimises for:

- explicit ownership boundaries
- clear readiness before execution
- independent review and verification
- durable operational knowledge without polluting product repos
- scalable delegation using worktrees and hidden specialists
- structured external retrieval through Pandora's Box

It does not optimise for:

- fully autonomous uncontrolled agent fan-out
- implicit tool usage without governance
- treating transient session memory as project truth
- closing tasks on code inspection alone without objective verification

## Suggested Reading Order

If you are new to the repo, read in this order:

1. `README.md`
2. `docs/architecture.md`
3. `agents/orchestrator.agent.md`
4. `agents/planner.agent.md`
5. `skills/README.md`
6. `mcp/pandoras-box/README.md`

That sequence gives you the top-level model first, then the concrete routing, policy, and external-context layers.