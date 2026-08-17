FROM --platform=$BUILDPLATFORM golang:1.26.6@sha256:640a234f4bea3e399c056b7b8f9c667c4939befae8db2f14e9785e16eccd4205 AS builder

WORKDIR /src

ARG TARGETOS
ARG TARGETARCH

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -o /out/egueducation-api ./cmd/server \
    && CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -o /out/egueducation-pdf-validator ./cmd/archive-pdf-validator

FROM gcr.io/distroless/base-debian12@sha256:76b3162a31477bca4a245b836c624f4c4a1a3705e99b9003907d992bec2c4bca

WORKDIR /app

COPY --from=builder /out/egueducation-api /app/egueducation-api
COPY --from=builder /out/egueducation-pdf-validator /app/egueducation-pdf-validator

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/egueducation-api"]
