# CLAUDE.md

Guidance for Claude Code when working in this repository.

## What this is

Kubernetes operator that creates, configures, and manages **Redis Sentinel-based
failovers** on top of Kubernetes. It watches a `RedisFailover` custom resource and
reconciles a full HA topology (N redis replicas + M sentinels) toward the declared
spec, self-healing master/replica and sentinel state.

**Fork status:** This is `dnse-tech`'s fork of the now-archived, unmaintained
upstream `spotahome/redis-operator` (last release Dec 2022). We intend to take over
and maintain it for our own use.

> Note: the Go module path and image names still say `spotahome` (`go.mod`,
> `Makefile` `IMAGE_NAME`/`REPOSITORY`). Renaming the module path is a deliberate,
> breaking decision — do **not** rename it casually; confirm with the maintainer
> first, since it touches every import and the generated client.

## Tech stack

- **Language:** Go (module declares `go 1.20`; local toolchain is newer — build/test still work)
- **Controller framework:** [kooper v2](https://github.com/spotahome/kooper) (not controller-runtime/kubebuilder)
- **CRD:** `RedisFailover`, group/version `databases.spotahome.com/v1`
- **Redis client:** `go-redis/redis/v8`
- **Metrics:** Prometheus (`prometheus/client_golang`), exposed on `/metrics`
- **Logging:** logrus wrapped in `log/` (wrapper exists so it can be mocked)
- **K8s libs:** `k8s.io/{api,apimachinery,client-go,apiextensions-apiserver}` v0.27.x

## Layout

```
api/redisfailover/v1/   CRD Go types, defaults, validation, deepcopy (zz_generated)
client/k8s/             Generated typed clientset/informers for the CRD (do not hand-edit)
cmd/redisoperator/      main entrypoint: flags, k8s client wiring, metrics server, run loop
cmd/utils/              k8s client creation helpers
operator/redisfailover/ Core reconcile logic (kooper controller)
  handler.go              Handle() — entrypoint per event
  ensurer.go              Ensure — create/update all owned k8s resources
  checker.go              CheckAndHeal — inspect live redis/sentinel, drive to desired state
  config.go, factory.go   Controller construction & config
  service/                Lower-level: generator (builds k8s objects), check, heal, client, names
  util/                   label/pod helpers
service/k8s/            Typed wrappers over k8s API (configmap, deployment, statefulset, service, secret, rbac, pdb, pod)
service/redis/          Redis/Sentinel client wrapper (CONFIG GET/SET, sentinel commands)
metrics/                Prometheus recorder
log/                    logrus wrapper
mocks/                  Generated mocks (mockery via `go generate`) — do not hand-edit
manifests/              CRD YAML + kustomize base/overlays/components
charts/redisoperator/   Helm chart
example/                Sample RedisFailover specs (one per feature)
docs/                   development.md (layout), logic.md (reconcile pipeline)
```

## Reconcile model (how it works)

Per `RedisFailover` event, `handler.go` runs two phases (see `docs/logic.md`):

1. **Ensure** — idempotently create/update every owned resource: redis
   configmap + shutdown configmap + statefulset, sentinel configmap + deployment +
   services, PDBs, RBAC. Manual edits to these are overwritten on next sync.
2. **CheckAndHeal** — connect to each redis and sentinel; enforce exactly one
   master, all replicas following it, all sentinels monitoring the same master,
   correct replica counts, and custom config applied live via `CONFIG SET` /
   `SENTINEL SET` (so custom config is **not** written into configmaps).
   Split-brain is **not** auto-resolved — it logs an error and waits for manual fix.

Created resources are named `rfr-<name>` (redis) and `rfs-<name>` (sentinel).
Custom-config changes are pushed to running processes, not persisted to configmaps.

## Build / test / lint

Run these directly (no Docker needed) unless reproducing CI exactly:

```bash
# Unit tests (what CI runs as `make ci-unit-test`)
go test $(go list ./... | grep -v /vendor/) -v
# or:
make ci-unit-test

# Lint (CI uses golangci-lint v1.53; check.sh enables goimports)
golangci-lint run -E goimports --timeout 3m

# Build the binary -> ./bin/redis-operator
./scripts/build.sh

# Regenerate mocks after changing an interface
go generate ./mocks
```

Docker-wrapped equivalents (`make unit-test`, `make build`, `make shell`) run the
same commands inside `docker/development` — use only if you need the pinned env.

**Integration tests** (`make ci-integration-test`, build tag `integration`) require
a running Kubernetes cluster (CI uses minikube `--driver=none`) with the CRD applied
first (`kubectl create -f manifests/databases.spotahome.com_redisfailovers.yaml`).
They will **not** run on a plain dev machine without a cluster.

**Helm tests:** `make helm-test`.

## CI (.github/workflows/ci.yaml)

Jobs: `golangci-lint` (v1.53) → unit test → integration test (matrix k8s
1.24–1.27, needs check+unit) → chart test. **Heads-up for the fork:** the workflow
triggers on push to `main`, but the default branch here is `master`, so pushes to
`master` currently only run via the `pull_request` trigger — revisit branch names
when taking over maintenance.

## Code generation (needs Docker + codegen image)

- `make update-codegen` — regenerate the typed client + deepcopy from `api/` types
- `make generate-crd` — regenerate the CRD manifest from Go types

After editing `api/redisfailover/v1/types.go`, regenerate both, and update the
example specs / CRD in `manifests/` and `charts/redisoperator/crds/`.

## Conventions & gotchas

- **Generated code is off-limits:** `client/k8s/**`, `mocks/**`,
  `api/**/zz_generated.deepcopy.go`, and the CRD YAMLs are produced by codegen —
  change the source (`api/` types, interfaces) and regenerate, never hand-edit.
- RedisFailover **name must be ≤ 48 chars** (statefulset + prefix limit).
- Validation lives in `api/redisfailover/v1/validate.go`; defaults in `defaults.go`
  (default image versions live here — referenced by the README).
- Each `service/k8s/*.go` and its `*_test.go` are paired; mirror that when adding.
- Follow existing style: table-driven tests with `testify`, mocks via mockery.
