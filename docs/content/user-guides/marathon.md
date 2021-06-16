# Marathon

This guide explains how to integrate Marathon and operate the cluster in a reliable way from Traefik's standpoint.

## Host detection

Marathon offers multiple ways to run (Docker-containerized) applications, the most popular ones being

- BRIDGE-networked containers with dynamic high ports exposed
- HOST-networked containers with host machine ports
- containers with dedicated IP addresses ([IP-per-task](https://mesosphere.github.io/marathon/docs/ip-per-task.html)).

Traefik tries to detect the configured mode and route traffic to the right IP addresses. It is possible to force using task hosts with the `forceTaskHostname` option.

## Port detection

Traefik also attempts to determine the right port (which is a [non-trivial matter in Marathon](https://mesosphere.github.io/marathon/docs/ports.html)) from the following sources:

1. An arbitrary port specified through label `traefik.http.services.serviceName.loadbalancer.server.port=8080`
1. The task port.
1. The port from the application's `portDefinitions` field.
1. The port from the application's `ipAddressPerTask` field.

### Port label syntax

To select a port, you can either

- specify the port directly: `traefik.http.services.serviceName.loadbalancer.server.port=8080`
- specify a port index: `traefik.http.services.serviceName.loadbalancer.server.port=index:0`
- specify a port name: `traefik.http.services.serviceName.loadbalancer.server.port=name:http`
- otherwise the first one is selected.

## Achieving high availability

### Scenarios

There are three scenarios where the availability of a Marathon application could be impaired along with the risk of losing or failing requests:

- During the startup phase when Traefik already routes requests to the backend even though it has not completed its bootstrapping process yet.
- During the shutdown phase when Traefik still routes requests to the backend while the backend is already terminating.
- During a failure of the application when Traefik has not yet identified the backend as being erroneous.

The first two scenarios are common with every rolling upgrade of an application (i.e. a new version release or configuration update).

The following sub-sections describe how to resolve or mitigate each scenario.

#### Startup

It is possible to define [readiness checks](https://mesosphere.github.io/marathon/docs/readiness-checks.html) (available since Marathon version 1.1) per application and have Marathon take these into account during the startup phase.

The idea is that each application provides an HTTP endpoint that Marathon queries periodically during an ongoing deployment in order to mark the associated readiness check result as successful if and only if the endpoint returns a response within the configured HTTP code range.
As long as the check keeps failing, Marathon will not proceed with the deployment (within the configured upgrade strategy bounds).

Beginning with version 1.4, Traefik respects readiness check results if the Traefik option is set and checks are configured on the applications accordingly.

!!! note
    Due to the way readiness check results are currently exposed by the Marathon API, ready tasks may be taken into rotation with a small delay.
    It is on the order of one readiness check timeout interval (as configured on the application specification) and guarantees that non-ready tasks do not receive traffic prematurely.

If readiness checks are not possible, a current mitigation strategy is to enable [retries](../middlewares/http/retry.md) and make sure that a sufficient number of healthy application tasks exist so that one retry will likely hit one of those.
Apart from its probabilistic nature, the workaround comes at the price of increased latency.

#### Shutdown

It is possible to install a [termination handler](https://mesosphere.github.io/marathon/docs/health-checks.html) (available since Marathon version 1.3) with each application whose responsibility it is to delay the shutdown process long enough until the backend has been taken out of load-balancing rotation with reasonable confidence (i.e., Traefik has received an update from the Marathon event bus, recomputes the available Marathon backends, and applies the new configuration).
Specifically, each termination handler should install a signal handler listening for a SIGTERM signal and implement the following steps on signal reception:

1. Disable Keep-Alive HTTP connections.
1. Keep accepting HTTP requests for a certain period of time.
1. Stop accepting new connections.
1. Finish serving any in-flight requests.
1. Shut down.

Traefik already ignores Marathon tasks whose state does not match `TASK_RUNNING`; since terminating tasks transition into the `TASK_KILLING` and eventually `TASK_KILLED` state, there is nothing further that needs to be done on Traefik's end.

How long HTTP requests should continue to be accepted in step 2 depends on how long Traefik needs to receive and process the Marathon configuration update.
Under regular operational conditions, it should be on the order of seconds, with 10 seconds possibly being a good default value.

Again, configuring Traefik to do retries (as discussed in the previous section) can serve as a decent workaround strategy.
Paired with termination handlers, they would cover for those cases where either the termination sequence or Traefik cannot complete their part of the orchestration process in time.

#### Failure

A failing application always happens unexpectedly, and hence, it is very difficult or even impossible to rule out the adversal effects categorically.

Failure reasons vary broadly and could stretch from unacceptable slowness, a task crash, or a network split.

There are two mitigation efforts:

1. Configure [Marathon health checks](https://mesosphere.github.io/marathon/docs/health-checks.html) on each application.
2. Configure Traefik health checks (possibly via the `traefik.http.services.yourServiceName.loadbalancer.healthcheck.*` labels) and make sure they probe with proper frequency.

The Marathon health check makes sure that applications once deemed dysfunctional are being rescheduled to different slaves.
However, they might take a while to get triggered and the follow-up processes to complete.

For that reason, the Treafik health check provides an additional check that responds more rapidly and does not require a configuration reload to happen.
Additionally, it protects from cases that the Marathon health check may not be able to cover, such as a network split.

### (Non-)Alternatives

There are a few alternatives of varying quality that are frequently asked for.

The remaining section is going to explore them along with a benefit/cost trade-off.

#### Reusing Marathon health checks

It may seem obvious to reuse the Marathon health checks as a signal to Traefik whether an application should be taken into load-balancing rotation or not.

Apart from the increased latency a failing health check may have, a major problem with this is that Marathon does not persist the health check results.
Consequently, if a master re-election occurs in the Marathon clusters, all health check results will revert to the _unknown_ state, effectively causing all applications inside the cluster to become unavailable and leading to a complete cluster failure.
Re-elections do not only happen during regular maintenance work (often requiring rolling upgrades of the Marathon nodes) but also when the Marathon leader fails spontaneously.
As such, there is no way to handle this situation deterministically.

Finally, Marathon health checks are not mandatory (the default is to use the task state as reported by Mesos), so requiring them for Traefik would raise the entry barrier for Marathon users.

Traefik used to use the health check results as a strict requirement but moved away from it as [users reported the dramatic consequences](https://github.com/traefik/traefik/issues/653).

#### Draining

Another common approach is to let a proxy drain backends that are supposed to shut down.
That is, once a backend is supposed to shut down, Traefik would stop forwarding requests.

On the plus side, this would not require any modifications to the application in question.
However, implementing this fully within Traefik seems like a non-trivial undertaking.

Additionally, the approach is less flexible compared to a custom termination handler since only the latter allows for the implementation of custom termination sequences that go beyond simple request draining (e.g., persisting a snapshot state to disk prior to terminating).

The feature is currently not implemented; a request for draining in general is at [issue 41](https://github.com/traefik/traefik/issues/41).
