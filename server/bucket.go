package server

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"

	"github.com/pressly/imgry"
	"github.com/pressly/lg"
)

var (
	ErrImageNotFound   = errors.New("image not found")
	ErrInvalidBucketID = errors.New("invalid bucket id - must be: [a-z0-9_:-] max-length: 40")

	BucketIDInvalidator = regexp.MustCompile(`(i?)[^a-z0-9\/_\-:\.]`)
)

type Bucket struct {
	ID string
}

func NewBucket(id string) (*Bucket, error) {
	if id == "" || len(id) > 40 {
		return nil, ErrInvalidBucketID
	}

	if BucketIDInvalidator.MatchString(id) {
		return nil, ErrInvalidBucketID
	}

	return &Bucket{ID: id}, nil
}

func (b *Bucket) AddImagesFromUrls(ctx context.Context, urls []string) ([]*Image, error) {
	responses, err := app.Fetcher.GetAll(ctx, urls)
	if err != nil {
		return nil, err
	}

	// TODO: do not release the image here, instead, return it
	// and get BucketFetchItem to set c.Env["image"] = image
	// and let the BucketGetItem release it instead..

	// Build images from fetched remote sources
	images := make([]*Image, 0, len(responses))
	for _, r := range responses {
		im := &Image{SrcURL: r.URL.String()}
		defer im.Release()

		if r.Status != 200 || len(r.Data) <= 0 {
			continue
		}

		im.Data = r.Data
		if err = im.LoadImage(); err != nil {
			lg.Errorf("LoadBlob data for %s returned error: %s", r.URL.String(), err)
			continue
		}
		if err := b.AddImage(ctx, im); err != nil {
			return images, err
		}

		images = append(images, im)
	}

	return images, nil
}

func (b *Bucket) AddImage(ctx context.Context, im *Image) error {
	if !im.IsValidImage() {
		return imgry.ErrInvalidImageData
	}
	im.genKey()

	// Save original size
	return b.DbSaveImage(ctx, im, nil)

	// TODO .. another time
	// Build and add seed image sizes for seed size < original
	// for _, sizing := range SEED_IMAGE_SIZES {
	//  // TODO: .. check if image > sizing.. if it is, then
	//  // let's make the smaller size, otherwise skip it..

	// note: build seed images in the background.. respond to the client right away

	//  seedSize, _ := image.MakeSize(sizing) // is this efficient.. or should we go more raw..?
	//  // maybe we should make InlineMakeSize() or something..?
	//  // or call it MakeNewSize() and Resize(sizing) .. like that..
	//  _ = dataStore.Put(StoreKey(seedSize), seedSize.Blob)
	//  SaveInDb(b, seedSize)
	//  // TODO: store this differently in the redis db .. as a label or something..
	// }
}

func (b *Bucket) GetImageSize(ctx context.Context, key string, sizing *imgry.Sizing) (*Image, error) {
	// Find the original image
	origIm, err := b.DbFindImage(ctx, key, nil)
	if err != nil {
		return nil, err
	}

	// Calculate the sizing ahead of time so our query is updated
	// and we can find it in our db
	sizing.CalcResizeRect(&imgry.Rect{origIm.Width, origIm.Height})
	sizing.Size.Width = sizing.GranularizedWidth()
	sizing.Size.Height = sizing.GranularizedHeight()

	// Find the specific size
	im, err := b.DbFindImage(ctx, key, sizing)
	if err != nil && err != ErrImageNotFound {
		return nil, err
	}
	if im != nil { // Got it!
		return im, nil
	}

	// Build a new size from the original
	im2, err := origIm.MakeSize(sizing)
	defer im2.Release()
	if err != nil {
		return nil, err
	}

	err = b.DbSaveImage(ctx, im2, sizing)
	return im2, err
}

// Loads the image from our table+data store with optional sizing
func (b *Bucket) DbFindImage(ctx context.Context, key string, sizing *imgry.Sizing) (*Image, error) {
	im := &Image{}

	idxKey := b.DbIndexKey(key, sizing)

	err := app.DB.HGet(idxKey, im)
	if err != nil {
		return nil, err
	}
	if im.Key == "" {
		idxKey = b.LegacyDbIndexKey(key, sizing)
		err = app.DB.HGet(idxKey, im)
		if err != nil {
			return nil, err
		}
		if im.Key == "" {
			return nil, ErrImageNotFound
		}
	}

	data, err := app.Chainstore.Get(ctx, idxKey) // TODO
	// data, err := app.Chainstore.Get(idxKey) // TODO
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, ErrImageNotFound
	}
	im.Data = data

	return im, nil
}

// Persists the image blob in our data store
func (b *Bucket) DbSaveImage(ctx context.Context, im *Image, sizing *imgry.Sizing) (err error) {
	if err := im.ValidateKey(); err != nil {
		return err
	}

	idxKey := b.DbIndexKey(im.Key, sizing)

	err = app.Chainstore.Put(ctx, idxKey, im.Data) // TODO
	// err = app.Chainstore.Put(idxKey, im.Data)
	if err != nil {
		return
	}
	err = app.DB.HSet(idxKey, im)
	return
}

// Persists the image blob in our data store
func (b *Bucket) UploadImage(ctx context.Context, im *Image) (err error) {
	if err := im.ValidateKey(); err != nil {
		return err
	}

	idxKey := b.DbIndexKey(im.Key, nil)

	err = app.Chainstore.Put(ctx, idxKey, im.Data) // TODO
	if err != nil {
		return
	}
	err = app.DB.HSet(idxKey, im)
	return
}

// TODO: should delete on *
func (b *Bucket) DbDelImage(ctx context.Context, key string) (err error) {
	idxKey := b.DbIndexKey(key, nil)

	err = app.DB.Del(idxKey) // + "*")
	if err != nil {
		return
	}

	err = app.Chainstore.Del(ctx, idxKey) // + "*") // TODO
	if err != nil {
		return
	}

	idxKey = b.LegacyDbIndexKey(key, nil)

	err = app.DB.Del(idxKey) // + "*")
	if err != nil {
		return
	}

	err = app.Chainstore.Del(ctx, idxKey) // + "*") // TODO

	return
}

func (b *Bucket) DbIndexKey(imageKey string, sizing *imgry.Sizing) string {
	k, _ := hex.DecodeString(imageKey)
	imageKey = base64.RawURLEncoding.EncodeToString(k)

	if sizing == nil {
		return fmt.Sprintf("%s/%s", imageKey[0:2], imageKey)
	}

	sk := sha1.Sum([]byte(sizing.ToQuery().Encode()))
	sizingKey := base64.RawURLEncoding.EncodeToString(sk[0:20])

	return fmt.Sprintf("%s/%s:q/%s", imageKey[0:2], imageKey, sizingKey)
}

func (b *Bucket) LegacyDbIndexKey(imageKey string, sizing *imgry.Sizing) string {
	key := fmt.Sprintf("%s/%s", b.ID, imageKey)
	if sizing != nil {
		key = fmt.Sprintf("%s:q/%x", key, sha1.Sum([]byte(sizing.ToQuery().Encode())))
	}
	return key
}
