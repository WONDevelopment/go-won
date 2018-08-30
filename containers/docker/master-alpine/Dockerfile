FROM alpine:3.7

RUN \
  apk add --update go git make gcc musl-dev linux-headers ca-certificates && \
  git clone --depth 1 --branch release/1.8 https://github.com/WONDevelopment/go-won && \
  (cd go-gwon && make gwon) && \
  cp go-gwon/build/bin/gwon /gwon && \
  apk del go git make gcc musl-dev linux-headers && \
  rm -rf /go-gwon && rm -rf /var/cache/apk/*

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/gwon"]
