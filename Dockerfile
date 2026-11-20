FROM golang:1.25.1-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/optimizer ./cmd/optimizer

FROM alpine:3.20.3
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/optimizer /optimizer
ENTRYPOINT ["/optimizer"]
