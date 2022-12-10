FROM golang:alpine AS builder

WORKDIR /build

COPY . .
RUN go build -ldflags="-s -w" -o /app/main main.go

RUN apk update && apk add openssh && apk add git

WORKDIR /app

FROM alpine

COPY --from=builder /app/main /app/main

CMD ["./main"]
