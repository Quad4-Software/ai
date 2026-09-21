---
name: devops
description: >
  This skill covers DevOps and platform engineering as of September
  2026: CI/CD systems (GitHub Actions, GitLab CI, Woodpecker, Argo,
  Dagger), GitOps (Argo CD v3.x, Flux v2.9), IaC post-Terraform-split
  (OpenTofu v1.11, Pulumi, Crossplane graduated), config management
  (Ansible default, Chef/Puppet legacy, InSpec license trap),
  observability (OpenTelemetry, Prometheus 3.x LTS, Grafana LGTM,
  Vector), DORA and SPACE metrics, internal developer platforms
  (Backstage, Crossplane, Humanitec post-acquisition), progressive
  delivery (OpenFeature, flagd, Argo Rollouts GA, Flagger), and SRE
  practices (SLOs, error budgets, incident management). Use it when
  designing pipelines, choosing deployment strategy, or standing up
  platform tooling.
---

## When to use this skill

- You are designing or hardening a CI/CD pipeline.
- You are choosing IaC, GitOps, or config-management tooling.
- You are setting up observability, SLOs, or release strategies.
- You are evaluating platform engineering components.

## How to use

1. Read this file for the landscape and decision rules.
2. For CI/workflow hardening see `ci-security`. For container
   runtimes see `docker`, `podman`, `k3s`. For isolation see
   `firecracker`.
3. Verify version claims against the tool's releases page. These
   notes are pinned to September 2026.

## Examples

- "Should we use OpenTofu or Pulumi for greenfield IaC?"
- "Design a canary release with automatic rollback on SLO burn."
- "What does a minimal observability stack look like?"

# DevOps

## CI/CD landscape

- GitHub Actions is the de facto default. Aug 2025: org/repo policy
  can enforce SHA pinning and block specific actions. This repo pins
  all actions to full SHAs with version comments.
- GitLab CI: integrated single-platform choice.
- Woodpecker CI: lightweight Apache-2.0 container-native CI, pairs
  with Forgejo/Gitea for fully self-hosted pipelines.
- Argo: Workflows for batch/ML DAGs, CD for GitOps, Events for
  triggers, Rollouts for progressive delivery.
- Tekton for K8s pipeline primitives, Dagger for programmable CI,
  Buildkite for large monorepos, Jenkins is legacy but huge.

## GitOps

Pull-based reconciliation per OpenGitOps:

- Argo CD (CNCF graduated, v3.x): app-centric UI, ApplicationSet
  generators including PR preview environments.
- Flux (CNCF graduated, v2.9): modular controllers, strongest for
  platform-managed multi-tenant fleets.
- Running both is a documented pattern: Flux for infra bootstrap,
  Argo CD for the app surface.

## IaC after the Terraform split

- Terraform moved to BUSL 1.1 (Aug 2023). IBM acquired HashiCorp
  (Dec 2024).
- OpenTofu (Linux Foundation, CNCF sandbox, MPL-2.0, v1.11.x):
  divergent features Terraform lacks: state encryption, ephemeral
  values, write-only attributes, `enabled` meta-argument, early
  variable evaluation, provider `for_each`. Terraform provider
  ecosystem stays compatible.
- Pulumi (Apache-2.0): language-native SDKs, runs HCL natively,
  `pulumi convert --from terraform` bridges providers.
- Crossplane graduated CNCF Nov 2025. V2.0 is the OSS control-plane
  foundation for platforms.
- Greenfield default: OpenTofu for HCL-style, Pulumi for
  language-native. Terraform remains dominant in existing estates.

## Configuration management

Ansible is the default for new work: agentless push over SSH/WinRM,
YAML, largest community. Puppet and Chef are legacy-anchored
(drift enforcement at fleet scale is Puppet's niche). License trap:
InSpec 6+ enforces license-key checks, InSpec 7 needs EULA
acceptance. Do not recommend it without noting that.

## Observability

- Instrument once with OpenTelemetry: traces stable, metrics stable,
  logs per-language (Go = Beta), profiling in development.
  Declarative YAML config (`OTEL_CONFIG_FILE`) went stable in 2026.
- Route via OTel Collector or Grafana Alloy. Vector for telemetry
  plumbing between systems.
- Store in Grafana LGTM (Loki, Grafana, Tempo, Mimir) or a vendor
  backend. Prometheus 3.x current, 3.5 is the LTS line.
- OTLP is the wire protocol. Never bet on a vendor-specific one.

## Metrics and delivery health

- DORA four keys: deployment frequency, lead time for changes,
  change failure rate, failed-deployment recovery time. Current
  reports add reliability as a fifth metric.
- SPACE complements DORA for human sustainability: Satisfaction,
  Performance, Activity, Communication, Efficiency. Measure teams,
  never individuals.
- No single metric captures productivity. Anyone selling one is
  selling dashboards.

## Platform engineering

IDP = portal + orchestrator + paved-path templates:

- Portal: Backstage (CNCF, the open standard. Note Backstage Plus
  moved some plugins commercial since Sept 2025), Port, Cortex.
- Orchestrator: Crossplane compositions, Kratix, Humanitec
  (original GmbH wound down, assets acquired by PlatCo mid-2024.
  Treat pre-2024 references as describing a different company).

## Progressive delivery

Separate deploy from release:

- Flags control exposure: OpenFeature is the CNCF vendor-neutral
  flag API. Flagd is its OSS evaluation daemon (pre-1.0). Caution:
  Unleash relicensed to AGPLv3 at v8.0 and OSS Unleash Edge is
  deprecated (EOL Dec 31, 2026) - verify and plan migration if used.
- Traffic shape: Argo Rollouts (GA March 2026, stable v1 APIs:
  canary, blue-green, analysis templates) or Flagger (CNCF
  graduated, Flux-native).
- Gate promotion on automated metric analysis (SLO burn), not on
  someone watching a dashboard.

## SRE practices

- SLIs/SLOs with error budgets as the release-velocity control.
- Toil capped at ~50% of SRE time. Automate it away.
- Incident management: severity matrix, single Incident Commander,
  structured comms, blameless postmortem with tracked actions.
- Mean-time-to-restore is the DORA metric incident practice feeds.
- Google SRE books remain free at sre.google/books.

## Deployment patterns

- Recreate: simplest, downtime. Dev only.
- Rolling: default for stateless services, watch pod-disruption
  budgets.
- Blue-green: instant cutover and rollback, double capacity cost.
- Canary: progressive traffic shift with metric gates via Argo
  Rollouts or Flagger.
- Whatever the strategy: test the rollback path before you need it,
  and test restores, not just backups.

## This repo

House standard for GitHub Actions: full SHA pins, harden-runner as
first step of every job, minimal permissions, `persist-credentials:
false`, OIDC over stored creds, zizmor gating workflow changes. See
`ci-security` for the rule set and `release` for GoReleaser plumbing.
