# ============= Compilation Stage ================
FROM golang:1.20-bullseye AS builder

WORKDIR /build
# Copy and download odyssey dependencies using go mod
COPY go.mod .
COPY go.sum .
RUN go mod download
# Copy the code into the container
COPY . .
# Build odysseygo
RUN ./scripts/build.sh

# ============= Cleanup Stage ================
FROM debian:11-slim
WORKDIR /

# Copy the executables into the container
COPY --from=builder /build/bin/odyssey .
RUN ls -la ./odyssey
ENTRYPOINT [ "./odyssey" ]
