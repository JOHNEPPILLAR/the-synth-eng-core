# Documentation

This folder contains the durable reference docs for the The Synthetic Engineer control plane.

## Documents

- [architecture.md](architecture.md) - end-to-end system overview of the core repo, including the runtime model, key components, routing model, and Mermaid diagrams.
- [scaling-design.md](scaling-design.md) - scaling topology, operational bottlenecks, backpressure model, and phased growth path for the architecture.
- [whiteboard-architecture.md](whiteboard-architecture.md) - higher-level architecture framing for discussion and presentation, including feasibility constraints, control-plane design principles, and a simplified system-context diagram.

## Suggested Reading Order

1. [whiteboard-architecture.md](whiteboard-architecture.md) for the higher-level narrative and presentation version.
2. [architecture.md](architecture.md) for the canonical repo-level architecture.
3. [scaling-design.md](scaling-design.md) for how the architecture should evolve under concurrency, retrieval pressure, and validation load.

## Intended Audience

- maintainers who need to understand how the repo is structured
- downstream teams adopting the control plane into another repository
- contributors changing orchestration, review, verification, or MCP retrieval behaviour