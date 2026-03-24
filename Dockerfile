# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Cache dependency downloads (copy only go.mod; go.sum will be generated fresh).
COPY go.mod ./
RUN go mod download

# Copy source and regenerate go.sum from the module cache.
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /server .

EXPOSE 8080

CMD ["./server"]
