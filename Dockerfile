FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/cartero ./cmd/cartero

FROM gcr.io/distroless/static-debian12

COPY --from=build /out/cartero /usr/local/bin/cartero
ENTRYPOINT ["/usr/local/bin/cartero"]
CMD ["--help"]
