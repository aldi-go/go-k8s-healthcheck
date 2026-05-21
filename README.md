# go-k8s-healthcheck

Production-grade Go HTTP service sebagai referensi belajar Kubernetes best practices.

## Endpoints

| Path | Probe | Deskripsi |
|------|-------|-----------|
| `GET /` | — | Info app: name, version, env, uptime |
| `GET /healthz` | Liveness | 200 jika proses hidup |
| `GET /readyz` | Readiness | 200 jika siap terima traffic, 503 jika belum |

## Struktur Repository

```
go-k8s-healthcheck/
├── cmd/server/main.go          # Entrypoint, graceful shutdown
├── internal/
│   ├── config/config.go        # Konfigurasi via env vars
│   └── handler/health.go       # HTTP handlers
├── k8s/
│   ├── base/                   # Kustomize base (semua resource)
│   └── overlays/
│       ├── dev/                # 1 replica, debug logging
│       └── prod/               # 3 replicas, resource limits ketat
├── .github/workflows/
│   ├── ci.yml                  # Lint, test, build on push
│   └── release.yml             # Build & push Docker on tag
├── Dockerfile                  # Multi-stage, distroless, non-root
└── Makefile                    # Shortcut commands
```

## Konfigurasi (Environment Variables)

| Variable | Default | Deskripsi |
|----------|---------|-----------|
| `PORT` | `8080` | HTTP listen port |
| `APP_ENV` | `development` | Environment name |
| `LOG_LEVEL` | `info` | debug / info / warn / error |
| `VERSION` | `dev` | App version (diset oleh CI) |
| `SHUTDOWN_TIMEOUT_SEC` | `30` | Waktu tunggu graceful shutdown |

## Kubernetes Best Practices yang Diterapkan

### Deployment
- `RollingUpdate` dengan `maxUnavailable: 0` — zero-downtime deploy
- `livenessProbe`, `readinessProbe`, `startupProbe` — health management lengkap
- `resources.requests` & `limits` — mencegah resource starvation
- `terminationGracePeriodSeconds: 60` — cukup untuk drain in-flight requests
- `podAntiAffinity` — spread pods ke node berbeda
- `securityContext` — non-root, read-only FS, drop ALL capabilities, seccomp

### Availability
- `HorizontalPodAutoscaler` — auto-scale berdasarkan CPU/memory
- `PodDisruptionBudget` — jamin minimal 1 pod saat node drain

### Security
- `ServiceAccount` dedicated, `automountServiceAccountToken: false`
- `NetworkPolicy` — default deny all, whitelist explicit
- Namespace dengan `pod-security.kubernetes.io/enforce: restricted`

### Docker
- Multi-stage build (builder + distroless final)
- Image final `gcr.io/distroless/static-debian12:nonroot`
- Tidak ada shell, tidak ada package manager di image final

## Cara Deploy ke k3s

### Build image lokal
```bash
make docker-build
```

### Deploy ke dev overlay
```bash
make deploy-dev
# atau manual:
kubectl apply -k k8s/overlays/dev
```

### Deploy ke prod overlay
```bash
make deploy-prod
```

### Cek status
```bash
kubectl get all -n healthcheck
kubectl logs -l app.kubernetes.io/name=go-k8s-healthcheck -n healthcheck
```

### Dry-run diff sebelum apply
```bash
make diff-dev
make diff-prod
```

## CI/CD Flow

```
push ke main/develop  →  ci.yml  →  lint + test + docker build (no push)
push tag v*.*.*       →  release.yml  →  build + push GHCR + update prod kustomize tag
```

## Run Lokal

```bash
make run
# curl http://localhost:8080/
# curl http://localhost:8080/healthz
# curl http://localhost:8080/readyz
```
