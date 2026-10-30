# syntax=docker/dockerfile:1

# The Python package. Build context: the repository root. Named context "wasm":
# the wasm stage of core/Dockerfile (docker-compose.yml wires it).
#
# The wheel and sdist end up in /dist, and the tests run against the installed
# wheel, the way users get the package.

ARG PYTHON_VERSION=3.13

FROM python:${PYTHON_VERSION}-slim AS build
ENV PIP_DISABLE_PIP_VERSION_CHECK=1 PIP_ROOT_USER_ACTION=ignore
WORKDIR /src/packages/python
COPY packages/python/ ./
COPY --from=wasm /calc.wasm src/ecommerce_cart_core/calc.wasm
RUN pip install --no-cache-dir build && python -m build --outdir /dist

FROM python:${PYTHON_VERSION}-slim
ENV PIP_DISABLE_PIP_VERSION_CHECK=1 PIP_ROOT_USER_ACTION=ignore PYTHONDONTWRITEBYTECODE=1
COPY --from=build /dist/ /dist/
RUN pip install --no-cache-dir /dist/*.whl "pytest>=8"
# /src mirrors the repository: the tests, the shared fixtures, the example, and
# dist/calc.wasm, the engine the package must carry.
WORKDIR /src/packages/python
COPY packages/python/pyproject.toml ./
COPY packages/python/tests/ tests/
COPY fixtures/ /src/fixtures/
COPY examples/python/ /src/examples/python/
COPY --from=wasm /calc.wasm /src/dist/calc.wasm
CMD ["python", "-m", "pytest"]
