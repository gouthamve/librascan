# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o librascan ./cmd/librascan

# Final stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/librascan .
EXPOSE 8080
CMD ["./librascan", "serve"]
