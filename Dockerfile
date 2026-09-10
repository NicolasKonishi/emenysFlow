FROM golang:1.26-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/emenys ./cmd/server

FROM alpine:3.22

RUN addgroup -S emenys && adduser -S -G emenys emenys \
    && mkdir -p /var/lib/emenys && chown emenys:emenys /var/lib/emenys
WORKDIR /app
COPY --from=build --chown=emenys:emenys /out/emenys /app/emenys
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo

USER emenys
EXPOSE 8080
ENTRYPOINT ["/app/emenys"]
CMD ["-address", ":8080", "-database", "/var/lib/emenys/buffetflow.db"]
