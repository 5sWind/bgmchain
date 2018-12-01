FROM alpine:3.5

RUN \
  apk add --update go git make gcc musl-dev linux-headers ca-certificates && \
  git clone --depth 1 https://github.com/bgmchain/bgmchain && \
  (cd bgmchain && make gbgm) && \
  cp bgmchain/build/bin/gbgm /gbgm && \
  apk del go git make gcc musl-dev linux-headers && \
  rm -rf /bgmchain && rm -rf /var/cache/apk/*

EXPOSE 7575
EXPOSE 17575

ENTRYPOINT ["/gbgm"]
