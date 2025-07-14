# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

ifeq ($(shell uname),Darwin)
SED_COMMAND=sed -i '' -n '/rules/,$$p'
else
SED_COMMAND=sed -i -n '/rules/,$$p'
endif

# generate: generate-controller generate-groups rbacs manifests fmt
generate: generate-controller rbacs manifests fmt

#generate helm documentation
docs: helm-docs
	$(HELM_DOCS) -t deployments/operator/README.gotmpl deployments/operator

manifests: controller-gen
	rm -f deployments/operator/crds/*
	$(CONTROLLER_GEN) paths="./api/..." crd:generateEmbeddedObjectMeta=true output:crd:artifacts:config=deployments/operator/crds
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=deployments/operator/crds

#Generate RBAC for each controller
rbacs: controller-gen
	rm -f deployments/operator/files/*

	$(CONTROLLER_GEN) paths="./pkg/company-controller" rbac:roleName=console-operator-company-controller output:rbac:stdout | awk -v RS="---\n" 'NR>1{f="./deployments/operator/files/console-operator-company-controller-" $$4 ".yaml";printf "%s",$$0 > f; close(f)}' && $(SED_COMMAND) deployments/operator/files/console-operator-company-controller-ClusterRole.yaml

# Install gci if not available
gci:
ifeq (, $(shell which gci))
	@go install github.com/daixiang0/gci@v0.11.2
GCI=$(GOBIN)/gci
else
GCI=$(shell which gci)
endif

# Install addlicense if not available
addlicense:
ifeq (, $(shell which addlicense))
	@go install github.com/google/addlicense@v1.0.0
ADDLICENSE=$(GOBIN)/addlicense
else
ADDLICENSE=$(shell which addlicense)
endif

# Run go fmt against code
fmt: gci addlicense
	go mod tidy
	go fmt ./...
	find . -type f -name '*.go' -a ! -name '*zz_generated*' -exec $(GCI) write -s standard -s default -s "prefix(github.com/mia-platform/console-operator)" {} \;
	find . -type f -name '*.go' -exec $(ADDLICENSE) -l apache -c "Mia-Platform" -y "2016-$(shell date +%Y)" {} \;

vet: ## Run go vet against code.
	go vet ./...

# Install golangci-lint if not available
golangci-lint:
ifeq (, $(shell which golangci-lint))
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@v2.1.0
GOLANGCILINT=$(GOBIN)/golangci-lint
else
GOLANGCILINT=$(shell which golangci-lint)
endif

markdownlint:
ifeq (, $(shell which markdownlint))
	@echo "markdownlint is not installed. Please install it: https://github.com/igorshubovych/markdownlint-cli#installation"
	@exit 1
else
MARKDOWNLINT=$(shell which markdownlint)
endif

md-lint: markdownlint
	@find . -type f -name '*.md' -a -not -path "./.github/*" \
		-not -path "./docs/_legacy/*" \
		-not -path "./deployments/*" \
		-not -path "./hack/code-generator/*" \
		-exec $(MARKDOWNLINT) {} +

lint: golangci-lint
	$(GOLANGCILINT) run --new

generate-controller: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./api/..."


# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.18.0
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif


helm-docs:
ifeq (, $(shell which helm-docs))
	@{ \
	set -e ;\
	HELM_DOCS_TMP_DIR=$$(mktemp -d) ;\
	cd $$HELM_DOCS_TMP_DIR ;\
	version=1.11.0 ;\
    arch=x86_64 ;\
    echo  $$HELM_DOCS_PATH ;\
    echo https://github.com/norwoodj/helm-docs/releases/download/v$${version}/helm-docs_$${version}_linux_$${arch}.tar.gz ;\
    curl -LO https://github.com/norwoodj/helm-docs/releases/download/v$${version}/helm-docs_$${version}_linux_$${arch}.tar.gz ;\
    tar -zxvf helm-docs_$${version}_linux_$${arch}.tar.gz ;\
    mv helm-docs $(GOBIN)/helm-docs ;\
	rm -rf $$HELM_DOCS_TMP_DIR ;\
	}
HELM_DOCS=$(GOBIN)/helm-docs
else
HELM_DOCS=$(shell which helm-docs)
endif




#.PHONY: lint
#lint: golangci-lint ## Run golangci-lint linter
#	$(GOLANGCI_LINT) run
#
#.PHONY: lint-fix
#lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
#	$(GOLANGCI_LINT) run --fix
#
#.PHONY: lint-config
#lint-config: golangci-lint ## Verify golangci-lint linter configuration
#	$(GOLANGCI_LINT) config verify
#
#
#
#
#
#PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
#.PHONY: docker-buildx
#docker-buildx: ## Build and push docker image for the manager for cross-platform support
#	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
#	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
#	- $(CONTAINER_TOOL) buildx create --name console-operator-builder
#	$(CONTAINER_TOOL) buildx use console-operator-builder
#	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross .
#	- $(CONTAINER_TOOL) buildx rm console-operator-builder
#	rm Dockerfile.cross
#
#.PHONY: build-installer
#build-installer: manifests generate kustomize ## Generate a consolidated YAML with CRDs and deployment.
#	mkdir -p dist
#	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
#	$(KUSTOMIZE) build config/default > dist/install.yaml
