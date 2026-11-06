# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS build
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/sentinel ./cmd/server

# AWS RDS CA bundle, used when the chart sets externalSecrets.sslMode=verify-full
FROM alpine:3 AS certs
RUN apk add --no-cache curl \
 && curl -fsSL -o /rds-ca.pem https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/sentinel /sentinel
COPY --from=certs /rds-ca.pem /certs/rds-ca.pem
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/sentinel"]
