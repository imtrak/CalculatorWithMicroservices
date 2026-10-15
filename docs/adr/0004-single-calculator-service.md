# ADR-0004: Use a single calculator service

- Status: Accepted
- Date: 2026-10-15
- Supersedes: [ADR-0003](0003-api-gateway-and-calculator-service.md)
- Amends: [ADR-0002](0002-kubernetes-deployment.md)

## Context

ADR-0003 split the backend into an `api-gateway` and a `calculator-service`.
The gateway was implemented and worked: it validated the request, rejected
unsupported characters and forwarded valid requests to the calculator.

In practice the gateway added no capability of its own. It had one route and one
destination, so it did not route; it only validated and relayed. The validation
it performed (JSON shape, size, expression length, allowed characters) is
validation the calculator service must perform anyway, because the calculator is
authoritative for what a valid expression is. Keeping the character allowlist in
two services also meant the two copies had to be changed together whenever the
grammar grew.

## Decision

The backend is a single service, `calculator-service`, deployed as one
`Deployment` with one `Service`. It is the public API and owns all validation and
evaluation:

- `POST /api/v1/calculations` with the contract from
  [ADR-0001](0001-free-form-arithmetic-expressions.md).
- Request-level validation: `Content-Type`, body size, JSON shape.
- Expression validation: length, allowed characters, syntax, nesting depth,
  division by zero and non-finite results.
- `GET /healthz` and `GET /readyz` for the Kubernetes probes.
- A request ID, taken from `X-Request-ID` when it is well formed and generated
  otherwise, which is returned on the response and included in the logs.

```text
React frontend --> Service calculator-service --> calculator-service pod
```

The Kubernetes `Service` is only a stable address and load balancer in front of
the pods; it contains no application logic.

## Consequences

- There is one image, one `Deployment` and one `Service` to build, deploy and
  test, and no network hop between services.
- The alphabet of the expression grammar lives in one place, the tokenizer.
- The project no longer demonstrates service-to-service communication. That was
  a goal of ADR-0003, and it is given up in exchange for a design that matches
  the size of the problem.
- A public edge concern such as authentication, rate limiting or routing to more
  backends would need either to be added to this service or to bring back a
  gateway. None is in scope today.
- The `Service` is `ClusterIP`. Reaching it from a browser still requires the
  `Ingress` or `kubectl port-forward` fallback described in ADR-0002.

## Considered alternatives

### Keep the gateway and the calculator as separate services (ADR-0003)

This separates the public contract from the calculation domain and demonstrates
service discovery. It was implemented and then rejected: with a single route and
a single backend the gateway was a relay, it duplicated validation, and it added
a hop, a second deployment and a second failure mode without a current
requirement that needed them.

### A gateway that forwards by request content

This would make the gateway a real router, but there is only one backend to
route to.
