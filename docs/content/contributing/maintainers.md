# Maintainers

## The Team

* Emile Vauge [@emilevauge](https://github.com/emilevauge)
* Vincent Demeester [@vdemeester](https://github.com/vdemeester)
* Ed Robinson [@errm](https://github.com/errm)
* Daniel Tomcej [@dtomcej](https://github.com/dtomcej)
* Manuel Zapf [@SantoDE](https://github.com/SantoDE)
* Timo Reimann [@timoreimann](https://github.com/timoreimann)
* Ludovic Fernandez [@ldez](https://github.com/ldez)
* Julien Salleyron [@juliens](https://github.com/juliens)
* Nicolas Mengin [@nmengin](https://github.com/nmengin)
* Marco Jantke [@mjantke](https://github.com/mjeri)
* Michaël Matur [@mmatur](https://github.com/mmatur)
* Gérald Croës [@geraldcroes](https://github.com/geraldcroes)
* Jean-Baptiste Doumenjou [@jbdoumenjou](https://github.com/jbdoumenjou)
* Mathieu Lonjaret [@mpl](https://github.com/mpl)
* Romain Tribotté [@rtribotte](https://github.com/rtribotte)
* Kevin Pollet [@kevinpollet](https://github.com/kevinpollet)
* Harold Ozouf [@jspdown](https://github.com/jspdown)

## Maintainer's Guidelines

Please read the [maintainer's guidelines](maintainers-guidelines.md)

## Issue Triage

Issues and PRs are triaged daily and the process for triaging may be found under [triaging issues](https://github.com/traefik/contributors-guide/blob/master/issue_triage.md) in our [contributors guide repository](https://github.com/traefik/contributors-guide).

## PR Review Process

The process for reviewing PRs may be found under [review guidelines](https://github.com/traefik/contributors-guide/blob/master/review_guidelines.md) in our contributors guide repository.

## Labels

A maintainer that looks at an issue/PR must define its `kind/*`, `area/*`, and `status/*`.

### Status - Workflow

The `status/*` labels represent the desired state in the workflow.

* `status/0-needs-triage`: all the new issues and PRs have this status. _[bot only]_
* `status/1-needs-design-review`: needs a design review. **(only for PR)**
* `status/2-needs-review`: needs a code/documentation review. **(only for PR)**
* `status/3-needs-merge`: ready to merge. **(only for PR)**
* `status/4-merge-in-progress`: merge is in progress. _[bot only]_

### Contributor

* `contributor/need-more-information`: we need more information from the contributor in order to analyze a problem.
* `contributor/waiting-for-feedback`: we need the contributor to give us feedback.
* `contributor/waiting-for-corrections`: we need the contributor to take actions in order to move forward with a PR. **(only for PR)** _[bot, humans]_
* `contributor/needs-resolve-conflicts`: use it only when there is some conflicts (and an automatic rebase is not possible). **(only for PR)** _[bot, humans]_

### Kind

* `kind/enhancement`: a new or improved feature.
* `kind/question`: a question. **(only for issue)**
* `kind/proposal`: a proposal that needs to be discussed.
    * _Proposal issues_ are design proposals
    * _Proposal PRs_ are technical prototypes that need to be refined with multiple contributors.

* `kind/bug/possible`: a possible bug that needs analysis before it is confirmed or fixed. **(only for issues)**
* `kind/bug/confirmed`: a confirmed bug (reproducible). **(only for issues)**
* `kind/bug/fix`: a bug fix. **(only for PR)**

### Resolution

* `resolution/duplicate`: a duplicate issue/PR.
* `resolution/declined`: declined (Rule #1 of open-source: no is temporary, yes is forever).
* `WIP`: Work In Progress. **(only for PR)**

### Platform

* `platform/windows`: Windows related.

### Area

* `area/acme`: ACME related.
* `area/api`: Traefik API related.
* `area/authentication`: Authentication related.
* `area/cluster`: Traefik clustering related.
* `area/documentation`: Documentation related.
* `area/infrastructure`: CI or Traefik building scripts related.
* `area/healthcheck`: Health-check related.
* `area/logs`: Logs related.
* `area/middleware`: Middleware related.
* `area/middleware/metrics`: Metrics related. (Prometheus, StatsD, ...)
* `area/middleware/tracing`: Tracing related. (Jaeger, Zipkin, ...)
* `area/oxy`: Oxy related.
* `area/provider`: related to all providers.
* `area/provider/boltdb`: Boltd DB related.
* `area/provider/consul`: Consul related.
* `area/provider/docker`: Docker and Swarm related.
* `area/provider/ecs`: ECS related.
* `area/provider/etcd`: Etcd related.
* `area/provider/eureka`: Eureka related.
* `area/provider/file`: file provider related.
* `area/provider/k8s`: Kubernetes related.
* `area/provider/kv`: KV related.
* `area/provider/marathon`: Marathon related.
* `area/provider/mesos`: Mesos related.
* `area/provider/rancher`: Rancher related.
* `area/provider/servicefabric`: Azure service fabric related.
* `area/provider/zk`: Zoo Keeper related.
* `area/rules`: Rules related.
* `area/server`: Server related.
* `area/sticky-session`: Sticky session related.
* `area/tls`: TLS related.
* `area/websocket`: WebSocket related.
* `area/webui`: Web UI related.

### Issues Priority

* `priority/P0`: needs hot fix.
* `priority/P1`: need to be fixed in next release.
* `priority/P2`: need to be fixed in the future.
* `priority/P3`: maybe.

### PR Size

Automatically set by a bot.

* `size/S`: small PR.
* `size/M`: medium PR.
* `size/L`: Large PR.
