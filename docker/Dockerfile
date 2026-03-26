FROM golang:1.21-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN cd cmd/node && go build -o neocoin .

FROM alpine:latest

RUN apk add --no-cache ca-certificates curl

WORKDIR /app

COPY --from=builder /build/cmd/node/neocoin .

EXPOSE 8080 9090

CMD ["./neocoin"]