# Observability + Service Mesh Enhancement Plan

Goal: turn the bookstore-api monolith into a playground for learning OpenTelemetry, Cilium, and Istio on a local Kind cluster.

## Current state assessment

- **Not microservices** — a single Go binary (`pkg/main.go`) serves both the REST API and the HTML-template frontend, talking directly to Postgres. This is fine for the goal: mesh value (mTLS, L7 routing, telemetry) shows up across the existing component boundaries (ingress → `frontend-svc` → pod, pod → `postgres-0.postgres.database.svc.cluster.local`).
- Good already: tier separation via namespaces (`frontend-api` vs `database`), StatefulSet + headless Service, three manifest variants, working Flux/GitOps push.
- Gaps:
  - No health/readiness probes in any Deployment.
  - Zero instrumentation in the app: no `/metrics`, no traces, only `fmt.Printf` logging, no request contexts/timeouts on HTTP or DB.
  - DB config hard-coded in `main.go`; the `POSTGRES_*` env vars in the manifests are dead code.
  - `resources: {}` everywhere despite `helm/values.yaml` having sane limits.
  - No auth for Postgres mesh ingress — `mTLS STRICT` can break the headless-FQDN DB link.
  - No Gateway API / Istio Ingress Gateway, no NetworkPolicies, no metrics scrape Service.

## Working rules

- No tests exist → every code PR is verified with `go build ./... && go vet ./...` (from the repo root) **plus a manual smoke run** against a local Postgres or the Kind cluster. A change that builds but crashes at runtime is the failure mode to guard against.
- Every change lands on a branch as a small PR.
- The three manifest sets (`manifests/`, `kustomize/` — the Flux artifact, `helm/`) must stay in sync on every infra edit.

## Phase 0 — Code hardening

| PR | Change | Why it blocks later phases |
|---|---|---|
| 0.1 | `GET /healthz` + `GET /readyz` (does `db.Ping()`) in `main.go`; add `livenessProbe`/`readinessProbe` to all 3 manifest sets | Istio traffic shifting + Cilium LB + mesh pod lifecycle need probes |
| 0.2 | Replace hard-coded DB constants with `os.Getenv("POSTGRES_*")` + fallbacks | Makes the already-injected manifest env vars real; needed before sidecars/DNS matter |
| 0.3 | Structured logging (needs Go >= 1.21 for `log/slog`) | OTel trace/log correlation |
| 0.4 | Request timeouts + `context` on DB calls | Cleaner spans, no hung requests skewing metrics |

Decision pending: bump `go.mod` and Dockerfile `golang:1.19-alpine` to 1.21 for `slog` (recommended), or stay on 1.19 and use stdlib `log` with key/value formatting.

## Phase 1 — OpenTelemetry in the app

- Add `go.opentelemetry.io/otel` + `contrib/instrumentation/net/http/otelhttp` for the server, manual spans around the 4 `books` queries, and a Prometheus exporter exposing `/metrics` on port 9090 (separate from the 8080 app port).
- Add a `metrics` Service + `prometheus.io/scrape: "true"` pod annotation across all 3 manifest sets.
- Trace export via OTLP to an in-cluster OTel Collector (default: **Jaeger** for traces + **Prometheus/Grafana** for metrics — cheap on Kind).

## Phase 2 — Cilium on Kind

- Kind config with kube-proxy disabled (`kubeProxyReplacement` mode) — Cilium replaces it; enable **Hubble** for L4 flow observability.
- Add a `CiliumNetworkPolicy` allowing only `frontend-api` → `database:5432`.

## Phase 3 — Istio

- Install Istio (`profile demo`) + **metallb** (Kind has no LB) for the ingress gateway; enable sidecar injection on both namespaces.
- `PeerAuthentication` with `mTLS STRICT` **after** confirming app → `postgres-0...svc.cluster.local` survives (postgres pod gets a sidecar too; may need a ServiceEntry for the headless FQDN — own step since STRICT can silently break the DB link).
- Replace the Ingress with Istio `Gateway`/`VirtualService` keeping host `app.santoshdts`; keep NodePort 31000 as fallback.
- **Kiali** for topology + mTLS dashboard.

## Phase 4 — Polish

- `resources` limits applied (values.yaml already has sane ones) so istio-proxy/cilium don't get throttled.

## Landing order

0.1 → 0.2 → 0.3 → 0.4 → Phase 1 → Phase 2 → Phase 3 → Phase 4, each merged green and smoke-tested before the next.
