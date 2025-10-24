ADO_ORG ?= your-ado-organization
ADO_PROJECT ?= your-ado-project
ADO_REPO ?= your-ado-repo
ADO_PAT ?= your-ado-pat

export ADO_ORG
export ADO_PROJECT
export ADO_REPO
export ADO_PAT

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: build
build: fmt vet build-linux build-windows ## Build manager binary.

.PHONY: run
run: build
	go run main.go \
		--ado-organization="$(ADO_ORG)" \
		--ado-project="$(ADO_PROJECT)" \
		--ado-repo="$(ADO_REPO)" \
		--ado-pat="$(ADO_PAT)"

.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/avm-sync-linux main.go

.PHONY: build-windows
build-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/avm-sync-windows.exe main.go

.PHONY: download-csv-files
download-csv-files:
	curl -o third_party/module-indexes/TerraformResourceModules.csv https://raw.githubusercontent.com/Azure/Azure-Verified-Modules/refs/heads/main/docs/static/module-indexes/TerraformResourceModules.csv
	curl -o third_party/module-indexes/TerraformPatternModules.csv https://raw.githubusercontent.com/Azure/Azure-Verified-Modules/refs/heads/main/docs/static/module-indexes/TerraformPatternModules.csv
	curl -o third_party/module-indexes/TerraformUtilityModules.csv https://raw.githubusercontent.com/Azure/Azure-Verified-Modules/refs/heads/main/docs/static/module-indexes/TerraformUtilityModules.csv
