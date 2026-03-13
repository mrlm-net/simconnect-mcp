FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-X main.version=${VERSION} -s -w" \
    -o /simconnect-mcp ./cmd/simconnect-mcp/

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /simconnect-mcp /simconnect-mcp
ENV MCP_MODE=docs
ENV GIN_MODE=release
EXPOSE 8080
ENTRYPOINT ["/simconnect-mcp"]
