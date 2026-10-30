# syntax=docker/dockerfile:1

# The cart rules, in Go. Build context: the repository root (the tests read fixtures/).
#
#   test   gofmt, go vet and go test on GO_VERSION
#   wasm   calc.wasm compiled with TinyGo, which both packages embed

ARG GO_VERSION=1.27
ARG TINYGO_VERSION=0.43.0

FROM golang:${GO_VERSION} AS test
# Use the image's Go. The toolchain line in go.mod is for machines whose Go is
# older than TinyGo requires.
ENV GOTOOLCHAIN=local
WORKDIR /src/core
COPY fixtures/ /src/fixtures/
COPY core/ ./
CMD ["sh", "-c", "files=$(gofmt -l .) && [ -z \"$files\" ] || { echo \"gofmt needed: $files\"; exit 1; }; go vet ./... && go test ./..."]

FROM tinygo/tinygo:${TINYGO_VERSION} AS build
# Go 1.27 builds encoding/json on its v2 implementation, which doubles
# calc.wasm (701 KB instead of 331 KB). The engine behaves the same on both,
# as the fixture tests show, so keep the smaller one.
ENV GOTOOLCHAIN=local GOEXPERIMENT=nojsonv2
WORKDIR /home/tinygo/core
COPY core/ ./
# wasm-unknown makes a module with no imports, so the same binary runs under
# wasmtime, Node.js and browsers. Its default GC never frees memory, which
# would make a long-lived instance grow forever: -gc=conservative fixes that.
RUN tinygo build -target=wasm-unknown -gc=conservative -panic=trap -opt=2 -no-debug \
    -o /tmp/calc.wasm ./cmd/wasm

FROM scratch AS wasm
COPY --from=build /tmp/calc.wasm /calc.wasm
