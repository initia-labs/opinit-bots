# Used for running OPinit bots in a Docker container
# 
# Usage: 
#  $ docker build --tag opinit-bots .

FROM golang:1.23.7-alpine AS builder

RUN apk add --no-cache g++ make bash

WORKDIR /app
COPY . /app/

RUN make install

FROM alpine:latest

COPY --from=builder /go/bin/opinitd /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/opinitd"]
