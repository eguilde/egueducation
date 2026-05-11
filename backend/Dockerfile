FROM golang:1.26 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/egueducation-api ./cmd/server

FROM gcr.io/distroless/base-debian12

WORKDIR /app

COPY --from=builder /out/egueducation-api /app/egueducation-api

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/egueducation-api"]
