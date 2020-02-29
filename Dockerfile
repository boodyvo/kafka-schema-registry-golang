FROM golang:1.13-alpine as builder

RUN apk add librdkafka-dev pkgconf gcc libc-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go install ./cmd/...

#FROM alpine:latest
RUN cp -r /go/bin /bin
RUN cp cmd/producer/schema.avsc /schema.avsc

CMD ["producer"]