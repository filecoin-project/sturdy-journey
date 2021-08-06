FROM golang:buster as builder

RUN apt-get update && apt-get install -y ca-certificates

ENV GO111MODULE=on \
    CGO_ENABLED=0  \
    GOOS=linux     \
    GOARCH=amd64

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build -o sturdy-journey ./cmd/sturdy-journey

FROM debian:buster

COPY --from=builder /etc/ssl/certs        /etc/ssl/certs
COPY --from=builder /build/sturdy-journey /usr/local/bin

EXPOSE 5100
EXPOSE 5101

ENTRYPOINT ["/usr/local/bin/sturdy-journey"]
CMD ["-help"]
