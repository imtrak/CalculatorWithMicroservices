# Calculator

A full-stack calculator. The user types or taps a free-form arithmetic expression such as `(12 + 3) * 4`; a Go backend parses and evaluates it, and a React frontend shows the result. Everything runs on Kubernetes, and it can also be run directly on the host for the fastest feedback loop.

```text
browser ──> Ingress (Traefik) ──┬── /      ──> calculator-dashboard  (nginx serving the React app)
 calculator.localhost:8088      └── /api   ──> calculator-service    (Go REST API)
```

- **Frontend** (`frontend/`): React 19 + TypeScript + Vite. Keypad and text input, immediate input feedback, readable errors, responsive, light and dark mode.
- **Backend** (`services/calculator-service/`): Go, standard library only. Tokenizer, parser and evaluator for an allowlisted grammar. Nothing is ever passed to `eval`, a shell or a database.
- **Kubernetes** (`kubernetes/`): manifests for both services, an Ingress, and a local `kind` cluster definition.
- **Decisions** (`docs/adr/`): the reasoning behind the architecture.

## What it calculates

Finite decimal numbers, parentheses, and the operators `+ - * /` with standard precedence (parentheses, then `*` and `/`, then `+` and `-`, left to right). Unary `+` and `-` are supported.

| Example | Result |
|---|---|
| `10+50+60/80*40` | `90` |
| `(12 + 3) * 4` | `60` |
| `-(2 + 3) * 2` | `-10` |
| `0.1 + 0.2` | `0.3` |
| `.5 + 5.` | `5.5` |

Limits: at most 256 characters and 64 levels of nesting. Whitespace means a space or a tab.

Results are rounded to 15 significant digits so that binary floating-point noise does not leak out (`0.1 + 0.2` is `0.3`, not `0.30000000000000004`). Integers of up to 15 digits stay exact.

Not implemented: exponentiation, square root and percentage (optional in the assignment). Adding one means changing the tokenizer, the parser and the tests, and the frontend's character check (`frontend/src/lib/input.ts`) must stay in sync with the backend's alphabet.

## Prerequisites

