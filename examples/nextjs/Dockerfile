# syntax=docker/dockerfile:1

# The Next.js example, built for production. Build context: the repository
# root. Named context "js": the image of the npm package (packages/js/Dockerfile),
# whose /src/packages/js holds the packed package.

ARG NODE_VERSION=24

FROM node:${NODE_VERSION}-slim AS build
ENV NEXT_TELEMETRY_DISABLED=1
WORKDIR /src
COPY --from=js /src/packages/js/ packages/js/
COPY fixtures/ fixtures/
WORKDIR /src/examples/nextjs
COPY examples/nextjs/package.json examples/nextjs/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY examples/nextjs/ ./
RUN npm run build

FROM node:${NODE_VERSION}-slim
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1
COPY --from=build /src/ /src/
WORKDIR /src/examples/nextjs
EXPOSE 3000
CMD ["node", "node_modules/next/dist/bin/next", "start", "-H", "0.0.0.0", "-p", "3000"]
