# syntax=docker/dockerfile:1.7

FROM golang:1.26.5-alpine3.23 AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

ENV GOTOOLCHAIN=local

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" \
    go build \
      -buildvcs=false \
      -trimpath \
      -ldflags="-s -w" \
      -o /out/api \
      ./cmd/api

FROM scratch AS runtime

ARG VERSION=dev
ARG REVISION=unknown

LABEL org.opencontainers.image.title="Tauco Cap Badak API" \
      org.opencontainers.image.description="Public API for Tauco Cap Badak Phase 1B" \
      org.opencontainers.image.source="https://github.com/ilhamnugraha8944/tauco" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${REVISION}"

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build --chown=65532:65532 /out/api /api

USER 65532:65532

ENV APP_ENV=production \
    SERVICE_NAME=tauco-api \
    HTTP_HOST=0.0.0.0 \
    PORT=8080 \
    LOG_FORMAT=json

EXPOSE 8080

ENTRYPOINT ["/api"]
