# go-k8s-healthcheck

Production-grade Go HTTP service sebagai referensi belajar Kubernetes best practices.

## Endpoints

| Path | Probe | Deskripsi |
|------|-------|-----------|
| `GET /` | — | Info app: name, version, env, uptime |
| `GET /healthz` | Liveness | `200 OK` jika proses hidup |
| `GET /readyz` | Readiness | `200 OK` siap terima traffic, `503` jika belum |

**Contoh response `GET /`:**
```json
{
  "app": "go-k8s-healthcheck",
  "version": "dev",
  "env": "development",
  "uptime": "34s",
  "status": "running"
}
```

---

## Struktur Repository

```
go-k8s-healthcheck/
├── cmd/server/main.go          # Entrypoint, graceful shutdown
├── internal/
│   ├── config/config.go        # Konfigurasi via env vars
│   └── handler/health.go       # HTTP handlers
├── k8s/
│   ├── base/                   # Kustomize base (semua resource K8s)
│   │   ├── namespace.yaml      # Namespace + Pod Security Standards
│   │   ├── serviceaccount.yaml # ServiceAccount dedicated
│   │   ├── configmap.yaml      # Konfigurasi non-sensitif
│   │   ├── deployment.yaml     # Deployment + probe + securityContext
│   │   ├── service.yaml        # ClusterIP service
│   │   ├── hpa.yaml            # HorizontalPodAutoscaler
│   │   ├── pdb.yaml            # PodDisruptionBudget
│   │   ├── networkpolicy.yaml  # Default deny + whitelist
│   │   └── kustomization.yaml
│   └── overlays/
│       ├── dev/                # 1 replica, debug logging, resource kecil
│       └── prod/               # 3 replicas, info logging, resource produksi
├── .github/workflows/
│   ├── ci.yml                  # Lint, test, build on push/PR
│   └── release.yml             # Build & push Docker on semver tag
├── Dockerfile                  # Multi-stage, distroless, non-root
├── Makefile                    # Shortcut commands
└── README.md
```

---

## Konfigurasi (Environment Variables)

| Variable | Default | Deskripsi |
|----------|---------|-----------|
| `PORT` | `8080` | HTTP listen port |
| `APP_ENV` | `development` | Environment name |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `VERSION` | `dev` | App version (diset oleh CI/overlay) |
| `SHUTDOWN_TIMEOUT_SEC` | `30` | Waktu tunggu graceful shutdown (detik) |

---

## Prerequisites

- Docker Engine
- k3s (atau Kubernetes cluster lain)
- `kubectl`
- `make`
- Go 1.24+ (untuk development lokal)

---

## Deployment ke k3s

### 1. Clone repository

```bash
git clone https://github.com/aldi-go/go-k8s-healthcheck.git
cd go-k8s-healthcheck
```

### 2. Build Docker image

```bash
make docker-build
```

Image akan di-tag sebagai:
- `ghcr.io/plabs/go-k8s-healthcheck:<git-short-sha>`
- `ghcr.io/plabs/go-k8s-healthcheck:latest`

> **Catatan k3s:** k3s menggunakan `containerd` sebagai container runtime, **terpisah dari Docker daemon**.
> Image yang di-build via `docker build` tidak otomatis tersedia di k3s.

### 3. Import image ke containerd k3s

Langkah ini diperlukan untuk deployment lokal tanpa registry eksternal:

```bash
# Tag sebagai 'dev' untuk overlay dev
docker tag ghcr.io/plabs/go-k8s-healthcheck:latest ghcr.io/plabs/go-k8s-healthcheck:dev

# Export dari Docker dan import ke containerd k3s
docker save ghcr.io/plabs/go-k8s-healthcheck:dev | k3s ctr images import -
```

Verifikasi image sudah masuk ke containerd:

```bash
k3s ctr images ls | grep go-k8s-healthcheck
```

### 4. Deploy ke cluster

**Dev overlay** (1 replica, debug logging):

```bash
make deploy-dev
# atau
kubectl apply -k k8s/overlays/dev
```

**Prod overlay** (3 replicas, production resources):

```bash
make deploy-prod
# atau
kubectl apply -k k8s/overlays/prod
```

### 5. Verifikasi deployment

```bash
# Cek semua resource di namespace healthcheck
kubectl get all -n healthcheck

# Tunggu deployment siap
kubectl wait deployment go-k8s-healthcheck \
  -n healthcheck \
  --for=condition=Available \
  --timeout=60s

# Cek detail pod (node, IP, status)
kubectl get pods -n healthcheck -o wide

# Cek events jika ada masalah
kubectl describe deployment go-k8s-healthcheck -n healthcheck
```

Output yang diharapkan:
```
NAME                                      READY   STATUS    RESTARTS   AGE
pod/go-k8s-healthcheck-5b5b85c54c-dthdj   1/1     Running   0          30s

NAME                         TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
service/go-k8s-healthcheck   ClusterIP   10.43.x.x       <none>        80/TCP    30s

NAME                                 READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/go-k8s-healthcheck   1/1     1            1           30s
```

### 6. Test endpoints

```bash
# Ambil ClusterIP service
CLUSTER_IP=$(kubectl get svc go-k8s-healthcheck -n healthcheck \
  -o jsonpath='{.spec.clusterIP}')

curl http://$CLUSTER_IP/       # info
curl http://$CLUSTER_IP/healthz  # liveness
curl http://$CLUSTER_IP/readyz   # readiness
```

### 7. Lihat logs (structured JSON)

```bash
# Logs real-time
kubectl logs -f -l app.kubernetes.io/name=go-k8s-healthcheck -n healthcheck

# Logs dari semua pod sekaligus
kubectl logs -l app.kubernetes.io/name=go-k8s-healthcheck \
  -n healthcheck \
  --prefix=true \
  --tail=50
```

