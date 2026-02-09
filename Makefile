CHART_OUTPUT := helm-chart
HELM_GEN     := helm-gen
K8S_DIR      := kubernetes

# Initialize and update git submodules
.PHONY: submodule
submodule:
	git submodule update --init
	git submodule foreach git pull origin master

.PHONY: helm-gen-build
helm-gen-build:
	cd $(HELM_GEN) && go build -o ../bin/helm-gen .

.PHONY: helm-gen-test
helm-gen-test:
	cd $(HELM_GEN) && go test ./...

# Full pipeline: render + generate
.PHONY: helm-chart
helm-chart: helm-gen-build
	cd $(K8S_DIR) && yarn install --frozen-lockfile
	cd $(K8S_DIR) && rm -rf rendered-manifests
	cd $(K8S_DIR) && pulumi stack select helm-render --create 2>/dev/null; \
		pulumi up --yes --skip-preview --stack helm-render
	./bin/helm-gen \
		--input-dir $(K8S_DIR)/rendered-manifests/1-manifest \
		--output-dir $(CHART_OUTPUT)/pulumi-deployment-agent

# Generate from existing rendered manifests (skip Pulumi render)
.PHONY: helm-chart-quick
helm-chart-quick: helm-gen-build
	./bin/helm-gen \
		--input-dir $(K8S_DIR)/rendered-manifests/1-manifest \
		--output-dir $(CHART_OUTPUT)/pulumi-deployment-agent

.PHONY: helm-lint
helm-lint:
	helm lint --strict $(CHART_OUTPUT)/pulumi-deployment-agent
	helm template test $(CHART_OUTPUT)/pulumi-deployment-agent \
		--set agent.token=test-token

.PHONY: clean-helm
clean-helm:
	rm -rf $(CHART_OUTPUT) bin/helm-gen
	cd $(K8S_DIR) && rm -rf rendered-manifests
