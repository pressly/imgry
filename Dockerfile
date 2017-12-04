FROM golang:1.9-alpine3.6 as builder

RUN apk add --no-cache \
        libpng \
        libjpeg-turbo \
        imagemagick \
        make \
        gcc \
        musl-dev \
        libpng-dev \
        libjpeg-turbo-dev \
        imagemagick-dev

ADD . /go/src/github.com/pressly/imgry
WORKDIR /go/src/github.com/pressly/imgry
RUN make dist


FROM alpine:3.6

RUN apk add --no-cache \
        libpng \
        libjpeg-turbo \
        imagemagick \
        ca-certificates

COPY --from=builder /go/src/github.com/pressly/imgry/bin/imgry-server /bin/imgry-server

EXPOSE 4446

CMD ["/bin/imgry-server", "-config=/imgry.conf"]
