FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o gomad ./cmd/server

FROM alpine:3.19
ENV PORT=8080
EXPOSE $PORT

# Copy the binary from builder stage
COPY --from=builder /app/gomad /gomad

# Run the service
CMD ["/gomad"]
