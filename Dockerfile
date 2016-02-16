FROM golang:1.6rc1

# Dependencies
RUN apt-get update && apt-get install --no-install-recommends -y build-essential \
    zlib1g-dev pkg-config

# Install libturbo-jpeg 1.4.2
RUN curl --silent -L https://sourceforge.net/projects/libjpeg-turbo/files/1.4.2/libjpeg-turbo-official_1.4.2_amd64.deb/download -o /tmp/libjpeg-turbo-official_1.4.2_amd64.deb
RUN cd /tmp && dpkg -i /tmp/libjpeg-turbo-official_1.4.2_amd64.deb && \
    echo /opt/libjpeg-turbo/lib64 > /etc/ld.so.conf.d/libjpeg-turbo.conf && ldconfig

# Install libpng 1.6.21
RUN curl --silent -L https://downloads.sourceforge.net/project/libpng/libpng16/1.6.21/libpng-1.6.21.tar.gz | tar -xvzf - -C /tmp
RUN cd /tmp/libpng-* && \
    ./configure --prefix=/usr && make && make install && ldconfig

RUN curl --silent -L https://www.imagemagick.org/download/ImageMagick-6.9.3-4.tar.xz | tar -xvJf - -C /tmp
RUN cd /tmp/ImageMagick-* && \
    ./configure --prefix=/usr \
                --enable-shared \
                --disable-openmp \
                --disable-opencl \
                --without-x \
                --with-quantum-depth=8 \
                --with-magick-plus-plus=no \
                --with-jpeg=yes \
                --with-png=yes \
                --with-jp2=yes \
                LIBS="-ljpeg -lturbojpeg" \
                LDFLAGS="-L/opt/libjpeg-turbo/lib64" \
                CFLAGS="-I/opt/libjpeg-turbo/include" \
                CPPFLAGS="-I/opt/libjpeg-turbo/include" \
    && make && make install && ldconfig

# Imgry
ADD . /go/src/github.com/pressly/imgry
WORKDIR /go/src/github.com/pressly/imgry
RUN make dist
RUN mv bin/imgry-server /bin/imgry-server

EXPOSE 4446

CMD ["/bin/imgry-server", "-config=/etc/imgry.conf"]
