
# Whiteboard Architecture

## Core Position

The Synthetic Engineer should be designed as a governed autonomous software factory, not just an LLM with tool access.

MCP matters because it standardises access to external context and tools, but MCP does not decide what context is authoritative or how an agent should apply it. Its architecture is host-client-server, with capability negotiation, isolated client-server sessions, and host-controlled security boundaries. Resources are application-driven for context sharing, while tools are model-invokable capabilities.

That means MCP is necessary, but not sufficient. The real architecture challenge is the control system layered on top of it.

## Is it possible

The short answer is no.

The long answer:

It’s not realistically implementable end-to-end with current LLMs and tooling because the system requires reliable long-horizon reasoning, grounded decision-making, and trustworthy self-evaluation, which today’s models don’t consistently provide. LLMs can generate code and plans, but they struggle to maintain coherent intent across multi-step workflows, especially when requirements are ambiguous or evolve. Tooling like MCP solves access to context, not context quality, trust or correctness. So the agent can easily base decisions on stale or conflicting information. Most critically, “LLM-as-a-judge” and self-validation loops are still correlated and fallible. The same model family often generates, tests, and evaluates outputs, leading to false confidence rather than true correctness. Without robust, deterministic validation, strong isolation, and human oversight at key decision points, the system would be brittle, unsafe and prone to confidently producing the wrong software.

## The System Must Do

At a high level, the system needs to:

- turn ambiguous human intent into a clear engineering contract
- gather and rank context from trusted sources
- plan work before implementation begins
- execute changes in isolation
- validate outcomes with evidence, not confidence
- recover safely when validation fails
- return control to a human at the right decision points

## Reference Architecture

[1] Human Brief + Steering
    -> requirement parser
    -> ambiguity / conflict detector
    -> assumption ledger
    -> approval gates

[2] Orchestrator / State Machine
    -> manages phases, retries, budgets, branches, audit trail

[3] Context Layer (via MCP)
    -> Confluence standards
    -> existing repos / patterns
    -> local files / build configs
    -> ticketing / docs / architecture records
    -> provenance, freshness, and trust scoring

[4] Planning Layer
    -> system design
    -> task graph
    -> milestones
    -> acceptance criteria
    -> test and benchmark plan

[5] Execution Layer
    -> codegen worker
    -> refactor worker
    -> test generation worker
    -> docs worker
    -> infra/setup worker

[6] Verification Layer
    -> compile / typecheck / lint
    -> unit + integration tests
    -> benchmark harness
    -> static analysis / security / licence checks
    -> LLM-as-a-judge review

[7] Recovery Layer
    -> diagnose failure
    -> propose fix
    -> patch in isolated branch
    -> re-run affected validations

[8] Delivery Layer
    -> PR / report / design summary / release notes
    -> final human review

## End-to-End Flow

### 1. Intake and contract formation

The first deliverable is not code. It is a structured contract that defines:

- goals
- non-goals
- constraints
- tech stack
- acceptance criteria
- measurable SLAs
- open assumptions

This matters because prompts such as "build a high-performance web search engine" are too underspecified to execute safely.

### 2. Context retrieval through MCP

The system uses MCP servers to access standards, repos, and local environment data. MCP gives standardised access to resources and tools, but the host still decides how context is selected and incorporated.

Because of that, I would add a context-ranking layer on top of MCP based on:

- freshness
- relevance
- authority
- project scope

### 3. Planning before implementation

Before any code is written, the planner should produce:

- architecture options
- dependency choices
- task DAG
- testing strategy
- benchmark plan
- escalation points

Implementation should not start until success can be expressed in measurable terms.

### 4. Autonomous execution in sandboxes

Workers generate code in isolated branches or worktrees. Every change should be tied to:

- the requirement it addresses
- the evidence it depends on
- the validation steps it must pass

### 5. Verification and self-correction

The system validates itself with an evidence stack:

- compile / type / lint
- unit and integration tests
- runtime execution
- performance benchmarks
- static analysis and dependency checks
- LLM semantic review

Only after failures are localised should the recovery loop attempt a patch.

### 6. Human review and delivery

The human should come back into the loop with a concise handoff that explains:

- what was built
- assumptions made
- tradeoffs taken
- evidence that it works
- unresolved risks

## Main Design Risks And Mitigations

### 1. Ambiguous requirements

Issue: Intent parsing and sentiment analysis can help, but they do not create a reliable engineering spec.

Risk: The agent builds the wrong thing with high confidence.

Mitigation: Force a contract-first step with measurable acceptance criteria and an assumption ledger.

### 2. Poor context quality

Issue: MCP standardises access, not truth. Old Confluence pages or legacy repos may dominate decisions.

Risk: The agent copies stale patterns or contradictory standards.

Mitigation: Add provenance, recency, authority, and applicability scoring on retrieved context. MCP resources are explicitly application-driven, so this ranking layer belongs in the host.

### 3. Unsafe tool autonomy

Issue: The same system may read repos, run code, modify files, or call external services.

Risk: One bad decision becomes a real operational incident.

Mitigation: Use least privilege, sandboxing, approval gates, replayable logs, and separate read/write policies. MCP guidance emphasises host-enforced security boundaries, capability negotiation, and user authorisation.

### 4. Long-horizon planning drift

Issue: Plans go stale as implementation changes.

