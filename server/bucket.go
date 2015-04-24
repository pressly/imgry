package server

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/pressly/imgry"
	"github.com/rcrowley/go-metrics"
)

var (
	ErrImageNotFound   = errors.New("image not found")
	ErrInvalidBucketID = errors.New("invalid bucket id - must be: [a-z0-9_:-] max-length: 40")

	// TODO: BucketIDValidator regexp here...
)

// TODO: unexport all of the Db methods...

type Bucket struct {
	ID string
}

func NewBucket(id string) (*Bucket, error) {
	b := &Bucket{ID: id}
	if _, err := b.ValidID(); err != nil {
		return nil, err
	}
	return b, nil
}

func (b *Bucket) ValidID() (bool, error) {
	if b.ID == "" || len(b.ID) > 40 {
		return false, ErrInvalidBucketID
	} else {
		m, _ := regexp.MatchString(`(i?)[^a-z0-9\/_\-:\.]`, b.ID)
		if !m {
			return true, nil
		} else {
			return false, ErrInvalidBucketID
		}
	}
}

func (b *Bucket) AddImagesFromUrls(urls []string) ([]*Image, error) {
	responses, err := app.HttpFetcher.GetAll(urls)
	if err != nil {
		return nil, err
	}

	// TODO: do not release the image here, instead, return it
	// and get BucketFetchItem to set c.Env["image"] = image
	// and let the BucketGetItem release it instead..

	// Build images from fetched remote sources
	images := make([]*Image, len(responses))
	for i, r := range responses {
		images[i] = NewImageFromSrcUrl(r.URL.String())
		defer images[i].Release()
		if r.Status == 200 && len(r.Data) > 0 {
			images[i].Data = r.Data
			if err = images[i].LoadImage(); err != nil {
				lg.Error("LoadBlob data for %s returned error: %s", r.URL.String(), err)
			}
		}
	}

	return images, b.AddImages(images)
}

// TODO: .. how do handle errors here... ? each image would
// have it's own error .. should we put an Err on each image object...?
// or return an errList ..
func (b *Bucket) AddImages(images []*Image) (err error) {
	for _, i := range images {
		err = b.AddImage(i)
	}
	return err
}

func (b *Bucket) AddImage(i *Image) (err error) {
	if !i.IsValidImage() || len(i.Data) == 0 {
		return imgry.ErrInvalidImageData
	}

	// Save original size
	err = b.DbSaveImage(i, nil)

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

	return
}

func (b *Bucket) GetImageSize(key string, sizing *imgry.Sizing) (*Image, error) {
	m := metrics.GetOrRegisterTimer("fn.bucket.GetImageSize", nil)
	defer m.UpdateSince(time.Now())

	// Find the original image
	origIm, err := b.DbFindImage(key, nil)
	if err != nil {
		return nil, err
	}

	// Calculate the sizing ahead of time so our query is updated
	// and we can find it in our db
	sizing.CalcResizeRect(&imgry.Rect{origIm.Width, origIm.Height})
	sizing.Size.Width = sizing.GranularizedWidth()
	sizing.Size.Height = sizing.GranularizedHeight()

	// Find the specific size
	im, err := b.DbFindImage(key, sizing)
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

	err = b.DbSaveImage(im2, sizing)
	return im2, err
}

// Loads the image from our table+data store with optional sizing
func (b *Bucket) DbFindImage(key string, optSizing ...*imgry.Sizing) (*Image, error) {
	m := metrics.GetOrRegisterTimer("fn.bucket.DbFindImage", nil)
	defer m.UpdateSince(time.Now())

	var sizing *imgry.Sizing
	if len(optSizing) > 0 { // sizing is optional
		sizing = optSizing[0]
	}

	idxKey := b.DbIndexKey(key, sizing)

	im := &Image{}
	err := app.DB.HGet(idxKey, im)
	if err != nil {
		return nil, err
	}
	if im.Key == "" {
		return nil, ErrImageNotFound
	}

	data, err := app.Chainstore.Get(idxKey)
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
func (b *Bucket) DbSaveImage(im *Image, sizing *imgry.Sizing) (err error) {
	m := metrics.GetOrRegisterTimer("fn.bucket.DbSaveImage", nil)
	defer m.UpdateSince(time.Now())

	if err := im.ValidateKey(); err != nil {
		return err
	}

	idxKey := b.DbIndexKey(im.Key, sizing)

	err = app.Chainstore.Put(idxKey, im.Data)
	if err != nil {
		return
	}
	err = app.DB.HSet(idxKey, im)
	return
}

// TODO: should delete on *
func (b *Bucket) DbDelImage(key string) (err error) {
	idxKey := b.DbIndexKey(key, nil)

	err = app.DB.Del(idxKey) // + "*")
	if err != nil {
		return
	}

	err = app.Chainstore.Del(idxKey) // + "*")
	return
}

func (b *Bucket) DbIndexKey(imageKey string, optSizing ...*imgry.Sizing) string {
	var sizing *imgry.Sizing
	if len(optSizing) > 0 { // sizing is optional
		sizing = optSizing[0]
	}
	key := fmt.Sprintf("%s/%s", b.ID, imageKey)
	if sizing != nil {
		key = fmt.Sprintf("%s:q/%s", key, sha1Hash(sizing.ToQuery().Encode()))
	}
	return key
}