| Tool | Needed for | Version |
|---|---|---|
| Go | backend | 1.27 or newer |
| Node.js | frontend | 20.19+ or 22.12+ |
| Docker | images, Kubernetes | recent |
| [kind](https://kind.sigs.k8s.io/) and `kubectl` | Kubernetes | recent |

## Run it

### Option 1: on the host (fastest)

```bash
# Terminal 1: the API on http://localhost:8080
cd services/calculator-service
go run ./cmd/calculator-service

# Terminal 2: the frontend on http://localhost:5173
cd frontend
npm install
npm run dev
```

Vite proxies `/api` to the backend, so the browser sees a single origin. To point at a backend elsewhere, set `VITE_DEV_API_TARGET`, for example `VITE_DEV_API_TARGET=http://localhost:9000 npm run dev`.

### Option 2: on Kubernetes with kind

```bash
# 1. Create the cluster (maps the node's port 80 to host port 8088)
kind create cluster --name calculator --config kubernetes/kind/cluster.yaml

# 2. Build the images and load them into the cluster
docker build -t calculator-service:dev services/calculator-service
docker build -t calculator-dashboard:dev frontend
kind load docker-image calculator-service:dev calculator-dashboard:dev --name calculator

# 3. Install the Ingress Controller, then the application
kubectl apply -f kubernetes/ingress-controller
kubectl -n traefik rollout status deployment/traefik
kubectl apply -k kubernetes/base
kubectl -n calculator rollout status deployment/calculator-service deployment/calculator-dashboard
```

Open **http://calculator.localhost:8088**. Browsers and most resolvers map `*.localhost` to the loopback address, so no `/etc/hosts` entry is needed.

To pick up a code change, rebuild the image, run `kind load docker-image ... --name calculator` again, and `kubectl -n calculator rollout restart deployment/<name>`.

Clean up with `kind delete cluster --name calculator`.

## API

Base path: `/api/v1`. Bodies are JSON.

### `POST /api/v1/calculations`

```bash
curl -X POST http://calculator.localhost:8088/api/v1/calculations \
  -H 'Content-Type: application/json' \
  -d '{"expression": "(12 + 3) * 4"}'
```

```json
{ "expression": "(12 + 3) * 4", "result": 60 }
```

Errors share one shape:

```bash
curl -X POST http://calculator.localhost:8088/api/v1/calculations \
  -H 'Content-Type: application/json' -d '{"expression": "1 / 0"}'
```

```json
{ "error": { "code": "division_by_zero", "message": "Division by zero." } }
```

| Status | `code` | When |
|---|---|---|
| 400 | `empty_expression` | blank or missing expression |
| 400 | `expression_too_long` | more than 256 characters |
| 400 | `invalid_characters` | a character outside the allowed alphabet, e.g. `2 + abc` |
| 400 | `invalid_number` | malformed number, e.g. `1.2.3` or `.` |
| 400 | `invalid_syntax` | e.g. `1 + * 2`, `2(3)`, `()` |
| 400 | `unmatched_parenthesis` | e.g. `(1 + 2` or `1 + 2)` |
| 400 | `nesting_too_deep` | more than 64 levels of nesting |
| 400 | `division_by_zero` | e.g. `1/0` or `1/(2-2)` |
| 400 | `non_finite_result` | the result is too large to represent |
| 400 | `invalid_request` | body is not a JSON object with a string `expression` field |
| 413 | `request_too_large` | body over 4 KiB |
| 415 | `unsupported_media_type` | `Content-Type` is not `application/json` |
| 500 | `internal_error` | unexpected server error |

Syntax errors are reported before arithmetic errors, so `1/0 +` is `invalid_syntax`.

### Other endpoints

| Endpoint | Purpose |
|---|---|
| `GET /api/v1` | Describes the API (useful to open in a browser). |
| `GET /api/healthz`, `GET /api/readyz` | Health, reachable through the Ingress. |
| `GET /healthz`, `GET /readyz` | Used by the Kubernetes probes. Directly on the service these are the root paths; through the Ingress `/healthz` is answered by the dashboard instead. |

Every response carries an `X-Request-ID` header. A well-formed incoming value is reused, otherwise one is generated, and it appears in the service logs so that a request can be traced.

## Tests and coverage

```bash
# Backend
cd services/calculator-service
go test -race -cover ./...
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

# Frontend
cd frontend
npm test            # run once
npm run coverage    # with a coverage report (HTML in frontend/coverage/)
npm run typecheck
```

Measured results:

| Area | Tests | Coverage |
|---|---|---|
| Backend: `internal/expression` | table-driven: valid expressions, every error code, nesting and length limits | 95.7% of statements |
| Backend: `internal/server` | HTTP contract, validation, error shape, request ID, health, panic recovery | 94.3% of statements |
| Frontend (all of `src`) | 47 tests: pure logic, API client, and the full UI against a mocked `fetch` | 98.9% statements, 95.5% branches, 100% lines |

`cmd/calculator-service/main.go` and `frontend/src/main.tsx` are start-up wiring and are not unit tested. The Kubernetes manifests and the images were verified by running them on a kind cluster, not by automated tests.

## Design decisions

The reasoning is in `docs/adr/`:

| ADR | Decision |
|---|---|
| [0001](docs/adr/0001-free-form-arithmetic-expressions.md) | One endpoint taking a free-form expression, parsed with an allowlisted grammar, instead of one endpoint per operation. |
| [0002](docs/adr/0002-kubernetes-deployment.md) | Deploy on Kubernetes, with probes, resource limits and a local kind cluster. |
| [0003](docs/adr/0003-api-gateway-and-calculator-service.md) | *Superseded.* An API gateway in front of a calculator service. |
| [0004](docs/adr/0004-single-calculator-service.md) | A single backend service instead: the gateway only relayed requests and duplicated validation. |
| [0005](docs/adr/0005-single-host-ingress-routing.md) | One host, routed by path (`/` and `/api`), so the browser sees one origin and no CORS is needed. |

Assumptions and trade-offs worth knowing:

- **The backend is authoritative.** The frontend checks characters and length for immediate feedback, but the backend validates everything again.
- **The service is stateless**: no database, no sessions, no history.
- **No authentication or rate limiting.** Both are out of scope; they would belong at the Ingress or in a gateway if needed later.
- **Security hardening is on by default:** containers run as a non-root numeric user with a read-only root filesystem and no capabilities, and the dashboard sends a strict Content-Security-Policy.
- **Images are tagged `:dev` and loaded straight into kind.** There is no registry or CI pipeline.
- **The Ingress Controller is Traefik**, installed from plain manifests, because `ingress-nginx` is archived.

## Project layout

```text
docs/
  assignment.md            the original brief
  adr/                     architecture decision records
frontend/                  React + TypeScript app, nginx config and Dockerfile
services/calculator-service/
  cmd/calculator-service/  entry point
  internal/expression/     tokenizer, parser, evaluator
  internal/server/         HTTP handlers and middleware
kubernetes/
  base/                    namespace, both services, Ingress (apply with -k)
  ingress-controller/      Traefik
  kind/                    local cluster definition
```
