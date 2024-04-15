FROM golang:1.22-alpine as base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY auth ./auth

RUN CGO_ENABLED=0 GOOS=linux go build -o /smtp2slack

CMD ["/smtp2slack"]

FROM gcr.io/distroless/static-debian11

COPY --from=base /smtp2slack .

CMD ["./main"]
