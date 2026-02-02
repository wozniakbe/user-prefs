FROM golang:1.25.5-alpine3.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /server .

FROM gcr.io/distroless/static-debian12

COPY --from=builder /server /server

EXPOSE 8080

ENTRYPOINT ["/server"]
