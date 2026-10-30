# syntax=docker/dockerfile:1

# The npm package. Build context: the repository root. Named context "wasm":
# the wasm stage of core/Dockerfile (docker-compose.yml wires it).
#
# The tarball ends up in /dist, and the tests run against the packed files,
# the way users get the package.

ARG NODE_VERSION=24

FROM node:${NODE_VERSION}-slim AS build
WORKDIR /src/packages/js
COPY packages/js/ ./
COPY --from=wasm /calc.wasm calc.wasm
RUN node scripts/embed-wasm.mjs calc.wasm src/calc-wasm.js \
    && mkdir /dist && npm pack --pack-destination /dist

FROM node:${NODE_VERSION}-slim
COPY --from=build /dist/ /dist/
WORKDIR /src/packages/js
RUN tar -xzf /dist/*.tgz --strip-components=1
# /src mirrors the repository: the tests, the shared fixtures, the example, and
# dist/calc.wasm, the engine the package must carry.
COPY packages/js/test/ test/
COPY fixtures/ /src/fixtures/
COPY --from=wasm /calc.wasm /src/dist/calc.wasm
COPY examples/node/ /src/examples/node/
RUN cd /src/examples/node && npm install --no-package-lock --no-audit --no-fund --loglevel=error
CMD ["node", "--test"]
