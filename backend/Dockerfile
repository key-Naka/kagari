FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/kagari-api ./cmd/api && go build -o /out/kagari-host-metrics ./cmd/hostmetrics

FROM alpine:3.22 AS api
RUN apk add --no-cache wget ca-certificates && addgroup -S kagari && adduser -S -G kagari kagari
COPY --from=build /out/kagari-api /usr/local/bin/kagari-api
USER kagari
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/kagari-api"]

FROM alpine:3.22 AS host-metrics
COPY --from=build /out/kagari-host-metrics /usr/local/bin/kagari-host-metrics
USER 65532:65532
EXPOSE 8090
ENTRYPOINT ["/usr/local/bin/kagari-host-metrics"]
