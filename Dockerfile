# syntax=docker/dockerfile:1
FROM node:20-alpine AS frontend-build
WORKDIR /app
COPY web-client/package*.json ./
RUN npm install
COPY web-client/ ./
RUN npm run build

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ledger-server ./cmd/ledger-server

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /out/ledger-server /ledger-server
COPY --from=frontend-build /app/dist /web-client/dist
COPY --from=build /src/migrations /migrations
ENTRYPOINT ["/ledger-server"]
