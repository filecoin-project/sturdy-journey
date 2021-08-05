FROM golang:buster as builder

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

EXPOSE 5100
EXPOSE 5101

COPY --from=builder /build/sturdy-journey /usr/local/bin

ENTRYPOINT ["/usr/local/bin/sturdy-journey"]
CMD ["-help"]
