FROM golang:1.22-alpine as builder
ARG VERSION="development"

WORKDIR /run

COPY ./ ./
RUN go mod download
RUN go build -o /run ./cmd/daemon.go

FROM golang:1.22-alpine
COPY --from=builder /run/daemon /run/
WORKDIR /run

CMD ["/run/daemon"]