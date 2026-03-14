FROM golang:1.25 AS builder

WORKDIR /src

RUN apt-get update && apt-get install -y git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags="-s -w" -o /out/lazypubk ./cmd/lazypubk

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags="-s -w" -o /out/lazy-dvc-auth ./cmd/lazy-dvc-auth

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags="-s -w" -o /out/restricted-shell ./cmd/restricted-shell

FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    openssh-server \
    curl \
    ca-certificates \
    fuse3 \
    rclone \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir /var/run/sshd

COPY --from=builder /out/lazypubk /usr/local/bin/lazypubk
COPY --from=builder /out/lazy-dvc-auth /usr/local/bin/lazy-dvc-auth
COPY --from=builder /out/restricted-shell /usr/local/bin/restricted-shell

RUN useradd -m -s /usr/local/bin/restricted-shell dvc-storage && \
    mkdir -p /home/dvc-storage/data && \
    chown -R dvc-storage:dvc-storage /home/dvc-storage

RUN sed -i 's/#AuthorizedKeysCommand/AuthorizedKeysCommand/' /etc/ssh/sshd_config && \
    sed -i 's/AuthorizedKeysCommand none/AuthorizedKeysCommand \/usr\/local\/bin\/lazy-dvc-auth %u/' /etc/ssh/sshd_config && \
    sed -i 's/AuthorizedKeysCommandUser nobody/AuthorizedKeysCommandUser root/' /etc/ssh/sshd_config && \
    sed -i 's/#AuthorizedKeysCommandUser/AuthorizedKeysCommandUser/' /etc/ssh/sshd_config && \
    sed -i 's/#PubkeyAuthentication/PubkeyAuthentication/' /etc/ssh/sshd_config && \
    sed -i 's/PubkeyAuthentication no/PubkeyAuthentication yes/' /etc/ssh/sshd_config && \
    sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin no/' /etc/ssh/sshd_config && \
    sed -i 's/#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config && \
    sed -i 's/^AcceptEnv.*/#AcceptEnv LDVC_GH_TOKEN LDVC_GH_ORG_NAME LDVC_GH_TEAM_NAME/' /etc/ssh/sshd_config

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 22

ENTRYPOINT ["/entrypoint.sh"]
