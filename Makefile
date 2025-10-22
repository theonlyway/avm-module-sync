.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: build
build: fmt vet ## Build manager binary.
	go build -o bin/avm-sync main.go

.PHONY: run
run:
	go run main.go \
		--ado-organization="$(ADO_ORG)" \
		--ado-project="$(ADO_PROJECT)" \
		--ado-repo="$(ADO_REPO)" \
		--ado-pat="$(ADO_PAT)"
