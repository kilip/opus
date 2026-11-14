# syntax=docker/dockerfile:1
FROM oven/bun:1-alpine AS ui-builder
WORKDIR /app/ui
COPY ui/package.json ui/bun.lock ./
RUN bun install --frozen-lockfile
COPY ui/ ./
RUN bun run build

FROM golang:1.27-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui-builder /app/ui/build/client ./ui/build/client
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /opus ./cmd/opus

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=go-builder /opus /opus
EXPOSE 7880
USER nonroot:nonroot
ENTRYPOINT ["/opus", "serve"]
