# Install default frontend deps with bun (exact versions via bun.lock)
FROM oven/bun:1 AS deps-default
WORKDIR /build
COPY web/default/package.json .
COPY web/default/bun.lock .
RUN bun install --frozen-lockfile

# Build default frontend with node (avoids bun baseline bin-shim bug)
FROM node:20-alpine AS builder
WORKDIR /build
COPY --from=deps-default /build/node_modules ./node_modules
COPY ./web/default .
COPY ./VERSION .
RUN DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) node_modules/.bin/rsbuild build

# Install classic frontend deps with bun (exact versions via bun.lock)
FROM oven/bun:1 AS deps-classic
WORKDIR /build
COPY web/classic/package.json .
COPY web/classic/bun.lock .
RUN bun install --frozen-lockfile

# Build classic frontend with node (avoids bun baseline bin-shim bug)
FROM node:20-alpine AS builder-classic
WORKDIR /build
COPY --from=deps-classic /build/node_modules ./node_modules
COPY ./web/classic .
COPY ./VERSION .
RUN VITE_REACT_APP_VERSION=$(cat VERSION) node_modules/.bin/vite build

FROM golang:1.26.1-alpine@sha256:2389ebfa5b7f43eeafbd6be0c3700cc46690ef842ad962f6c5bd6be49ed82039 AS builder2
ENV GO111MODULE=on CGO_ENABLED=0

ARG TARGETOS
ARG TARGETARCH
ENV GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64}
ENV GOEXPERIMENT=greenteagc

WORKDIR /build

ADD go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=builder /build/dist ./web/default/dist
COPY --from=builder-classic /build/dist ./web/classic/dist
RUN go build -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$(cat VERSION)'" -o new-api

FROM debian:bookworm-slim@sha256:f06537653ac770703bc45b4b113475bd402f451e85223f0f2837acbf89ab020a

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata libasan8 wget \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

COPY --from=builder2 /build/new-api /
EXPOSE 3000
WORKDIR /data
ENTRYPOINT ["/new-api"]
