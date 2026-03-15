FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags="-s -w" -o /out/lazypubk ./cmd/lazypubk

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags="-s -w" -o /out/authpubk ./cmd/authpubk

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags="-s -w" -o /out/noshell ./cmd/noshell

FROM alpine:3.23

RUN apk add --no-cache \
    openssh \
    curl \
    ca-certificates \
    fuse3 \
    rclone \
    netcat-openbsd \
    && mkdir /var/run/sshd

COPY --from=builder /out/lazypubk /usr/local/bin/lazypubk
COPY --from=builder /out/authpubk /usr/local/bin/authpubk
COPY --from=builder /out/noshell /usr/local/bin/noshell

RUN adduser -D -s /usr/local/bin/noshell dvc-storage && \
    passwd -d dvc-storage && \
    mkdir -p /home/dvc-storage/data && \
    chown -R root:root /home/dvc-storage && \
    chmod -R 755 /home/dvc-storage && \
    chown -R dvc-storage:dvc-storage /home/dvc-storage/data && \
    chmod -R 777 /home/dvc-storage/data && \
    mkdir -p /var/cache/lazy-dvc && \
    chown -R dvc-storage:dvc-storage /var/cache/lazy-dvc && \
    chmod 755 /var/cache/lazy-dvc

COPY sshd_config /etc/ssh/sshd_config

# Generate SSH host keys
RUN ssh-keygen -A

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD nc -z localhost 22 && mountpoint -q /home/dvc-storage/data || exit 1

EXPOSE 22

ENTRYPOINT ["/entrypoint.sh"]