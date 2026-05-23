FROM golang:1.26-alpine AS builder
WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build \
    -tags timetzdata \
    -ldflags="-s -w" \
    -trimpath \
    -o server .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/server /server

# Re-declare and set the TZ environment variable in the FINAL stage
ARG TZ=Europe/Stockholm
ENV TZ=${TZ}

USER 1000:1000

EXPOSE 8000

ENTRYPOINT ["/server"]
