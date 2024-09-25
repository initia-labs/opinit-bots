# docker build --tag opinit-bots .
# docker run --name opinit-bots-container -e BOT_NAME="executor" opinit-bots
# for debug: docker run -it --name debug-container --entrypoint /bin/sh opinit-bots
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache g++ make bash

WORKDIR /app

COPY . /app/

RUN make install

FROM alpine:latest

COPY --from=builder /go/bin/opinitd /usr/local/bin/

COPY --chmod=755 ./entrypoint.sh /usr/local/bin

ENTRYPOINT ["entrypoint.sh"]
