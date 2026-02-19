# Event Contracts (Kafka/schema-repo style mimic)

This package defines a minimal event contract layer with a shared envelope and typed v1 payload schemas.

## Envelope

All events use the same JSON envelope fields:

- `id`
- `type`
- `ts`
- `correlation_id`
- `user_id` (optional)
- `payload`

## Supported event types

- `user.logged_in`
- `session.created`
- `session.assigned_server`
- `matchmaking.enqueued`
- `matchmaking.matched`
- `gateway.send_to_user`

## NATS subject mapping

Where `pcgb` stands for `paul-cloud-game-backend`.

- `user.logged_in` -> `pcgb.user.logged_in`
- `session.created` -> `pcgb.session.created`
- `session.assigned_server` -> `pcgb.session.assigned_server`
- `matchmaking.enqueued` -> `pcgb.mm.enqueued`
- `matchmaking.matched` -> `pcgb.mm.matched`
- `gateway.send_to_user` -> `pcgb.gateway.send_to_user`
