FROM golang:1.9

# Dependencies
RUN apt-get update && apt-get install --no-install-recommends -y build-essential \
    zlib1g-dev pkg-config

# Install libturbo-jpeg
ADD https://sourceforge.net/projects/libjpeg-turbo/files/1.5.2/libjpeg-turbo-official_1.5.2_amd64.deb/download /tmp/libjpeg-turbo-official_1.5.2_amd64.deb
RUN cd /tmp && dpkg -i /tmp/libjpeg-turbo-official_1.5.2_amd64.deb && \
    echo /opt/libjpeg-turbo/lib64 > /etc/ld.so.conf.d/libjpeg-turbo.conf && ldconfig

# Install libpng
ADD https://sourceforge.net/projects/libpng/files/libpng16/1.6.32/libpng-1.6.32.tar.gz/download /tmp/libpng-1.6.32.tar.gz
RUN cd /tmp && tar -zxvf libpng-1.6.32.tar.gz && cd libpng-1.6.32 && \
    ./configure --prefix=/usr && make && make install && ldconfig

# Install ImageMagick v7
ADD https://www.imagemagick.org/download/ImageMagick-7.0.6-10.tar.xz /tmp/
RUN cd /tmp && tar -xvf ImageMagick-7.0.6-10.tar.xz && cd ImageMagick-7.0.6-10 && \
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
