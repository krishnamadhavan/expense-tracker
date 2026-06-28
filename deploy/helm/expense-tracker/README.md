# expense-tracker Helm chart

Deploy the Go API + embedded UI to **Kubernetes**, **Minikube**, or **OpenShift**.

**PostgreSQL is a chart dependency** (Bitnami). With `postgresql.enabled=true` (default), `helm dependency update` vendors the subchart and one `helm install` brings up **DB + app**. Set `postgresql.enabled=false` and `secret.databaseUrl` for an external database.

## Prerequisites

- Helm 3.10+
- Cluster access (`kubectl` / `oc`)
- Network access to `charts.bitnami.com` once (for `helm dependency update`), **or** a pre-vendored `charts/` directory committed/CI-cached
- App image available on the cluster (default `ghcr.io/krishnamadhavan/expense-tracker:latest`), e.g.:

```bash
docker build -t ghcr.io/krishnamadhavan/expense-tracker:latest .
minikube image load ghcr.io/krishnamadhavan/expense-tracker:latest
```

## Install (self-contained: app + Postgres)

```bash
cd deploy/helm/expense-tracker
helm dependency update

helm upgrade --install expense-tracker . \
  --namespace expense-tracker --create-namespace \
  -f values-minikube.yaml

# Minikube ingress
minikube addons enable ingress
# /etc/hosts: $(minikube ip) expense.local
```

Default DB URL is derived as:

`postgres://expense:expense@<release>-postgresql:5432/expense_tracker?sslmode=disable`

Override passwords via `postgresql.auth.*` and/or `secret.bootstrapPassword`. Prefer `--set` / sealed secrets in real environments.

## Install (OpenShift)

```bash
cd deploy/helm/expense-tracker
helm dependency update

helm upgrade --install expense-tracker . \
  --namespace expense-tracker --create-namespace \
  -f values-openshift.yaml
oc get route -n expense-tracker
```

If the Bitnami subchart fails restricted SCC, either adjust `postgresql.primary.podSecurityContext` / `volumePermissions`, or set `postgresql.enabled=false` and point `secret.databaseUrl` at platform Postgres.

## External database only

```bash
helm upgrade --install expense-tracker . \
  --namespace expense-tracker --create-namespace \
  -f values-external-db.yaml \
  --set secret.databaseUrl='postgres://user:pass@host:5432/expense_tracker?sslmode=require' \
  --set secret.bootstrapPassword='your-admin-password'
```

(`values-external-db.yaml` sets `postgresql.enabled=false`.)

## Useful commands

```bash
helm dependency update
helm lint .
helm template expense-tracker . -f values-minikube.yaml | less
kubectl -n expense-tracker rollout status deploy/expense-tracker
kubectl -n expense-tracker logs -l app.kubernetes.io/name=expense-tracker -c api
kubectl -n expense-tracker port-forward svc/expense-tracker 8080:80
```

## Values overview

| Key | Purpose |
| --- | --- |
| `image.*` | App container |
| `postgresql.enabled` | Install Bitnami Postgres subchart (default **true**) |
| `postgresql.auth.*` | DB user / password / database name |
| `secret.databaseUrl` | Override app DSN (optional if subchart enabled) |
| `secret.bootstrapPassword` | First admin password |
| `secret.existingSecret` | Use your own Secret (`database-url`, `bootstrap-password`, `migrate-database-url`) |
| `migrate.*` | Init-container migrations from `files/migrations` |
| `ingress.*` | K8s Ingress (skipped when `openshift.enabled`) |
| `openshift.enabled` + `route.*` | OpenShift Route |

Change default passwords before any shared environment.
