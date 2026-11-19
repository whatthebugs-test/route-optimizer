FROM golang:1.24.4-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/optimizer ./cmd/optimizer

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/optimizer /optimizer
ENTRYPOINT ["/optimizer"]
