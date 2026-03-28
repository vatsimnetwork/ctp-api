FROM golang:1.26-alpine AS builder
WORKDIR /app
RUN go install github.com/swaggo/swag/cmd/swag@latest
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN swag init
RUN CGO_ENABLED=0 go build -o ctp-api .

FROM gcr.io/distroless/base-debian12
COPY --from=builder /app/ctp-api /ctp-api
CMD ["/ctp-api"]
