# Use buildx for multi-platform builds
# Build stage
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder
LABEL org.opencontainers.image.source="https://github.com/interlynk-io/sbommv"

RUN apk add --no-cache make git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build for multiple architectures
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -a -o sbommv .

RUN chmod +x sbommv

# Final stage
FROM alpine:3.19
LABEL org.opencontainers.image.source="https://github.com/interlynk-io/sbommv"
LABEL org.opencontainers.image.description="Transfer SBOM's between different systems."
LABEL org.opencontainers.image.licenses=Apache-2.0

COPY --from=builder /app/sbommv /app/sbommv

ENTRYPOINT ["/app/sbommv"]
