# Plug-in Cluster Runbook

This runbook keeps the default deployment in single-node mode. Extra shard
nodes, the shard gateway, and cluster monitoring are opt-in pieces that can be
attached for validation and removed after the test window.

## Modes

| Mode | Switch | Behavior |
| --- | --- | --- |
| Single node | `AUCTION_CLUSTER_MODE=single` | Current production path. Backend uses one MySQL and one Redis. `/clusterz` returns `mode=single`. |
| Sharded backend | `AUCTION_CLUSTER_MODE=sharded` | Backend exposes its shard registry snapshot through `/clusterz`. |
| Shard gateway | `/app/auction-shard-gateway` | Unified HTTP/WebSocket entry. Routes writes by room/lot/order and aggregates dashboard list reads across shards. |
| Realtime bus | `AUCTION_REALTIME_BUS=redis` / `redis_stream` / `nats` | Publishes WebSocket events through Redis Pub/Sub, Redis Stream, or NATS so every backend can receive cross-instance room events. |
| Cluster monitoring | `deploy/prod/docker-compose.cluster-monitoring.yml` | Mounts gateway scraping and cluster alert rules without changing the single-node Prometheus file. |
| Gateway admin | `AUCTION_GATEWAY_ADMIN_TOKEN` | Enables protected runtime shard upsert, drain, room assignment, and autoscale evaluation endpoints. Empty means write control plane is disabled. |

## Registry Example

```json
{
  "shards": [
    {"id": 0, "name": "shard-0", "backendUrl": "http://example.com", "status": "active", "weight": 1, "maxActiveRoom": 100},
    {"id": 1, "name": "shard-1", "backendUrl": "http://172.31.226.179:18080", "status": "active", "weight": 1},
    {"id": 2, "name": "hot-room-pool", "backendUrl": "http://172.31.234.151:18080", "status": "active", "hotDedicated": true}
  ],
  "assignments": [{"roomId": "room-hot-001", "shardId": 2}]
}
```

`hotDedicated=true` keeps that shard out of normal new-room assignment. It only
serves rooms explicitly listed in `assignments`. `maxActiveRoom` is a soft cap
for new-room placement.

## Plug-in Steps

1. Keep `example.com` running in the current single-node deployment.
2. Confirm `LIVE_AUCTION_ENV_FILE` points to the server env file. The scripts default to `/opt/live-auction/.env`.
3. Start shard stacks on the additional nodes with `deploy/prod/shard-stack-run.sh up`.
4. Start the optional gateway on the entry node with `deploy/prod/cluster-entry-run.sh up`.
5. Use a Redis route table through `AUCTION_GATEWAY_ROUTE_REDIS_ADDR` so room, lot, and order ownership survives gateway restarts.
6. Enable a realtime bus before running more than one backend:
   `AUCTION_REALTIME_BUS=redis_stream` for Redis-backed recovery,
   `AUCTION_REALTIME_BUS=redis` for simple low-latency validation, or
   `AUCTION_REALTIME_BUS=nats` for a brokered fanout path. For NATS WebSocket
   broadcast, leave `AUCTION_REALTIME_BUS_NATS_QUEUE` empty; queue groups are
   only for load-balanced workers and would make some backend instances miss
   room events.
   For Redis Stream broadcast, keep the default per-instance group
   `auction-realtime-${AUCTION_INSTANCE_ID}`. Sharing one group across backend
   instances turns the stream into load balancing and can make a WebSocket
   instance miss remote room events.
   In a multi-server validation, expose NATS on the private network with
   `AUCTION_NATS_BIND_HOST=<entry-private-ip-or-0.0.0.0>` and set every shard
   stack to `AUCTION_REALTIME_BUS_NATS_URL=nats://<entry-private-ip>:14222`.
7. Replace the 120 Nginx site with `deploy/prod/live-auction.gateway.nginx.conf.template` after confirming the gateway listens on `127.0.0.1:18081`.
8. Run preflight checks: `GET /readyz`, `GET /workerz`, `GET /clusterz`, `GET /api/admin/lots`, and `GET /ws/rooms/{room_id}`.
9. Optionally start the NATS broker with `deploy/prod/cluster-entry-run.sh nats-up`.
10. Optionally start cluster monitoring with `deploy/prod/cluster-entry-run.sh monitoring-up`.
11. Run sharded load tests through the public entry. The gateway should route create/update/bid/pay paths to one shard and aggregate dashboard list reads.

## Monitoring Targets

Cluster monitoring uses Prometheus `file_sd` so adding or removing a server is
an edit to target metadata, not a Prometheus config rewrite.

| File | Purpose |
| --- | --- |
| `deploy/prometheus/file_sd/live-auction-backends.yml` | Backend scrape targets with `shard`, `shard_name`, and `role` labels. Update this when plugging or removing shard stacks. |
| `deploy/prometheus/file_sd/live-auction-gateways.yml` | Shard gateway scrape targets. Keep at least one gateway target while cluster mode is enabled. |
| `deploy/grafana/dashboards/live-auction-cluster-operations.json` | Cluster operations dashboard for gateway routes, shard state, room assignments, projection lag, WebSocket fanout, and pool pressure. |

