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
	"github.com/armon/go-metrics"
	"github.com/goware/lg"
	"github.com/pressly/chainstore"
	"github.com/pressly/chainstore/boltstore"
	"github.com/pressly/chainstore/lrumgr"
	"github.com/pressly/chainstore/memstore"
	"github.com/pressly/chainstore/metricsmgr"
	"github.com/pressly/chainstore/s3store"
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

	// [statsd]
	StatsD struct {
		Enabled     bool   `toml:"enabled"`
		Address     string `toml:"address"`
		ServiceName string `toml:"service_name"`
	}
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
	// chainstore.DefaultTimeout = 60 * time.Second // TODO: ....

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
	memStore := metricsmgr.New("fn.store.mem", nil,
		memstore.New(cf.Chainstore.MemCacheSize*1024*1024),
	)

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

		// store = chainstore.New(memStore, chainstore.Async(diskStore, s3Store))
		store = chainstore.New(memStore, chainstore.Async(s3Store))
	} else {
		store = chainstore.New(memStore, chainstore.Async(diskStore))
	}

	if err := store.Open(); err != nil {
		return nil, err
	}
	return store, nil
}

func (cf *Config) SetupStatsD() error {
	if cf.StatsD.Enabled {
		sink, err := metrics.NewStatsdSink(cf.StatsD.Address)
		if err != nil {
			return err
		}

		config := &metrics.Config{
			ServiceName:          cf.StatsD.ServiceName, // Client service name
			HostName:             "",
			EnableHostname:       true,             // Enable hostname prefix
			EnableRuntimeMetrics: true,             // Enable runtime profiling
			EnableTypePrefix:     false,            // Disable type prefix
			TimerGranularity:     time.Millisecond, // Timers are in milliseconds
			ProfileInterval:      time.Second * 60, // Poll runtime every minute
		}

		config.HostName, _ = os.Hostname()

		metrics.NewGlobal(config, sink)
	}
	return nil
}
