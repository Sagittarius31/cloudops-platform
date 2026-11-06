KIND_CLUSTER ?= sentinel
.PHONY: set-github kind-up kind-deploy kind-forward kind-down tidy test run up down fmt helm-lint helm-modes tf-validate argocd-install load

tidy:            ## generate go.sum (run once, then commit it)
	cd app && go mod tidy

test:
	cd app && go vet ./... && go test -race -cover ./...

run:             ## run API locally (needs Postgres, see `make up`)
	cd app && go run ./cmd/server

up:              ## full local stack: API + Postgres + Prometheus + Grafana
	cd deploy && docker compose up --build -d

down:
	cd deploy && docker compose down -v

fmt:
	cd app && gofmt -l -w .
	terraform fmt -recursive infra

helm-lint:
	helm lint charts/sentinel --set image.repository=example/sentinel

helm-modes:      ## render all three secret modes
	helm template t charts/sentinel --set image.repository=x > /dev/null
	helm template t charts/sentinel --set image.repository=x --set postgres.enabled=false --set database.existingSecret=s > /dev/null
	helm template t charts/sentinel -f charts/sentinel/values-rds.yaml --set image.repository=x > /dev/null
	helm template t charts/sentinel -f charts/sentinel/values-aks.yaml -f charts/sentinel/values-aks-kv.yaml > /dev/null
	helm template t charts/sentinel --set image.repository=x --set ingress.enabled=true --set ingress.host=h.example.com > /dev/null

tf-validate:
	cd infra/terraform/envs/dev && terraform init -backend=false && terraform validate
	cd infra/terraform/envs/azure-dev && terraform init -backend=false && terraform validate

argocd-install:  ## install ArgoCD into the current kubectl context
	kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -n argocd --server-side -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

load:
	k6 run -e BASE_URL=http://localhost:8080 -e TOKEN=local-dev-token-0123456789 loadtest/api.js

# ---- Free local Kubernetes (kind = Kubernetes inside Docker). Works in Codespaces or any machine with Docker ----
kind-up:         ## create a local cluster and build+load the image
	command -v kind >/dev/null || go install sigs.k8s.io/kind@latest
	kind create cluster --name $(KIND_CLUSTER) || true
	docker build -t sentinel:dev app
	kind load docker-image sentinel:dev --name $(KIND_CLUSTER)

kind-deploy:     ## install the Helm chart (in-cluster Postgres, 1 replica, no HPA)
	helm upgrade --install sentinel charts/sentinel -n sentinel --create-namespace \
	  --set image.repository=sentinel --set image.tag=dev --set image.pullPolicy=Never \
	  --set replicaCount=1 --set autoscaling.enabled=false --set config.allowPrivateTargets=true
	kubectl -n sentinel rollout status deploy/sentinel --timeout=180s

kind-forward:    ## expose the API on localhost:8080 (token: change-me-dev-only-token-0123)
	kubectl -n sentinel port-forward svc/sentinel 8080:80

kind-down:
	kind delete cluster --name $(KIND_CLUSTER)

# One command to fill in your GitHub username everywhere it is needed (Linux/Codespaces sed).
#   make set-github GITHUB_USER=yourname
set-github:
	@test -n "$(GITHUB_USER)" || { echo "usage: make set-github GITHUB_USER=yourname"; exit 1; }
	grep -rl 'YOUR_GITHUB_USERNAM[E]' gitops infra charts | xargs sed -i 's/YOUR_GITHUB_USERNAM[E]/$(GITHUB_USER)/g'
	@echo "Done. Remaining placeholders: grep -rn REPLACE_ gitops charts infra"
