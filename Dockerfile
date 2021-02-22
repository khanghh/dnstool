FROM golang:1.14.2-alpine as builder

WORKDIR /app

RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates

COPY ./main.go .
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o /go/bin/app

FROM scratch
WORKDIR /app

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/app ./app

CMD ["./app"]
