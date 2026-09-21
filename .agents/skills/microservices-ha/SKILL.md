---
name: microservices-ha
description: >
  This skill covers microservices vs modular monolith decisions,
  high-availability design, and horizontal scaling as of September
  2026. Use it when decomposing services, designing for HA (quorum,
  failover, multi-AZ vs multi-region), choosing resilience patterns
  (timeouts, retries, circuit breakers, sagas, outbox), picking a
  service mesh or gateway, autoscaling, or scaling databases. Includes
  the anti-patterns and what small deployments actually need.
metadata:
  sources:
    - https://istio.io/latest/about/service-mesh/
    - https://keda.sh/
    - https://sre.google/workbook/alerting-on-slos/
    - https://principlesofchaos.org/
---

## When to use this skill

- Deciding monolith vs microservices, or extracting a service.
- Designing for HA: failover, quorum, RTO/RPO, AZ vs region.
- Adding resilience: timeouts, retries, circuit breakers, sagas.
- Scaling: load balancing, autoscaling, caching, database growth.
- Choosing a service mesh, API gateway, or message queue.

## The default answer

Monolith first. Microservices are an organizational scaling tool
(Conway's law), not a technical upgrade. Below roughly 10+ engineers
the distributed-systems tax exceeds the payoff. Amazon Prime Video
famously consolidated a serverless pipeline into a monolith and cut
cost ~90%. Segment reverted from 100+ services.

A modular monolith - one deployable, enforced module boundaries,
module-owned data behind public interfaces - gives most of the benefit
at a fraction of the cost. Go internal/ packages, Spring Modulith, or
packwerk enforce the seams.

Extract a service only for a proven reason: measured independent
scaling need, independent deploy cadence, fault isolation, a
regulatory/data boundary, or a different runtime profile. Extract the
least-coupled module first (strangler fig), modularize inside the
monolith before cutting.

## Availability math

- Serial components multiply: five 99.9% services in a sync chain give
  ~99.5% best case. Keep sync request paths to 2-3 hops.
- Parallel replicas combine as 1 - (1-A)^n. Replicas help, chains hurt.
- N+1 means peak capacity survives losing one unit, not "a spare
  exists". Run hot tiers under ~66% at peak.
- Set RTO and RPO before designing. They pick the architecture:
  near-zero RPO forces synchronous replication, which is AZ-scale
  cheap and cross-region expensive.

## Resilience rules that prevent most outages

- Always set timeouts: connect, per-request, and an end-to-end
  deadline propagated across hops (gRPC deadlines, context.Context).
- Retries: exponential backoff plus full jitter, capped attempts,
  only on idempotent operations or with idempotency keys. Retry at
  ONE layer of the call graph - R retries at N layers is R^N
  amplification against the struggling dependency. Add a retry budget.
- Circuit breakers fail fast while a dependency recovers. Bulkheads
  (per-dependency pools, per-tenant quotas) bound blast radius.
- Bounded queues with early rejection (429/503 + Retry-After) create
  backpressure. Unbounded queues turn overload into OOM.
- TTLs always carry random jitter. Use request coalescing and
  stale-while-revalidate against stampedes.
- Assume at-least-once delivery everywhere. Consumers must be
  idempotent (dedup table or idempotency keys).
- Health checks: liveness = is it stuck, readiness = can it take
  traffic. Keep readiness shallow - failing a pod because a
  downstream is down turns a partial outage into a full one. Handle
  SIGTERM, drain in-flight, allow LB propagation lag.

## Data and distribution

- Database-per-service. A shared database is a distributed monolith.
- Cross-service reads: API composition or materialized views fed by
  events/CDC (Debezium). Never query another service's tables.
- Multi-service writes: sagas (orchestration preferred default) with
  compensating transactions. Not 2PC/XA.
- Dual writes need the outbox pattern: domain row + outbox row in one
  transaction, relayed by CDC. "Write to DB then publish" fails
  partially on every crash.
- Contracts: additive-only schema changes, expand-and-contract for
  breaking ones. OpenAPI/protobuf/AsyncAPI plus consumer-driven
  contract tests (Pact).
- Database scaling order: indexes and queries, vertical, connection
  pooling (PgBouncer), read replicas (watch replication lag and
  read-your-writes), partitioning, sharding last and rarely.

## Quorum and failover

- Quorum is floor(n/2)+1: deploy odd counts. Two etcd nodes have
  quorum 2 and lose it on any single failure. Two-server HA needs an
  external datastore or a witness/arbiter (k3s docs say external SQL
  for 2-server).
- Never trust a paused lock holder: fencing tokens, not just leases.
- Prefer LB/anycast failover over DNS failover, which is bounded by
  TTL and resolver behavior.
- Multi-AZ is the default HA baseline. Multi-region is a step change
  justified by DR, residency, or user latency - not "five nines"
  aspirations. Active-passive is dramatically simpler for stateful
  systems. Active-active needs conflict resolution.
- Do not implement consensus. Use etcd. Raft is what k8s already runs.

## Autoscaling and mesh (k8s)

- HPA on a leading indicator (queue depth, in-flight requests), not
  just CPU, which lags for I/O-bound services. VPA for rightsizing.
- KEDA (CNCF graduated) for queue/lag-driven scaling and
  scale-to-zero workers. Karpenter or cluster autoscaler at node level.
- Gateway API is the standard for north-south traffic. Ingress NGINX
  is being retired. Envoy Gateway, Cilium, Kong, or NGINX Gateway
  Fabric for new work.
- Service mesh (Istio ambient GA since 1.24, Linkerd, Cilium) buys
  automatic mTLS, uniform L7 telemetry, and traffic splitting - at the
  price of operating a second distributed system. Under ~10 services
  with no compliance mTLS requirement, use NetworkPolicy +
  cert-manager + library-level resilience instead.
- Kubernetes DNS + ClusterIP is sufficient service discovery.

## Anti-patterns

- Distributed monolith: lockstep deploys, shared DB, sync call chains.
- Deep sync chains (>3 hops), nanoservices, retry storms, thundering
  herd (synchronized cache expiry, mass reconnect), dual writes
  without outbox, sticky sessions, entity services (CRUD-per-table),
  sharing domain libraries between services.

## What a small deployment actually needs

2-node or k3s scale:

- Stateless app tier with 2+ replicas behind a simple LB (Traefik or
  a Gateway API impl), health checks, graceful shutdown. That buys
  most real-world availability.
- Stateful tier honestly: managed DB is the boring correct answer.
  Otherwise primary/standby Postgres + Patroni with a third witness
  node (2-node consensus splits brain), or a single primary with
  tested automated restore and accepted RTO. One tuned Postgres
  serves most SMB workloads entirely.
- Queue at small scale: Postgres SKIP LOCKED, NATS, or Valkey
  streams. Kafka is justified by throughput/replay, not fashion.
- Valkey over Redis for greenfield (license-clean fork. Redis 8 is
  AGPLv3 option).
- Minimum ops stack: tested restore drills (test restores, not
  backups), centralized logs, RED metrics, TLS automation, alerting
  to a human.

Cargo cult list: service mesh for a handful of services, multi-region
active-active, microservices themselves, CQRS/event sourcing as
default, Kafka for trickle workloads, cell-based architecture. See
"Choose Boring Technology" (McKinley, innovation tokens) and "You Are
Not Google" (Onay).

## Observability tie-in

- W3C Trace Context (traceparent/tracestate) is the propagation
  standard. OpenTelemetry handles it. Traces and metrics stable, logs
  stable at spec level with per-language variance.
- RED for services (rate, errors, duration), USE for resources
  (utilization, saturation, errors).
- Alert on SLO error-budget burn rate (multi-window, multi-burn-rate
  per SRE Workbook ch. 5), on symptoms users feel.
- Backend choices (LGTM, Prometheus) are in the devops skill. Chaos
  tooling: Chaos Mesh or LitmusChaos, hypothesis-driven game days not
  random destruction. Pin Chaos Mesh current and restrict its RBAC -
  it had critical CVEs in late 2025.
