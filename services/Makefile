.PHONY: help dev-build up down logs clean

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


## PROD 
AWS_ACCOUNT_ID ?= $(shell echo $$AWS_ACCOUNT_ID)
AWS_REGION ?= $(shell echo $$AWS_REGION)
ECR_REPO_PREFIX ?= minimaldo
SERVICE ?= $(shell echo $$SERVICE)
IMAGE_NAME ?= $(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/$(ECR_REPO_PREFIX)/$(SERVICE)
#TAG ?= $(git rev-parse --abbrev-ref HEAD)-$(git rev-parse --short HEAD)-$(date +%s) 
TAG ?= $(shell echo $$TAG) 
## tag: branch-gitcommit-time

prod-image-build: ## Build Image for Prod (Need to pass AWS_ACCOUNT_ID, AWS_REGION, SERVICE)
	@TIME=$$(echo "$(TAG)" | awk -F'-' '{print $$3}'); \
	echo "Build Time: $$(date -d @$${TIME})"
	@docker compose build $(SERVICE)
	@docker tag services-$(SERVICE):latest $(IMAGE_NAME):latest
	@docker tag services-$(SERVICE):latest $(IMAGE_NAME):$(TAG)
	@docker rmi services-$(SERVICE):latest

prod-image-push:
	@docker push $(IMAGE_NAME):$(TAG)
	@docker push $(IMAGE_NAME):latest

prod-image-cleanup:
	@docker rmi $(IMAGE_NAME):$(TAG)
	@docker rmi $(IMAGE_NAME):latest

## DEV 

dev-build: ## Build all Docker images
	docker compose build

up: ## Start all services
	docker compose up -d
	@echo "Services started!"
	@echo "Frontend: http://localhost:3000"
	@echo "Backend API: http://localhost:8080"
	@echo "PostgreSQL: localhost:5432"

down: ## Stop all services
	docker compose down

logs: ## Show logs
	docker compose logs -f

clean: ## Clean up everything
	docker compose down -v --remove-orphans
	docker rmi minimaldo-frontend minimaldo-backend

