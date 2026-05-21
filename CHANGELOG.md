# Changelog

All notable changes to this project will be documented in this file.

Format: [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)
Versioning: [Semantic Versioning](https://semver.org/spec/v2.0.0.html)

---

## [Unreleased]

## [0.1.0] - 2026-05-21

### Added
- Go HTTP service with `/healthz` (liveness), `/readyz` (readiness), `/` (info) endpoints
- Graceful shutdown via SIGTERM with configurable timeout (`SHUTDOWN_TIMEOUT_SEC`)
- Structured JSON logging via `log/slog`
- Environment-based configuration (`PORT`, `APP_ENV`, `LOG_LEVEL`, `VERSION`)
- Multi-stage Dockerfile: `golang:1.24-alpine` builder + `distroless/static-debian12:nonroot` final (~3.5 MB)
- Kubernetes base manifests: Namespace, ServiceAccount, ConfigMap, Deployment, Service, HPA, PDB, NetworkPolicy
- Kustomize overlays: `dev` (1 replica, debug) and `prod` (3 replicas, production resources)
- GitHub Actions CI workflow: `go vet`, `go test -race`, docker build on push
- GitHub Actions release workflow: build + push to GHCR on semver tag, auto-update prod overlay image tag
- Full README: deployment guide, cleanup instructions, troubleshooting, best practices reference
- Architecture documentation in `docs/architecture.md`

### Security
- Container runs as non-root (UID 65532)
- Read-only root filesystem
- All Linux capabilities dropped
- Seccomp profile: RuntimeDefault
- NetworkPolicy: default deny-all, explicit whitelist
- ServiceAccount with `automountServiceAccountToken: false`
- Namespace enforced with Pod Security Standards `restricted`
