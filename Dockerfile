# Stage 1: Build (release)
FROM golang:1.24 AS builder

RUN apt-get update && apt-get install -y --no-install-recommends make coreutils ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY . .

ENV GOOS=linux
ENV GOARCH=amd64

RUN make deps

# Stage 2a: Build production binary
FROM builder as production_build

RUN make build

# Stage 2b: Build production Image (minimal)
FROM scratch AS production
COPY --from=production_build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=production_build /src/bin/opsicle_linux_amd64 /opsicle
ENTRYPOINT ["/opsicle"]

# Stage 3a: Build debug binary
FROM builder as debug_build

RUN make build_debug

# Stage 3b: Build debug Image
FROM debian:bookworm-slim AS debug

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    netcat \
    iproute2 \
    iputils-ping \
    tcpdump \
    dnsutils \
    telnet \
    && rm -rf /var/lib/apt/lists/*

COPY --from=debug_build /src/bin/opsicle_linux_amd64_debug /opsicle
ENTRYPOINT ["/opsicle"]
