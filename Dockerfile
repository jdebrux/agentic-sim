# syntax=docker/dockerfile:1.6

##
## Build
##
ARG GO_VERSION=1.25.3
FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /src

COPY go.mod ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /out/simserver ./cmd/simserver

##
## Runtime
##
FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=builder /out/simserver /app/simserver

# Default port (can be overridden by PORT env)
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/simserver"]
