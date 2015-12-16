FROM golang:1.5.1

# Dependencies
RUN apt-get update && apt-get install --no-install-recommends -y build-essential \
    zlib1g-dev pkg-config

# Install libturbo-jpeg 1.4.2
RUN wget -q https://sourceforge.net/projects/libjpeg-turbo/files/1.4.2/libjpeg-turbo-official_1.4.2_amd64.deb/download -O /tmp/libjpeg-turbo-official_1.4.2_amd64.deb && \
  cd /tmp && dpkg -i /tmp/libjpeg-turbo-official_1.4.2_amd64.deb && \
  echo /opt/libjpeg-turbo/lib64 > /etc/ld.so.conf.d/libjpeg-turbo.conf && ldconfig

# Install libpng 1.6.19
RUN wget -q https://downloads.sourceforge.net/project/libpng/libpng16/1.6.19/libpng-1.6.19.tar.gz -O /tmp/libpng-1.6.19.tar.gz && \
  cd /tmp && tar -zxvf libpng-1.6.19.tar.gz && cd libpng-1.6.19 && \
    ./configure --prefix=/usr && make && make install && ldconfig

RUN wget -q http://www.imagemagick.org/download/ImageMagick-6.9.2-8.tar.xz -O /tmp/ImageMagick-6.9.2-8.tar.xz && \
  cd /tmp && tar -xvf ImageMagick-6.9.2-8.tar.xz && cd ImageMagick-6.9.2-8 && \
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

# Install ffmpeg
RUN echo "deb http://www.deb-multimedia.org jessie main non-free" >> /etc/apt/sources.list && \
  echo "deb-src http://www.deb-multimedia.org jessie main non-free" >> /etc/apt/sources.list && \
  wget -q https://www.deb-multimedia.org/pool/main/d/deb-multimedia-keyring/deb-multimedia-keyring_2015.6.1_all.deb -O /tmp/deb-multimedia-keyring_2015.6.1_all.deb && \
  dpkg -i /tmp/deb-multimedia-keyring_2015.6.1_all.deb && \
  apt-get update && \
  apt-get install -y ffmpeg

# Imgry
ADD . /go/src/github.com/pressly/imgry
WORKDIR /go/src/github.com/pressly/imgry
RUN make dist
RUN mv bin/imgry-server /bin/imgry-server

EXPOSE 4446

CMD ["/bin/imgry-server", "-config=/etc/imgry.conf"]
