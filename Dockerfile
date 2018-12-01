# Build Gbgm in a stock Go builder container
FROM golang:1.9-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers

ADD . /bgmchain
RUN cd /bgmchain && make gbgm

# Pull Gbgm into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /bgmchain/build/bin/gbgm /usr/local/bin/

EXPOSE 7575 8546 17575 17575/udp 30304/udp
ENTRYPOINT ["gbgm"]
