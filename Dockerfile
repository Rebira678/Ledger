# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ledger-server ./cmd/ledger-server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/ledger-server /ledger-server
COPY --from=build /src/web/templates /web/templates
COPY --from=build /src/migrations /migrations
ENV LEDGER_TEMPLATES_DIR=/web/templates
ENTRYPOINT ["/ledger-server"]
