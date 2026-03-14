FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags="-s -w" -o /out/lazydvc ./cmd/lazydvc

FROM alpine:3.21

RUN apk add --no-cache ca-certificates
RUN adduser -D -H -u 10001 appuser

USER appuser

COPY --from=builder /out/lazydvc /usr/local/bin/lazydvc

ENTRYPOINT ["lazydvc"]
