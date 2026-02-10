FROM golang:1.25-alpine3.23 AS builder
RUN apk update && apk upgrade && \
      apk add go git
WORKDIR /sms-webhook
COPY go.sum go.mod /sms-webhook/
RUN go mod download
COPY *.go /sms-webhook/
ENV CGO_ENABLED=0
RUN go build

FROM alpine:3.23 AS certbuilder
RUN apk add openssl
WORKDIR /certs
RUN openssl req -nodes -new -x509 -subj="/C=US/ST=Denial/L=springfield/O=Dis/CN=ircwebhook.terminaldweller.com" -keyout server.key -out server.cert

FROM alpine:3.23
COPY --from=certbuilder /certs /certs
COPY --from=builder /sms-webhook/sms-webhook /sms-webhook/
ENTRYPOINT ["/sms-webhook/sms-webhook"]
