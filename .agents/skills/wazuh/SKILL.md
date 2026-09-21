---
name: wazuh
description: >
  This skill covers Wazuh, the free GPLv2 SIEM+XDR platform, as of
  September 2026 (stable 4.14.x, 5.0 in beta with breaking changes).
  Use it for architecture (manager, indexer, dashboard, agent),
  deployment modes (all-in-one, distributed, Docker, Kubernetes),
  agent enrollment, capabilities (FIM, SCA, vulnerability detection,
  log analysis, active response, compliance mappings), custom
  rules/decoders, integrations (VirusTotal, Slack, Shuffle, MISP),
  sizing, and common pitfalls.
metadata:
  sources:
    - https://documentation.wazuh.com/
    - https://github.com/wazuh/wazuh
---

## When to use this skill

- Deploying or scaling Wazuh as a free SIEM/XDR.
- Writing custom decoders, rules, or CDB lists.
- Enrolling agents, wiring integrations, sizing indexer nodes.
- Deciding Wazuh vs Elastic Security, Security Onion, or Splunk.

## Version reality (Sept 2026)

- Stable line is 4.14.x (v4.14.7, July 2026). Target 4.x for
  production.
- 5.0 is beta (v5.0.0-beta4, July 2026), no announced GA and NO
  in-place upgrade from 4.x. Breaking changes: a new Engine replaces
  analysisd, YAML decoders replace XML, KVDB replaces CDB lists,
  Filebeat is removed, and agents get a new protocol. Plan a fresh
  install plus migration, not an upgrade.

## Architecture (4.x)

- Manager/server: analysisd (decoders/rules), remoted (agent comms,
  TCP/UDP 1514), authd (enrollment, 1515), wazuh-db, modulesd,
  syscheckd (FIM), logcollector. Cluster mode: one master plus workers.
- Indexer: OpenSearch fork storing alerts and agent state (9200).
- Dashboard: OpenSearch Dashboards fork (5601/443).
- Filebeat ships alerts from manager to indexer. If Filebeat is not
  running, alerts never arrive - the most common "empty dashboard"
  cause.
- Agent: lightweight endpoint agent, real-time encrypted channel,
  outbound to 1514/1515. Manager version must be >= agent version.

## What it covers

Log analysis, FIM (syscheck, real-time/whodata via auditd), SCA policy
checks (CIS benchmarks), vulnerability detection (agent inventory vs
CVE feed), malware detection (rootcheck, YARA integration, VirusTotal
hash lookups, CDB-list IOC matching), active response (block IP, kill
process on rule match), cloud log ingestion (AWS CloudTrail/GuardDuty/
S3, Azure, GCP, O365, GitHub audit), Docker/K8s log collection, MITRE
ATT&CK-mapped rules, and compliance dashboards for PCI DSS, HIPAA,
GDPR, NIST 800-53, CIS.

## Deployment

- Quickstart all-in-one: manager+indexer+dashboard on one host, fine
  for labs and small fleets.
- Distributed production: 3+ indexer nodes, manager cluster (1 master
  + N workers) behind a load balancer.
- Docker: wazuh/wazuh-docker compose files. Kubernetes:
  wazuh/wazuh-kubernetes (indexer and manager as StatefulSets, agents
  can run as DaemonSet).
- Wazuh Cloud is the paid managed SaaS. Self-hosted is $0 license.
- Agent enrollment via authd on 1515 with password or certs. Deploy-
  time env vars: WAZUH_MANAGER, WAZUH_REGISTRATION_SERVER,
  WAZUH_REGISTRATION_PASSWORD, WAZUH_AGENT_NAME, WAZUH_AGENT_GROUP.
  Change the default authd password in packaged deploys.

## Customization

- Put custom decoders in etc/decoders/local_decoder.xml and rules in
  etc/rules/local_rules.xml. Stock ruleset files are overwritten on
  upgrade - never edit them.
- CDB lists (etc/lists/) for IOC lookups and allowlists.
- ossec.conf global config plus per-group agent.conf pushed to agent
  classes.
- integratord for outbound integrations: VirusTotal, Slack,
  PagerDuty, MISP, Shuffle SOAR webhooks, TheHive, syslog forwarding.

## Sizing and pitfalls

- Indexer: recommended 16 GB RAM / 8 cores per node, JVM heap at 50%
  with Xms=Xmx, swap off. Watch disk watermarks - OpenSearch hits
  flood-stage around 95% and blocks indexing. Alert at 80%.
- Shards: roughly one primary per node, use ISM policies for
  retention and rollover before index-per-day sprawl.
- Manager: ~1,000 agents per manager node is the rule of thumb, scale
  horizontally with workers.
- Storage estimate: ~3.7 GB per server per 90 days of alerts, less
  for workstations, more for network devices.
- Pitfalls: Filebeat down = empty dashboards, cloned VMs cause agent
  key mismatches, vuln detector needs manager-to-indexer connectivity,
  no native multi-tenancy (matters for MSSP use).

## Positioning

Wazuh is the best free endpoint-centric SIEM: own agent, FIM, SCA,
compliance out of the box. Weak on network visibility, so pair it
with Security Onion or Zeek/Suricata when packet-level coverage
matters. Elastic Security wins if you already run Elastic. Splunk
Free is not viable beyond learning.
