FROM alpine:3.5

RUN \
  apk add --update go git make gcc musl-dev linux-headers ca-certificates && \
  git clone --depth 1 --branch release/1.7 https://github.com/bgmchain/go-bgmchain && \
  (cd go-bgmchain && make gbgm) && \
  cp go-bgmchain/build/bin/gbgm /gbgm && \
  apk del go git make gcc musl-dev linux-headers && \
  rm -rf /go-bgmchain && rm -rf /var/cache/apk/*

EXPOSE 7575
EXPOSE 17575

ENTRYPOINT ["/gbgm"]
