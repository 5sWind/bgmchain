FROM ubuntu:xenial

RUN \
  apt-get update && apt-get upgrade -q -y && \
  apt-get install -y --no-install-recommends golang git make gcc libc-dev ca-certificates && \
  git clone --depth 1 https://github.com/bgmchain/bgmchain && \
  (cd bgmchain && make gbgm) && \
  cp bgmchain/build/bin/gbgm /gbgm && \
  apt-get remove -y golang git make gcc libc-dev && apt autoremove -y && apt-get clean && \
  rm -rf /bgmchain

EXPOSE 7575
EXPOSE 17575

ENTRYPOINT ["/gbgm"]
