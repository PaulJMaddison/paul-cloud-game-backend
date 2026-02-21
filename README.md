# paul-cloud-game-backend

**Production-Inspired, Event-Driven Multiplayer Game Backend in Go**

## 🚀 What This Project Is

`paul-cloud-game-backend` is a ready-to-run backend template for teams that want to build and validate multiplayer game features quickly, without inventing backend fundamentals from scratch.

It is designed for:
- multiplayer prototypes
- live-ops experimentation
- session-based games
- matchmaking systems

At its core, the platform combines:
- an event-driven architecture for service-to-service coordination
- real-time gateway routing for connected players
- session orchestration across services
- a local-first development workflow so you can test complete flows on your own machine

## 🎮 Why Use paul-cloud-game-backend?

- Avoid rebuilding auth, sessions, and matchmaking plumbing from zero.
- Prototype multiplayer features locally without cloud lock-in.
- Use production-inspired patterns (gateway + routing + event bus) early.
- Test real-time player flows end-to-end on a laptop before deployment.
- Start local now, then evolve toward Kubernetes or managed cloud later.
- Great fit for indie studios, technical designers, and backend learning projects.

## 🧩 What Problems It Solves

Multiplayer backends are less about one API and more about coordinating player state across services in real time. This project gives you that foundation by handling:

- **Player session lifecycle**: login, session creation, and session-aware service interactions.
- **Real-time client messaging**: persistent WebSocket connections through a gateway service.
- **Matchmaking flows**: enqueueing players and emitting match events.
- **Outbound user routing**: delivering server-generated messages to the right connected users.
- **Event-driven orchestration**: pub/sub workflows between backend services.

## 🏗️ Architecture Overview

The system is split into focused services you can run and scale independently:

- **Gateway Service**: HTTP edge + WebSocket entrypoint for real-time clients.
- **Login Service**: authentication/login flows.
- **Sessions Service**: session lifecycle and session-oriented operations.
- **Matchmaking Service**: queue + match orchestration workflows.
- **Routing Service**: internal routing layer for backend-to-backend messaging.

Supporting infrastructure:
- **NATS** for event transport
- **Redis** for low-latency state/cache patterns
- **Postgres** for durable relational data

## ⚡ Local-First Developer Experience

- Docker-based infrastructure dependencies.
- Bring up infra and services locally in minutes.
- Full test tiers available (unit + integration + e2e).
- Simulate realistic multiplayer flows end-to-end without cloud deployment.

## 🛠️ Use Cases

- Session-based multiplayer games
- Lobby and party systems
- PvP matchmaking prototypes
- Backend learning / portfolio projects
- Live-ops feature prototyping and validation

## 📈 Designed for Production-Style Scaling

This template is intentionally built with patterns that can grow with your project:

- Event bus abstraction for asynchronous workflows
- Gateway-centric routing model for real-time player delivery
- Service boundaries that support stateless scaling strategies
- Architecture that can be adapted to Kafka / Kubernetes style deployments later

## 🚦 Getting Started

```bash
# 1) Clone
git clone <your-fork-or-repo-url>
cd paul-cloud-game-backend

# 2) Setup
cp .env.example .env

# 3) Run infra
make docker-up
make migrate-up

# 4) Run services (single service example)
make run-local

# 5) Run tests
make test
make lint
```

For a full local end-to-end demo (infra + all services), run:

```bash
scripts/local-demo.sh
```

## 🤝 Contributing / Extending

This repository is intended to be an extensible backend foundation, not a fixed product. You can add game-specific domains (inventory, progression, parties, tournaments, live-ops controls) while keeping the same event-driven, service-oriented backbone.

Suggested extension path:
- add new bounded services under `cmd/`
- reuse shared packages under `pkg/`
- introduce new event subjects and handlers as features grow
- keep local-first validation as your default development loop
