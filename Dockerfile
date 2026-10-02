FROM alpine:latest AS libredwg

RUN apk add --no-cache \
    git \
    libtool \
    make \
    autoconf \
    automake \
    gcc \
    g++ \
    build-base \
    texinfo

RUN git clone --recurse-submodules --branch 0.14 \
    https://github.com/LibreDWG/libredwg.git \
    /libredwg

WORKDIR /libredwg

RUN ./autogen.sh
RUN ./configure

ENV CCACHE_DIR=/cache/ccache

# Make sure config.h actually exists after configure
RUN find /libredwg -name config.h -print

RUN --mount=type=cache,target=/cache/ccache \
    make -j$(nproc)

RUN make install

FROM alpine:latest AS golibredwg

WORKDIR /golibredwg

COPY --from=libredwg /libredwg/include ./include
COPY --from=libredwg /libredwg/bindings/dwg.i ./dwg.i

RUN apk add --no-cache swig

RUN swig \
    -go \
    -cgo \
    -package golibredwg \
    -Iinclude \
    dwg.i


FROM golang:1-alpine AS golang-builder

RUN apk add --no-cache \
    gcc \
    musl-dev \
    g++

COPY --from=golibredwg /golibredwg/*.go /libredwg/golibredwg/
COPY --from=golibredwg /golibredwg/dwg_wrap.c /libredwg/golibredwg/
COPY --from=libredwg /libredwg/include /libredwg/include
COPY --from=libredwg /libredwg/src /libredwg/src
RUN --mount=type=bind,from=libredwg,source=/,target=/libredwg-root cp /libredwg-root/usr/local/lib/libredwg* /usr/local/lib/
RUN ln -s /usr/local/lib/libredwg.so /usr/local/lib/libdwg.so

WORKDIR /libredwg/golibredwg
RUN go mod init github.com/ski7777/golibredwg
RUN go mod tidy

COPY ./src /src

WORKDIR /src
ENV CGO_ENABLED=1
ENV CCACHE_DIR=/cache/ccache
ENV CGO_CFLAGS="-I/libredwg/include -I/libredwg/src -I/usr/local/include -Wno-stringop-overflow"
ENV CGO_LDFLAGS="-L/usr/local/lib -lredwg"
RUN \
    --mount=type=cache,target=/cache/ccache \
    --mount=type=cache,target=/root/.cache/go-build \
    go build \
    -ldflags="-s -w" \
    -o /bin/csd-sll-mapper \
    ./cmd


FROM alpine:latest

RUN --mount=type=bind,from=libredwg,source=/,target=/libredwg-root cp /libredwg-root/usr/local/lib/libredwg* /usr/local/lib/
COPY --from=golang-builder /bin/csd-sll-mapper /bin/csd-sll-mapper
CMD ["/bin/csd-sll-mapper"]