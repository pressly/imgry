FROM golang:1.9-stretch as builder

ENV IMAGEMAGICK_VERSION 7.0.7-14

RUN apt-get update && apt-get install --no-install-recommends -y \
        build-essential \
        libpng-dev \
        libjpeg-dev \
        libwebp-dev \
        libexif-dev \
        liblzma-dev \
        libtiff-dev \
        libopenjp2-7-dev \
        liblcms2-dev \
        libxml2-dev \
        zlib1g-dev \
        ca-certificates \
        gpg \
        pkg-config \
        wget

ENV GOSU_VERSION 1.10
RUN set -ex; \
	\
	fetchDeps='ca-certificates wget'; \
	apt-get update; \
	apt-get install -y --no-install-recommends $fetchDeps; \
	rm -rf /var/lib/apt/lists/*; \
	\
	dpkgArch="$(dpkg --print-architecture | awk -F- '{ print $NF }')"; \
	wget -O /usr/local/bin/gosu "https://github.com/tianon/gosu/releases/download/$GOSU_VERSION/gosu-$dpkgArch"; \
	wget -O /usr/local/bin/gosu.asc "https://github.com/tianon/gosu/releases/download/$GOSU_VERSION/gosu-$dpkgArch.asc"; \
	export GNUPGHOME="$(mktemp -d)"; \
	gpg --keyserver ha.pool.sks-keyservers.net --recv-keys B42F6819007F00F88E364FD4036A9C25BF357DD4; \
	gpg --batch --verify /usr/local/bin/gosu.asc /usr/local/bin/gosu; \
	chmod +x /usr/local/bin/gosu; \
	gosu nobody true


RUN mkdir /build && \
        cd /build && \
        wget https://github.com/ImageMagick/ImageMagick/archive/${IMAGEMAGICK_VERSION}.tar.gz && \
        tar zxf ${IMAGEMAGICK_VERSION}.tar.gz && \
        cd ImageMagick-${IMAGEMAGICK_VERSION} && \
        ./configure \
                --prefix=/usr \
                --enable-shared \
                --disable-openmp \
                --disable-hdri \
                --disable-largefile \
                --disable-static \
                --with-bzlib \
                --with-jpeg \
                --with-jp2 \
                --with-lcms \
                --with-png \
                --with-tiff \
                --with-webp \
                --with-xml \
                --with-zlib \
                --with-quantum-depth=8 \
                --without-dot \
                --without-dps \
                --without-fpx \
                --without-freetype \
                --without-gslib \
                --without-magick-plus-plus \
                --without-perl \
                --without-wmf \
                --without-x
RUN cd /build/ImageMagick-${IMAGEMAGICK_VERSION} && \
        make && \
        make install

ADD . /go/src/github.com/pressly/imgry
WORKDIR /go/src/github.com/pressly/imgry
RUN make dist


FROM debian:stretch-slim

RUN apt-get update && apt-get install --no-install-recommends -y \
                ca-certificates \
                libpng16-16 \
                libjpeg62-turbo \
                libwebp6 \
                libexif12 \
                lzma \
                libtiff5 \
                libopenjp2-7 \
                liblcms2-2 \
                libxml2 \
                zlib1g && \
        rm -rf /var/lib/apt/lists/*

COPY --from=builder /usr/etc/ImageMagick-7 /usr/etc/ImageMagick-7
COPY --from=builder /usr/include/ImageMagick-7 /usr/include/ImageMagick-7
COPY --from=builder /usr/lib/* /usr/lib/

RUN ldconfig

COPY --from=builder /usr/local/bin/gosu /usr/local/bin/gosu

COPY --from=builder /go/src/github.com/pressly/imgry/bin/imgry-server /bin/imgry-server
COPY --from=builder /go/src/github.com/pressly/imgry/scripts/docker-entrypoint.sh /usr/local/bin/

ENTRYPOINT ["docker-entrypoint.sh"]

EXPOSE 4446

CMD ["/bin/imgry-server", "-config=/imgry.conf"]
