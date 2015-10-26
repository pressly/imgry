package server

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/goware/lg"
	"github.com/pressly/chainstore"
	"github.com/pressly/chainstore/boltstore"
	"github.com/pressly/chainstore/lrumgr"
	"github.com/pressly/chainstore/memstore"
	"github.com/pressly/chainstore/metricsmgr"
	"github.com/pressly/chainstore/s3store"
	"github.com/pressly/go-metrics/librato"
	"github.com/rcrowley/go-metrics"
)

type Config struct {
	Bind        string `toml:"bind"`
	MaxProcs    int    `toml:"max_procs"`
	LogLevel    string `toml:"log_level"`
	CacheMaxAge int    `toml:"cache_max_age"`
	TmpDir      string `toml:"tmp_dir"`
	Profiler    bool   `toml:"profiler"`

	// [cluster]
	Cluster struct {
		LocalNode string   `toml:"local_node"`
		Nodes     []string `toml:"nodes"`
	} `toml:"cluster"`

	// [limits]
	Limits struct {
		MaxRequests    int `toml:"max_requests"`
		BacklogSize    int `toml:"backlog_size"`
		RequestTimeout time.Duration
		BacklogTimeout time.Duration
		MaxFetchers    int `toml:"max_fetchers"`
		MaxImageSizers int `toml:"max_image_sizers"`

		RequestTimeoutStr string `toml:"request_timeout"`
		BacklogTimeoutStr string `toml:"backlog_timeout"`
	} `toml:"limits"`

	// [db]
	DB struct {
		RedisUri string `toml:"redis_uri"`
	} `toml:"db"`

	// [airbrake]
	Airbrake struct {
		ApiKey string `toml:"api_key"`
	} `toml:"airbrake"`

	// [chainstore]
	Chainstore struct {
		Path          string `toml:"path"`
		MemCacheSize  int64  `toml:"mem_cache_size"`
		DiskCacheSize int64  `toml:"disk_cache_size"`
		S3Bucket      string `toml:"s3_bucket"`
		S3AccessKey   string `toml:"s3_access_key"`
		S3SecretKey   string `toml:"s3_secret_key"`
	} `toml:"chainstore"`

	// [librato]
	Librato struct {
		Enabled   bool   `toml:"enabled"`
		Email     string `toml:"email"`
		Token     string `toml:"token"`
		Namespace string `toml:"namespace"`
		Source    string `toml:"source"`
	} `toml:"librato"`
}

var (
	ErrNoConfigFile = errors.New("no configuration file specified")

	DefaultConfig = Config{}
)

func init() {
	cf := Config{
		Bind:        "0.0.0.0:4446",
		MaxProcs:    -1,
		LogLevel:    "INFO",
		CacheMaxAge: 0,
		TmpDir:      "",
		Profiler:    false,
	}

	cf.Limits.MaxRequests = 1000
	cf.Limits.BacklogSize = 5000
	cf.Limits.RequestTimeout = 45 * time.Second
	cf.Limits.BacklogTimeout = 1500 * time.Millisecond
	cf.Limits.MaxFetchers = 100
	cf.Limits.MaxImageSizers = 20

	DefaultConfig = cf
}

func NewConfig() *Config {
	cf := DefaultConfig
	return &cf
}

func NewConfigFromFile(confFile string, confEnv string) (*Config, error) {
	var err error

	if confFile == "" {
		confFile = confEnv
	}
	if _, err = os.Stat(confFile); os.IsNotExist(err) {
		return nil, ErrNoConfigFile
	}

	cf := NewConfig()

	if _, err = toml.DecodeFile(confFile, &cf); err != nil {
		return nil, err
	}
	return cf, nil
}

func (cf *Config) Apply() (err error) {
	// runtime
	if cf.MaxProcs <= 0 {
		cf.MaxProcs = runtime.NumCPU()
	}
	runtime.GOMAXPROCS(cf.MaxProcs)

	// logging
	if err := lg.SetLevelString(strings.ToLower(cf.LogLevel)); err != nil {
		return err
	}

	// limits
	if cf.Limits.RequestTimeoutStr != "" {
		to, err := time.ParseDuration(cf.Limits.RequestTimeoutStr)
		if err != nil {
			return err
		}
		cf.Limits.RequestTimeout = to
	}
	if cf.Limits.BacklogTimeoutStr != "" {
		to, err := time.ParseDuration(cf.Limits.BacklogTimeoutStr)
		if err != nil {
			return err
		}
		cf.Limits.BacklogTimeout = to
	}

	return nil
}

func (cf *Config) GetDB() (*DB, error) {
	db, err := NewDB(cf.DB.RedisUri)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func (cf *Config) GetChainstore() (chainstore.Store, error) {
	// First, reset cache storage path
	err := filepath.Walk(
		cf.Chainstore.Path,
		func(path string, info os.FileInfo, err error) error {
			if cf.Chainstore.Path == path {
				return nil // skip the root
			}
			if err = os.RemoveAll(path); err != nil {
				return fmt.Errorf("Failed to remove or clean the directory: %s, because: %s", path, err)
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	// TODO: impl another kind of lrumgr (or option) to be based on number of keys, not filesize
	// at which point, we can add a method called .Keys() that will return the keys
	// matching some query from a Store, and we can seed the LRU this way, and keep
	// the bolt data..

	// Build the stores and setup the chain
	memStore := memstore.New(cf.Chainstore.MemCacheSize * 1024 * 1024)

	diskStore := lrumgr.New(cf.Chainstore.DiskCacheSize*1024*1024,
		metricsmgr.New("fn.store.bolt", nil,
			boltstore.New(cf.Chainstore.Path+"store.db", "imgry"),
		),
	)

	var store chainstore.Store

	if cf.Chainstore.S3AccessKey != "" && cf.Chainstore.S3SecretKey != "" {
		s3Store := metricsmgr.New("fn.store.s3", nil,
			s3store.New(cf.Chainstore.S3Bucket, cf.Chainstore.S3AccessKey, cf.Chainstore.S3SecretKey),
		)

		store = chainstore.New(memStore, chainstore.Async(diskStore, s3Store))
		// store = chainstore.New(memStore, diskStore, s3Store)

	} else {
		store = chainstore.New(memStore, chainstore.Async(diskStore))
		// store = chainstore.New(memStore, diskStore)
	}

	chainstore.DefaultTimeout = 60 * time.Second // TODO: ....

	if err := store.Open(); err != nil {
		return nil, err
	}
	return store, nil
}

// Setup app stats & instrumentation
func (cf *Config) SetupLibrato() (err error) {

	if cf.Librato.Enabled {
		reporter := librato.NewReporter(
			metrics.DefaultRegistry,
			10e9,
			cf.Librato.Email,
			cf.Librato.Token,
			cf.Librato.Source,
			[]float64{0.5, 0.95},
			time.Millisecond,
		)
		reporter.Namespace = cf.Librato.Namespace
		go reporter.Run()

		// TODO: should we add "pulse" gauge metric to 1 ...? ...... or as a counter.....?

		// TODO: is there an easy way to offer sys.cpu sys.mem sys.swap stats..?
		// for convienience..

		// go metrics.Log(metrics.DefaultRegistry, 10e9, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))

		// Capture Runtime stats periodically.
		// NOTE: reading the MemStats will stop the world, so do this every > 1min
		metrics.RegisterRuntimeMemStats(metrics.DefaultRegistry)
		go metrics.CaptureRuntimeMemStats(metrics.DefaultRegistry, 60e9)
	} else {
		metrics.UseNilMetrics = true
	}

	return nil
}
