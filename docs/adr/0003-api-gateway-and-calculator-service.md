# ADR-0003: Use an API gateway and a calculator service

- Status: Superseded by [ADR-0004](0004-single-calculator-service.md)
- Date: 2026-10-15

## Context

The calculator accepts free-form arithmetic expressions such as `(12 + 3) * 4`.
We want to demonstrate a practical Kubernetes deployment with independently
deployable services, while avoiding an artificial microservice for each
arithmetic operator.

The frontend needs one stable public API. Expression parsing and evaluation are
one cohesive domain capability and should remain together.

## Decision

The application will use two backend services:

1. `api-gateway` is the public API boundary. It receives requests from the
   frontend, exposes the versioned REST contract, performs request-level
   validation, and forwards calculation requests to the internal calculator
   service.
2. `calculator-service` owns expression tokenization, parsing, semantic
   validation, evaluation, and calculation-domain errors.

Each service will run in its own Kubernetes `Deployment` and be exposed through
its own `Service`. `calculator-service` uses a `ClusterIP` service and is not
exposed outside the cluster. The frontend communicates only with `api-gateway`;
the gateway communicates with `calculator-service` through Kubernetes DNS.

The first version of `api-gateway` will deliberately stay thin. It will not
repeat expression parsing or arithmetic logic. Its responsibilities are routing,
public-contract validation, request correlation, and translating service errors
to the public HTTP response. Authentication, rate limiting, and caching are not
part of the initial scope.

```text
React frontend --> api-gateway --> calculator-service
                         |
                    public API
```

## Consequences

- The public API boundary can evolve independently from the calculation domain.
- The calculator service is private to the cluster, reducing its externally
  reachable surface.
- The project demonstrates Kubernetes service discovery and service-to-service
  communication with an architecture that has a clear responsibility split.
- Each calculation has one additional network hop, and deployments, tests,
  observability, and error handling must cover both services.
- Gateway and calculator error contracts must be explicitly tested so the
  frontend receives stable, useful errors.

## Considered alternatives

### A single calculator API service

This is simpler and would be adequate for the calculator's current functional
scope. It does not create a separate public API boundary or demonstrate
service-to-service communication.

### One microservice per arithmetic operation

Separate addition, subtraction, multiplication, and division services would
force the system to coordinate several calls for most expressions. This adds
latency, failure modes, deployment objects, and testing complexity without a
meaningful domain boundary.

### Expose calculator-service directly

This avoids the extra gateway hop but couples the frontend to the internal
calculation contract and makes later public API concerns harder to introduce.