Gateway-specific metrics:

| Metric | Meaning |
| --- | --- |
| `auction_gateway_routed_requests_total{route,shard,result}` | Routed HTTP/WebSocket-entry requests by route source and destination shard. |
| `auction_gateway_aggregate_shard_requests_total{path,shard,result}` | Per-shard subrequest result for cross-shard list aggregation. |
| `auction_gateway_shard_status{shard,status}` | Gateway registry status gauge; one status should be `1` per shard. |
| `auction_gateway_room_assignments{shard}` | Number of pinned/known room assignments per shard. |

## Runtime Control Plane

Set `AUCTION_GATEWAY_ADMIN_TOKEN` only during an explicit cluster operation
window. Requests must pass either `X-Auction-Admin-Token` or
`Authorization: Bearer <token>`.

| Endpoint | Method | Purpose |
| --- | --- | --- |
| `/cluster/admin/shards` | `POST` / `PUT` | Upsert a shard with full `Shard` JSON. Used when a node is plugged in. |
| `/cluster/admin/shards?id={id}` | `DELETE` | Remove an unassigned shard from the runtime registry. |
| `/cluster/admin/shards/status` | `POST` / `PATCH` | Set `active`, `draining`, or `offline`. Used for drain and recovery. |
| `/cluster/admin/shards/failover` | `POST` | Explicitly move room assignments from one shard to another and optionally mark the source offline. |
| `/cluster/admin/rooms/assign` | `POST` / `PUT` | Pin a room to a specific shard, including hot-room isolation. |
| `/cluster/admin/rooms/assign?roomId={id}` | `DELETE` | Clear a manual room assignment. |
| `/cluster/admin/autoscale/evaluate` | `POST` | Evaluate scale policy from supplied metrics; with `apply=true`, safe recommendations are applied. |

Autoscale can activate a pre-registered draining standby shard, recommend adding
a new node when no standby exists, or drain the least-loaded non-default active
shard when load is low. It does not provision cloud machines by itself.

## Drain Steps

1. Mark the shard as `draining` in the registry.
2. Reload gateway/backend registry config.
3. New rooms stop landing on that shard; existing rooms continue there.
4. Wait for active auctions and Projection pending count to reach zero.
5. Stop the shard stack.
6. Mark it `offline` or remove it from the registry.
7. Confirm `/clusterz` no longer reports active traffic for that shard.

## Failover Behavior

- New rooms avoid `draining` and `offline` shards.
- Existing room assignments on a draining shard keep working until drained.
- A lot/order already bound to an unavailable shard fails closed instead of falling back to the default shard. This avoids reading or writing the wrong MySQL/Redis stack.
- Explicit room failover uses `POST /cluster/admin/shards/failover`:

```json
{
  "sourceShardId": 1,
  "targetShardId": 0,
  "roomIds": ["room-a", "room-b"],
  "markSourceOffline": true
}
```

If `roomIds` is omitted, every room assignment currently owned by
`sourceShardId` is moved. If `targetShardId` is omitted, the gateway chooses an
active non-hot-dedicated shard. A hot-dedicated shard is used only when
`includeHotDedicatedTarget=true` is supplied. Room assignment updates are
written through the route table, so a Redis-backed gateway route table stays in
sync with the registry.
- Transparent stateful failover for active rooms still requires data replication or an explicit room migration procedure.

## Rollback To Single Node

1. Restore the original single-node Nginx site.
2. Run `deploy/prod/cluster-entry-run.sh rollback` on the entry node.
3. Run `deploy/prod/shard-stack-run.sh rollback` on each extra shard node.
4. The entry rollback stops `auction-shard-gateway` and optional NATS, then
   restarts the base Prometheus/Grafana definitions so cluster monitoring
   overrides do not remain active.
5. Keep the 120 backend with `AUCTION_CLUSTER_MODE=single` and `AUCTION_REALTIME_BUS=local` or unset.
6. Confirm `GET /readyz` returns OK, `GET /clusterz` returns `mode=single`, and `GET /workerz` shows only the 120 workers.

## Verification Checklist

- `POST /api/realtime/ws-ticket` with a `roomId` reaches the same shard as `/ws/rooms/{roomId}`.
- `POST /api/lots` binds the returned `lot.id` to the room shard.
- `POST /api/lots/{lot_id}/bid` uses the stored lot route.
- `POST /api/orders/{order_id}/mock-pay` uses the stored order route.
- `GET /api/admin/lots`, `/api/admin/orders`, `/api/admin/rooms`, `/api/rooms`, `/api/me/orders`, and `/api/me/bids` aggregate list results across active and draining shards.
- `/clusterz` includes registry, shard readiness, worker state, and runtime projection status.
- Gateway `/metrics` exposes `auction_gateway_routed_requests_total`, `auction_gateway_aggregate_shard_requests_total`, `auction_gateway_shard_status`, and `auction_gateway_room_assignments`.
- Cluster dashboard `LiveAuction Cluster Operations` appears in Grafana when the dashboard provisioning directory is mounted.
- Cluster alert rules are loaded only when `deploy/prod/docker-compose.cluster-monitoring.yml` is included.
