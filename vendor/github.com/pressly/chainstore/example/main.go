package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pressly/chainstore"
	"github.com/pressly/chainstore/boltstore"
	"github.com/pressly/chainstore/lrumgr"
	"github.com/pressly/chainstore/metricsmgr"
	"github.com/pressly/chainstore/s3store"
	"golang.org/x/net/context"
)

var (
	bucketID  string
	accessKey string
	secretKey string
)

func init() {
	bucketID = os.Getenv("S3_BUCKET")
	accessKey = os.Getenv("S3_ACCESS_KEY")
	secretKey = os.Getenv("S3_SECRET_KEY")
}

func main() {
	ctx := context.Background()

	diskStore := lrumgr.New(500*1024*1024, // 500MB of working data
		metricsmgr.New("chainstore.ex.bolt",
			boltstore.New("/tmp/store.db", "myBucket"),
		),
	)

	remoteStore := metricsmgr.New("chainstore.ex.s3",
		// NOTE: you'll have to supply your own keys in order for this example to work properly
		s3store.New(bucketID, accessKey, secretKey),
	)

	dataStore := chainstore.New(diskStore, remoteStore)

	// OR.. define inline. Except, I wanted to show store independence & state.
	/*
		dataStore := chainstore.New(
			lrumgr.New(500*1024*1024, // 500MB of working data
				metricsmgr.New("chainstore.ex.bolt",
					boltstore.New("/tmp/store.db", "myBucket"),
				),
			),
			metricsmgr.New("chainstore.ex.s3",
				// NOTE: you'll have to supply your own keys in order for this example to work properly
				s3store.New("myBucket", "access-key", "secret-key"),
			),
		)
	*/

	var err error

	err = dataStore.Open()
	if err != nil {
		log.Fatalf("Open: %q", err)
	}

	// Since we've used the metricsManager above (metricsmgr), any calls to the boltstore
	// and s3store will be measured. Next is to send metrics to librato, graphite, influxdb,
	// whatever.. via github.com/goware/go-metrics
	// go librato.Librato(metrics.DefaultRegistry, 10e9, ...)

	//--

	// Save the object in the chain. It will be Put() synchronously into diskStore,
	// the boltdb engine, and then immediately dispatch background Put()'s to the
	// other stores down the chain, in this case S3.
	fmt.Println("Example 1...")
	obj := []byte{1, 2, 3}
	err = dataStore.Put(ctx, "k", obj)
	if err != nil {
		log.Fatalf("Put: %q", err)
	}
	fmt.Println("Put 'k':", obj, "in the chain")

	v, err := dataStore.Get(ctx, "k")
	if err != nil {
		log.Fatalf("Put: %q", err)
	}
	fmt.Println("Grabbing 'k' from the chain:", v) // => [1 2 3]

	// For demonstration, let's grab the key directly from the store instead of
	// through the chain. This is pretty much the same as above, as the chain's Get()
	// stops once it finds the object.
	v, err = diskStore.Get(ctx, "k")
	if err != nil {
		log.Fatalf("Put: %q", err)
	}
	fmt.Println("Grabbing 'k' directly from boltdb:", v) // => [1 2 3]

	// lets pause for a moment and then try to retrieve the value from the s3 store
	time.Sleep(1e9)

	// Grab the object from s3
	v, err = remoteStore.Get(ctx, "k")
	if err != nil {
		log.Fatalf("Put: %q", err)
	}
	fmt.Println("Grabbing 'k' directly from s3:", v) // => [1 2 3]

	// Delete the object from everywhere
	dataStore.Del(ctx, "k")
	time.Sleep(1e9) // pause for s3 demo
	v, _ = dataStore.Get(ctx, "k")
	fmt.Println("Deleted 'k' from the chain (all stores). Get(k) returns:", v)

	//--

	// Another interesting behavior of the chain is when doing a Get(), it goes down
	// the entire chain looking for the value, and when found, it will Put() that
	// object back up the chain for subsequent retrievals. Lets see..
	fmt.Println("Example 2...")
	obj = []byte("hope you enjoy")
	err = dataStore.Put(ctx, "hi", obj)
	if err != nil {
		log.Fatalf("Put: %q", err)
	}
	fmt.Println("Put 'hi':", obj, "in the chain")
	time.Sleep(1e9) // lets wait for s3 again with more then enough time

	err = diskStore.Del(ctx, "hi")
	if err != nil {
		log.Fatalf("Get: %q", err)
	}

	v, _ = diskStore.Get(ctx, "hi")
	fmt.Println("Delete 'hi' from boltdb. diskStore.Get(k) returns:", v)

	v, err = dataStore.Get(ctx, "hi")
	if err != nil {
		log.Fatalf("Get: %q", err)
	}
	fmt.Println("Let's ask the chain for 'hi':", v)
	time.Sleep(1e9) // pause for bg routine to fill our local cache

	// The diskStore now has the value again from remoteStore lower down the chain.
	v, err = diskStore.Get(ctx, "hi")
	if err != nil {
		log.Fatalf("Get: %q", err)
	}
	fmt.Println("Now, let's ask our diskStore again! diskStore.Get(k) returns:", v)

	// Also.. even though it hasn't been demonstrated here, the diskStore will only
	// store a max of 500MB (as defined with diskLru) worth of objects. Give it a shot.
}

/* OUTPUT:

Example 1...
Put 'k': [1 2 3] in the chain
Grabbing 'k' from the chain: [1 2 3]
Grabbing 'k' directly from boltdb: [1 2 3]
Grabbing 'k' directly from s3: [1 2 3]
Deleted 'k' from the chain (all stores). Get(k) returns: []
Example 2...
Put 'hi': [104 111 112 101 32 121 111 117 32 101 110 106 111 121] in the chain
Delete 'hi' from boltdb. diskStore.Get(k) returns: []
Let's ask the chain for 'hi': [104 111 112 101 32 121 111 117 32 101 110 106 111 121]
Now, let's ask our diskStore again! diskStore.Get(k) returns: [104 111 112 101 32 121 111 117 32 101 110 106 111 121]

*/
