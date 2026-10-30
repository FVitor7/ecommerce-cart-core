# Every target runs in Docker through docker-compose.yml: the host needs only
# Docker and make, and the working tree gets no .venv or node_modules.
# Artifacts are copied out of the images into dist/.

# Default versions; override per call, e.g. make test-python PYTHON_VERSION=3.10
GO_VERSION     ?= 1.27
PYTHON_VERSION ?= 3.13
NODE_VERSION   ?= 24
TINYGO_VERSION ?= 0.43.0
export GO_VERSION PYTHON_VERSION NODE_VERSION TINYGO_VERSION

# Versions covered by test-versions.
GO_VERSIONS     := 1.25 1.26 1.27
PYTHON_VERSIONS := 3.9 3.10 3.11 3.12 3.13 3.14 3.15
NODE_VERSIONS   := 18 20 22 24 26

COMPOSE := docker compose
RUN     := $(COMPOSE) run --rm -T --quiet-build

# $(call copy-out,<image>,<path in the image>,<host path>)
copy-out = id=$$(docker create $(1) none) && docker cp -q $$id:$(2) $(3); \
	status=$$?; docker rm $$id >/dev/null; exit $$status

.DEFAULT_GOAL := help
.PHONY: help build build-go build-python build-js test test-go test-python test-js \
	test-nextjs test-versions test-go-versions test-python-versions test-js-versions \
	demo nextjs fixtures clean

help: ## list the targets
	@echo "Usage: make <target>"
	@awk 'BEGIN {FS = ":.*## "} /^##@ / {printf "\n%s\n", substr($$0, 5)} \
		/^[a-z0-9%.-]+:.*## / {printf "  %-22s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo
	@echo "Default versions: GO_VERSION=$(GO_VERSION) PYTHON_VERSION=$(PYTHON_VERSION) NODE_VERSION=$(NODE_VERSION)"
	@echo "test-versions:    Go $(GO_VERSIONS) | Python $(PYTHON_VERSIONS) | Node.js $(NODE_VERSIONS)"

##@ Build (artifacts in dist/)

build: build-go build-python build-js ## calc.wasm and the Python and npm packages

build-go: ## calc.wasm, compiled with TinyGo
	$(COMPOSE) build wasm
	@mkdir -p dist && $(call copy-out,ecommerce-cart-core/wasm,/calc.wasm,dist/calc.wasm)
	@ls -l dist/calc.wasm

build-python: ## Python package: wheel and sdist
	$(COMPOSE) build python
	@rm -rf dist/python && mkdir -p dist/python \
		&& $(call copy-out,ecommerce-cart-core/python:$(PYTHON_VERSION),/dist/.,dist/python)
	@ls -l dist/python

build-js: ## npm package (.tgz)
	$(COMPOSE) build js
	@rm -rf dist/js && mkdir -p dist/js \
		&& $(call copy-out,ecommerce-cart-core/js:$(NODE_VERSION),/dist/.,dist/js)
	@ls -l dist/js

##@ Tests (on the default versions)

test: test-go test-python test-js test-nextjs ## every test suite

test-go: ## gofmt, go vet and go test
	@echo "==> Go $(GO_VERSION)"
	$(COMPOSE) build go
	$(RUN) go

test-python: ## pytest on the package installed from the wheel
	@echo "==> Python $(PYTHON_VERSION)"
	$(COMPOSE) build python
	$(RUN) python

test-js: ## node --test on the files of the packed package
	@echo "==> Node.js $(NODE_VERSION)"
	$(COMPOSE) build js
	$(RUN) js

test-nextjs: ## production build of the Next.js example, then a test of its page and API
	$(COMPOSE) build nextjs
	$(RUN) nextjs node smoke-test.mjs

##@ Tests per version

test-versions: test-go-versions test-python-versions test-js-versions ## every suite on every version

test-go-versions: $(GO_VERSIONS:%=test-go-%) ## Go on every version
test-python-versions: $(PYTHON_VERSIONS:%=test-python-%) ## Python on every version
test-js-versions: $(NODE_VERSIONS:%=test-js-%) ## Node.js on every version

test-go-%: ## one version, e.g. make test-go-1.25
	@$(MAKE) --no-print-directory test-go GO_VERSION=$*
test-python-%: ## one version, e.g. make test-python-3.9
	@$(MAKE) --no-print-directory test-python PYTHON_VERSION=$*
test-js-%: ## one version, e.g. make test-js-18
	@$(MAKE) --no-print-directory test-js NODE_VERSION=$*

##@ Examples

demo: ## the same checkout in Python and in Node.js, with the outputs compared
	$(COMPOSE) build python js
	@out=$$(mktemp -d) \
		&& $(RUN) python python /src/examples/python/checkout.py > $$out/python.json \
		&& $(RUN) js node /src/examples/node/checkout.mjs > $$out/node.json \
		&& cat $$out/node.json \
		&& if cmp -s $$out/python.json $$out/node.json; then \
			printf '\nPython and Node.js produced exactly the same output.\n'; rm -rf $$out; \
		else diff $$out/python.json $$out/node.json; rm -rf $$out; exit 1; fi

nextjs: ## the Next.js example at http://localhost:3000 (Ctrl+C to stop)
	$(COMPOSE) up --build nextjs

##@ Maintenance

fixtures: ## rewrite the expected outputs in fixtures/ from the engine (review the diff)
	$(COMPOSE) build go
	$(RUN) -v "$(CURDIR)/fixtures:/src/fixtures" go go test ./cart -run 'Fixture|Scenarios' -update

clean: ## delete dist/, the containers and the images of the project
	rm -rf dist
	$(COMPOSE) down --remove-orphans
	@images=$$(docker image ls --filter 'reference=ecommerce-cart-core/*' --format '{{.Repository}}:{{.Tag}}'); \
		[ -z "$$images" ] || docker image rm $$images
