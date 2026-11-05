FROM golang:1.26.8-bookworm AS builder
RUN apt-get update && apt-get install -y \
    libvirt-dev \
    genisoimage \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o /app/my-vm-manager .

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    libvirt0 \
    libvirt-daemon-system \
    qemu-system-x86 \
    qemu-utils \
    genisoimage \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m -s /bin/bash appuser

WORKDIR /app

RUN mkdir /app/tmp/

RUN touch /app/vms

COPY --from=builder /app/my-vm-manager .

EXPOSE 8080


CMD ["./my-vm-manager"]
