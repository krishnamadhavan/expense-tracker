# expense-tracker Helm chart

Deploy the Go API + embedded UI to **Kubernetes**, **Minikube**, or **OpenShift**.

## Prerequisites

- Helm 3.10+
- Cluster access (`kubectl` / `oc`)
- Container image built and pushable (default `ghcr.io/krishnamadhavan/expense-tracker:latest`), **or** load into Minikube:

```bash
docker build -t ghcr.io/krishnamadhavan/expense-tracker:latest .
minikube image load ghcr.io/krishnamadhavan/expense-tracker:latest
```

## Install (Minikube + bundled Postgres)

```bash
cd deploy/helm/expense-tracker
# Postgres (example Bitnami, separate release)
helm repo add bitnami https://charts.bitnami.com/bitnami
helm upgrade --install et-pg bitnami/postgresql -n expense-tracker --create-namespace \
  --set auth.username=expense --set auth.password=expense --set auth.database=expense_tracker

# App chart (points at et-pg-postgresql service when using defaults in values)
helm upgrade --install expense-tracker . \
  --namespace expense-tracker \
  -f values-minikube.yaml \
  --set secret.databaseUrl='postgres://expense:expense@et-pg-postgresql:5432/expense_tracker?sslmode=disable'

minikube addons enable ingress
# /etc/hosts: <minikube ip> expense.local
```

## Install (OpenShift)

```bash
helm upgrade --install expense-tracker . \
  --namespace expense-tracker --create-namespace \
  -f values-openshift.yaml
oc get route -n expense-tracker
```

Bitnami PostgreSQL on OpenShift may need extra SCC tweaks; prefer **external DB** (`values-external-db.yaml`) if the subchart fails restricted SCC.

## External database only

```bash
helm upgrade --install expense-tracker . \
  --namespace expense-tracker --create-namespace \
  -f values-external-db.yaml \
  --set secret.databaseUrl='postgres://user:pass@host:5432/expense_tracker?sslmode=require' \
  --set secret.bootstrapPassword='your-admin-password'
```

## Useful commands

```bash
helm template expense-tracker . -f values-minikube.yaml | less
helm lint .
kubectl -n expense-tracker rollout status deploy/expense-tracker
kubectl -n expense-tracker logs -l app.kubernetes.io/name=expense-tracker -c api
```

## Values overview

| Key | Purpose |
| --- | --- |
| `image.*` | App container |
| `postgresql.enabled` | Bundle Bitnami Postgres |
| `secret.databaseUrl` | App DSN (auto from subchart if empty and PG enabled) |
| `secret.bootstrapPassword` | First admin password |
| `secret.existingSecret` | Use your own Secret keys `database-url`, `bootstrap-password`, `migrate-database-url` |
| `migrate.*` | Init-container migrations from `files/migrations` |
| `ingress.*` | K8s Ingress (disabled path on OpenShift when `openshift.enabled`) |
| `openshift.enabled` + `route.*` | OpenShift Route |

Change default passwords before any shared environment.
