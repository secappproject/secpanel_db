# File: Dockerfile

# Tahap 1: Build aplication
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /main .

# Tahap 2: Buat final image yang super kecil
FROM alpine:latest
WORKDIR /
COPY --from=builder /main /main
EXPOSE 8080
CMD ["/main"]