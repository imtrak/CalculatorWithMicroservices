# ADR-0005: Expose the application on one host with path-based Ingress routing

- Status: Accepted
- Date: 2026-10-15
- Amends: [ADR-0002](0002-kubernetes-deployment.md)

## Context

[ADR-0002](0002-kubernetes-deployment.md) calls for an `Ingress` that exposes the
frontend and routes API requests, without fixing how. The application has two
deployable parts: `calculator-dashboard` (the compiled React app served by nginx)
and `calculator-service` (the REST API, see [ADR-0004](0004-single-calculator-service.md)).

A browser only reaches the cluster through the Ingress, because both Services are
`ClusterIP`. The choice is how the browser should address the two parts.

## Decision

Serve both parts from a single host and route by path:

| Host | Path | Backend |
|---|---|---|
| `calculator.localhost` | `/api` (prefix) | Service `calculator-service` |
| `calculator.localhost` | `/` (prefix) | Service `calculator-dashboard` |

- The frontend calls the relative URL `/api/v1/calculations`. It has no
  configurable backend address and works unchanged in every environment.
- The browser sees one origin, so **CORS is not needed** and the backend does not
  implement it. In development, Vite's dev-server proxy forwards `/api` to the
  backend and gives the same single-origin behavior.
- The dashboard's nginx answers `404` for anything under `/api/`. That path is
  the Ingress's job; if a request reaches nginx, the routing is wrong, and it is
  better to fail than to return `index.html` for the app to misread as JSON.
- The Ingress Controller is **Traefik**, installed from plain manifests in
  `kubernetes/ingress-controller/`. `ingress-nginx` is not used because its
  repository is archived and no longer maintained.
- Locally, `kubernetes/kind/cluster.yaml` labels the node and maps its port 80 to
  host port 8088, so the application is available at
  `http://calculator.localhost:8088`. Browsers and most resolvers map
  `*.localhost` to the loopback address, so no `/etc/hosts` entry is needed.

## Consequences

- No CORS code, configuration or tests, and no backend URL to configure in the
  frontend.
- One host name to manage. Both parts share it, and so share cookies and
  origin-level security policy; the dashboard's Content-Security-Policy is
  restricted to `'self'`, which fits this setup.
- The Ingress Controller is a separate piece of infrastructure that must be
  installed before the application manifests, and is not part of
  `kubernetes/base`.
- The routing rule `/api` is a contract between the Ingress, the dashboard's
  nginx and the backend's route prefix (`/api/v1/...`). Changing one means
  changing the others.
- The API has no host of its own for direct clients. They can use the same host
  with the `/api` prefix.
- The port mapping (host 8088) is specific to local kind clusters; other
  environments need their own entry point and host name.

## Considered alternatives

### Separate hosts for the dashboard and the API, with CORS

Gives each part its own name, but the browser then sees two origins, so the
backend would have to send CORS headers and answer preflight requests, with the
allowed origin configured per environment, and the frontend would need the
API's address at build or deploy time. This adds code and configuration for no
current requirement.

### Separate hosts, with `/api` also routed on the dashboard host

Avoids CORS and gives the API its own name, but it keeps two names for one
application and is only useful if non-browser clients need a dedicated API host.
None exists today. It can be added later with one more Ingress rule.

### An nginx reverse proxy inside the dashboard pod

Also gives the browser a single origin, but reintroduces a component that
forwards requests, which the project deliberately removed in ADR-0004, and the
routing rules would then live in two places.

### `ingress-nginx` as the controller

Widely documented, and the default in many kind guides, but archived in 2026 and
no longer receiving fixes.

### `kubectl port-forward` only

Needs no controller, but it exposes one Service at a time, so the single-origin
setup could not be demonstrated, and it is not a deployment mechanism.
