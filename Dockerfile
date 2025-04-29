# Build
FROM golang:1.24-alpine as builder

WORKDIR /app

COPY ./go.mod								./go.mod
COPY ./go.sum								./go.sum
COPY ./internal								./internal
COPY ./cmd									./cmd

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/main /app/cmd/app/main.go

# Production image
FROM gcr.io/distroless/static

COPY --from=builder /app/build/main /main

CMD ["/main"]