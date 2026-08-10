FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/kagari-api ./cmd/api

FROM alpine:3.22
RUN apk add --no-cache wget ca-certificates
COPY --from=build /out/kagari-api /usr/local/bin/kagari-api
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/kagari-api"]