Risk: Subtasks optimise local progress while breaking global coherence.

Mitigation: Use an event-driven orchestrator with replanning triggers, dependency tracking, and durable project state.

### 5. LLM-as-a-judge correlation failure

Issue: If the same model family writes the code, writes the tests, and grades the result, failures are correlated.

Risk: The system passes against its own misunderstandings.

Mitigation: Use LLM judging only after deterministic checks and runtime evidence, and calibrate with human-reviewed evals instead of trusting judges blindly.

### 6. Shallow self-validation

Issue: The agent may optimise for "tests pass" instead of real correctness.

Risk: It writes tests that confirm its own implementation.

Mitigation: Separate requirement tests, regression tests, integration tests, and benchmarks. Require externally meaningful evidence.

### 7. Fake performance claims

Issue: "Low latency" depends on workload, concurrency, dataset, hardware, warmup, and p99 behaviour.

Risk: The agent ships benchmark theater.

Mitigation: Use a standardised performance harness, baseline comparisons, environment-controlled runs, and p95/p99 thresholds.

### 8. Retry-loop thrashing

Issue: Autonomous patch loops can oscillate between fixes and regressions.

Risk: Tangled code and wasted compute.

Mitigation: Use isolated branches, rollback points, capped retry budgets, and patch rationales before execution.

### 9. Missing durable memory

Issue: Chat context alone is not enough for multi-phase delivery.

Risk: The system forgets why architectural decisions were made.

Mitigation: Persist requirements, assumptions, evidence, failures, decisions, and benchmark history in a project state store.

### 10. Human steering that is too weak or too noisy

Issue: Too many interruptions kill autonomy, while too few create silent product decisions.

Risk: You either get poor UX or misaligned output.

Mitigation: Escalate only on high-uncertainty, high-impact, or irreversible decisions.

##  Sentiment Analysis

Sentiment analysis belongs in the system, but only as a supporting signal, not as a core controller.

It is useful for detecting:

- frustration
- hesitation
- hidden dissatisfaction
- contradiction between stated intent and tone

It should help decide when to escalate. It should not decide what to build or override explicit structured requirements.

## MCP

MCP is an interoperability layer, not a reasoning layer. It gives the system standardised access to resources and tools, with capability negotiation and host-managed security boundaries, but the host still needs policies for trust, ranking, permissions, and context selection.

## Recommended Implementation Choices

I would recommend:

- one orchestrator with explicit state
- specialised workers only where independent review adds value
- sandboxed execution
- branch-based retries
- evidence-driven validation
- human approval only at key decision gates

I would avoid:

- one giant prompt
- fully unrestricted tool access
- using LLM judgement as the primary quality gate
- inferring requirements without traceability

## Presentation Walkthrough

### Step 1: Requirements to contract

"We start by converting the human brief into a structured contract: goals, constraints, assumptions, and measurable acceptance criteria. This is critical because vague inputs like 'high-performance' are otherwise ambiguous."

### Step 2: Orchestrator as the control brain

"At the centre is an orchestrator implemented as a state machine. It manages phases like planning, execution, validation, and recovery, and it maintains persistent project memory."

### Step 3: Context through MCP

"We pull in external knowledge through MCP, including Confluence docs and Git repos, but we do not trust that context blindly. We rank it by freshness, authority, and relevance before using it."

### Step 4: Planning layer

"Before writing code, the system generates an architecture, a task DAG, a testing strategy, and a benchmark plan."

### Step 5: Execution layer

"Workers generate code, tests, and documentation in isolated environments, typically using branch-based or sandboxed execution."

### Step 6: Verification layer

"Validation is evidence-driven: compile and lint, unit and integration tests, runtime execution, performance benchmarks, security and static analysis, and only then LLM-as-a-judge."

### Step 7: Recovery loop

"Failures trigger a bounded retry loop where the system diagnoses issues, proposes fixes, and revalidates."

### Step 8: Delivery and human review

"Finally, the system outputs a PR and a report explaining decisions, tradeoffs, and evidence for correctness."

## System Context

```mermaid
flowchart TD

    A[Human Input\nRequirements / Feedback] --> B[Requirement Interpreter\n- Intent parsing\n- Ambiguity detection\n- Acceptance criteria]

    B --> C[Orchestrator (State Machine)\n- Task planning\n- Phase control\n- Retry & rollback\n- Audit & memory]

    C --> D[Context Layer (via MCP)\n- Confluence\n- Git repos\n- Local files\n- APIs\n- Ranked by trust/relevance]

    C --> E[Planning Layer\n- Architecture design\n- Task DAG\n- Test strategy\n- Benchmark plan]

    D --> E

    E --> F[Execution Layer\n- Code generation\n- Refactoring\n- Test generation\n- Docs]

    F --> G[Verification Layer\n- Compile / lint\n- Unit & integration tests\n- Runtime execution\n- Benchmarks\n- Security scans\n- LLM-as-judge]

    G --> H{Validation Passed?}

    H -->|No| I[Recovery Loop\n- Diagnose failures\n- Generate fixes\n- Retry (bounded)]

    I --> F

    H -->|Yes| J[Delivery Layer\n- PR / repo output\n- Reports & summary\n- Human review]

    C --> K[Project Memory Store\n- Requirements\n- Decisions\n- Context provenance\n- Failures & history]

    K --> C
```
