# ADR-0002: Deploy the application on Kubernetes

- Status: Accepted, amended by [ADR-0004](0004-single-calculator-service.md) (the backend is a single service, so there is no API gateway or separate calculator `Deployment`) and by [ADR-0005](0005-single-host-ingress-routing.md) (how the Ingress routes requests)
- Date: 2026-10-15

## Context

The calculator consists of a React frontend and a backend API. The assignment
can be run with a simpler local process setup, but the project is also an
opportunity to demonstrate how the application would be packaged and operated
in an environment closer to production.

Using Kubernetes makes deployment concerns explicit: independently deployable
services, internal networking, configuration, health checks, and scalable
workload definitions. It also provides concrete material to discuss during an
interview.

## Decision

The frontend, API gateway, and calculator service will be packaged as separate
container images and deployed to Kubernetes. The repository will include
Kubernetes manifests that can run on a local cluster such as kind or minikube.

The initial deployment will include:

- A `Deployment` and `Service` for the API gateway.
- A `Deployment` and internal `ClusterIP` `Service` for the calculator API.
- A `Deployment` and `Service` for the frontend.
- An `Ingress` (or documented port-forward fallback) that exposes the frontend
  and routes API requests without requiring the browser to know an internal
  service address.
- Readiness and liveness probes for the backend, once it exposes health
  endpoints.
- Resource requests and limits with conservative values appropriate for local
  development.
- Environment-specific settings in `ConfigMap` objects; secrets will not be
  committed to the repository.

The manifests will remain intentionally small. Features that do not provide
value for this application—such as a service mesh, Helm chart, autoscaling, or
a multi-node production topology—are out of scope unless a later requirement
needs them.

## Consequences

- The frontend and backend can be built, versioned, and deployed independently.
- Kubernetes service discovery keeps backend addresses out of frontend source
  code and supports later scaling.
- The project demonstrates containerization and basic Kubernetes operational
  practices during an interview.
- Local setup becomes more demanding: contributors need Docker and a local
  Kubernetes cluster, and manifests must be maintained alongside the services.
- Documentation must include both a quick local development path and the
  Kubernetes deployment steps, so Kubernetes does not become a barrier to
  reviewing the core application.

## Considered alternatives

### Run both services directly on the host

This is the fastest option for development, and it will remain supported for
the shortest feedback loop. It does not demonstrate deployment configuration or
service-to-service networking.

### Docker Compose only

Compose is simpler than Kubernetes for local multi-container development, but
it does not cover Kubernetes concepts that are relevant to the project's
learning and interview goals.

### Use managed Kubernetes

A managed cluster is closer to a production platform, but it requires cloud
credentials, can incur costs, and is unnecessary for an interview assignment.
Local Kubernetes keeps the setup reproducible and self-contained.
