# Architecture

## Overview

`go-k8s-healthcheck` adalah HTTP service minimal yang dirancang sebagai referensi
implementasi production-grade deployment di Kubernetes. Aplikasi tidak memiliki
business logic — fokusnya adalah pada pola deployment, observability, dan keamanan.

## Application Layer

```
cmd/server/main.go
│
├── config.Load()              ← env vars → Config struct
├── handler.NewHealthHandler() ← stateful: tracks ready flag & start time
├── http.Server{}              ← timeouts: read 10s, write 10s, idle 60s
│   ├── GET /                  ← info: app, version, env, uptime
│   ├── GET /healthz           ← liveness: always 200 if process alive
│   └── GET /readyz            ← readiness: 200 after SetReady(true), else 503
│
└── graceful shutdown
    ├── signal.Notify(SIGTERM, SIGINT)
    ├── SetReady(false)        ← stop receiving traffic first
    └── srv.Shutdown(ctx)      ← drain in-flight requests (max 30s)
```

## Container Layer

```
Stage 1: golang:1.24-alpine (builder)
  └── CGO_ENABLED=0 go build -trimpath -ldflags="-s -w"
      └── static binary, no libc dependency

Stage 2: gcr.io/distroless/static-debian12:nonroot (final)
  └── COPY binary only
  └── USER nonroot:nonroot (UID 65532)
  └── No shell, no package manager, no OS utilities
  └── Final size: ~3.5 MB
```

## Kubernetes Layer

```
Namespace: healthcheck
  │
  ├── ServiceAccount (no automount token)
  │
  ├── ConfigMap
  │   └── PORT, LOG_LEVEL, SHUTDOWN_TIMEOUT_SEC, APP_NAME
  │
  ├── Deployment
  │   ├── replicas: 2 (base) / 1 (dev) / 3 (prod)
  │   ├── strategy: RollingUpdate (maxUnavailable=0, maxSurge=1)
  │   ├── probes: liveness /healthz, readiness /readyz, startup /healthz
  │   ├── securityContext: runAsNonRoot, readOnlyRootFilesystem,
  │   │                    allowPrivilegeEscalation=false, drop ALL caps
  │   ├── resources: requests + limits (CPU + memory)
  │   ├── podAntiAffinity: prefer different nodes
  │   └── terminationGracePeriodSeconds: 60
  │
  ├── Service (ClusterIP :80 → pod :8080)
  │
  ├── HorizontalPodAutoscaler
  │   ├── min: 2, max: 5
  │   ├── scale up: CPU > 70% or Memory > 80%
  │   └── scale down: stabilize 5 min, 1 pod per 60s
  │
  ├── PodDisruptionBudget
  │   └── minAvailable: 1
  │
  └── NetworkPolicy
      ├── default-deny-all (ingress + egress)
      ├── allow-ingress-controller (from kube-system :8080)
      └── allow-dns (egress to kube-system :53)
```

## Kustomize Overlay Strategy

```
k8s/base/          ← semua resource, image tag: dev, replicas: 2
    │
    ├── overlays/dev/
    │   ├── image newTag: dev
    │   └── patch: replicas=1, resources kecil, LOG_LEVEL=debug
    │
    └── overlays/prod/
        ├── image newTag: <semver> (auto-updated by CI on release)
        └── patch: replicas=3, resources besar, LOG_LEVEL=info
```

## CI/CD Pipeline

```
Developer
  │
  ├── push to main/develop
  │     └── ci.yml: go vet → go test -race → docker build (no push)
  │
  └── git tag v*.*.*
        └── release.yml:
              ├── docker build + push GHCR
              │   (tags: semver, major.minor, sha)
              └── sed update newTag in k8s/overlays/prod/kustomization.yaml
                  + git push [skip ci]
```
