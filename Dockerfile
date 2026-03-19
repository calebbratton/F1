# ---- builder ----
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /f1-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /f1-seed ./cmd/seed

# ---- runtime ----
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /f1-api  ./f1-api
COPY --from=builder /f1-seed ./f1-seed
EXPOSE 8080
CMD ["./f1-api"]