---

## Dry-run sebelum apply

Selalu periksa diff sebelum apply ke cluster (terutama prod):

```bash
make diff-dev   # kubectl diff -k k8s/overlays/dev
make diff-prod  # kubectl diff -k k8s/overlays/prod
```

---

## Scaling Manual

```bash
# Scale up sementara
kubectl scale deployment go-k8s-healthcheck -n healthcheck --replicas=3

# Lihat HPA status
kubectl get hpa -n healthcheck

# Describe HPA untuk detail metrics
kubectl describe hpa go-k8s-healthcheck -n healthcheck
```

---

## Update Image (Rolling Update)

```bash
# Build image baru dengan tag baru
docker build -t ghcr.io/plabs/go-k8s-healthcheck:v1.1.0 .

# Import ke k3s
docker save ghcr.io/plabs/go-k8s-healthcheck:v1.1.0 | k3s ctr images import -

# Update image di deployment (zero-downtime rolling update)
kubectl set image deployment/go-k8s-healthcheck \
  server=ghcr.io/plabs/go-k8s-healthcheck:v1.1.0 \
  -n healthcheck

# Pantau proses rollout
kubectl rollout status deployment/go-k8s-healthcheck -n healthcheck
```

**Rollback jika ada masalah:**

```bash
kubectl rollout undo deployment/go-k8s-healthcheck -n healthcheck

# Rollback ke revisi tertentu
kubectl rollout history deployment/go-k8s-healthcheck -n healthcheck
kubectl rollout undo deployment/go-k8s-healthcheck -n healthcheck --to-revision=2
```

---

## Cleanup

### Hapus deployment saja (namespace tetap ada)

```bash
kubectl delete deployment go-k8s-healthcheck -n healthcheck
kubectl delete svc go-k8s-healthcheck -n healthcheck
```

### Hapus semua resource overlay dev

```bash
kubectl delete -k k8s/overlays/dev
```

### Hapus semua resource overlay prod

```bash
kubectl delete -k k8s/overlays/prod
```

### Hapus namespace beserta semua isinya

> ⚠️ Ini menghapus **semua** resource di namespace `healthcheck` secara permanen.

```bash
kubectl delete namespace healthcheck
```

### Hapus image dari containerd k3s

```bash
k3s ctr images rm ghcr.io/plabs/go-k8s-healthcheck:dev
k3s ctr images rm ghcr.io/plabs/go-k8s-healthcheck:latest
```

### Hapus image dari Docker

```bash
docker rmi ghcr.io/plabs/go-k8s-healthcheck:dev
docker rmi ghcr.io/plabs/go-k8s-healthcheck:latest
```

---

## Troubleshooting

**Pod tidak mau Running (`ImagePullBackOff` / `ErrImageNeverPull`)**

Image belum diimport ke containerd k3s. Jalankan langkah import di bagian [3. Import image ke containerd k3s](#3-import-image-ke-containerd-k3s).

```bash
# Cek apakah image ada di containerd
k3s ctr images ls | grep go-k8s-healthcheck
```

**Pod status `Pending`**

```bash
kubectl describe pod -l app.kubernetes.io/name=go-k8s-healthcheck -n healthcheck
# Cek bagian Events: di output untuk detail masalah
```

**Pod restart terus (`CrashLoopBackOff`)**

```bash
# Lihat logs dari pod yang crash
kubectl logs -l app.kubernetes.io/name=go-k8s-healthcheck \
  -n healthcheck --previous
```

**Readiness probe gagal**

```bash
# Verifikasi endpoint readyz dari dalam cluster
CLUSTER_IP=$(kubectl get svc go-k8s-healthcheck -n healthcheck \
  -o jsonpath='{.spec.clusterIP}')
curl -v http://$CLUSTER_IP/readyz
```

---

## Kubernetes Best Practices yang Diterapkan

### Deployment
- `RollingUpdate` dengan `maxUnavailable: 0` — zero-downtime deploy
- `livenessProbe`, `readinessProbe`, `startupProbe` — health management lengkap
- `resources.requests` & `limits` — mencegah resource starvation
- `terminationGracePeriodSeconds: 60` — cukup untuk drain in-flight requests
- `podAntiAffinity` — preferensikan spread pods ke node berbeda
- `securityContext` — non-root, read-only FS, drop ALL capabilities, seccomp

### Availability
- `HorizontalPodAutoscaler` — auto-scale 2–5 replicas berdasarkan CPU/memory
- `PodDisruptionBudget` — jamin minimal 1 pod saat node drain / cluster upgrade

### Security
- `ServiceAccount` dedicated, `automountServiceAccountToken: false`
- `NetworkPolicy` — default deny-all ingress+egress, whitelist explicit
- Namespace dengan `pod-security.kubernetes.io/enforce: restricted`

### Docker
- Multi-stage build: `golang:1.24-alpine` (builder) + `distroless/static-debian12:nonroot` (final)
- Image final tanpa shell, tanpa package manager, berjalan sebagai UID 65532
- Final image size ~3.5 MB

---

## CI/CD Flow

```
push ke main/develop
  └── ci.yml
        ├── go vet
        ├── go test -race
        └── docker build (no push)

push tag v*.*.*
  └── release.yml
        ├── docker build + push ke GHCR
        │   tags: v1.2.3, v1.2, sha-xxxxxxx
        └── auto-update newTag di k8s/overlays/prod/kustomization.yaml
```

---

## Run Lokal (tanpa Docker/Kubernetes)

```bash
make run
# Server berjalan di http://localhost:8080

curl http://localhost:8080/
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

Build binary lokal:

```bash
make build
./bin/server
```
