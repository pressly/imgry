Imgry
=====

Imgry is an on-demand image delivery web service for responsive applications.

## Usage

First install Go 1.7+ and copy the etc/imgry.conf.sample (the default is fine), then..

```zsh
cd imgry/
make tools
make deps
make build
./bin/imgry-server -config=etc/imgry.conf
```

Open browser to:

`http://localhost:4446/mybucket?url=http://i.imgur.com/vEZy2Oh.jpg`

this will download the image from the source, cache it, persist it,
and return the sized image (in this case, with zero sizing) to the client.

**Now, some other variations:**

*Scale to 300x*

`http://localhost:4446/mybucket?url=http://i.imgur.com/vEZy2Oh.jpg&size=300x`

*Resize to exactly 300x300*

`http://localhost:4446/mybucket?url=http://i.imgur.com/vEZy2Oh.jpg&size=300x300`

*Resize to 300x300 and maintain aspect ratio*

`http://localhost:4446/mybucket?url=http://i.imgur.com/vEZy2Oh.jpg&size=300x300&op=cover`

*Same as above with a cropbox at points (x1:10%,y1:10%) to (x2:90%,y2:90%)*

`http://localhost:4446/mybucket?url=http://i.imgur.com/vEZy2Oh.jpg&size=300x300&op=cover&cb=0.1,0.1,0.9,0.9`

## Webapp usage

```html
<img src="http://localhost:4446/mybucket?url=http%3A%2F%2Fi.imgur.com%2FvEZy2Oh.jpg&size=300x300&op=cover" />
```

## Caching and persistence

Imgry is built with some clever caching such as:
* A layered cache store that stores/loads data from memory > on-disk (boltdb) > s3
* Once an image has been downloaded once, every other sizing operation will be loaded
from the chainstore
* Saving to the on-disk and s3 layers are done in the background
* Hashing of the sizing operations to find already sized images
* Redisdb is used for storing the bucket information of images sized


## Deployment

A Dockerfile is packaged with the project that includes a custom build of ImageMagick 6.9
with the latest libjpeg-turbo and libpng.

We use github.com/siddontang/ledisdb in production instead of redisdb. It's an Redis-API
compatible engine that is designed for long-term persistence of the data set.. pretty much
Redis on LevelDB.


## Other

* Imgry and its sizing operations can be used as a library, without the API server
* Imgry supports pluggable image processing engines, but for now comes packaged
with an ImageMagick engine by default (`imgry/imagick`)


## License

MIT License (See LICENSE file)

