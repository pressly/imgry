package server

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	raven "github.com/getsentry/raven-go"
	"github.com/goware/errorx"
	"github.com/pkg/errors"
	"github.com/pressly/chainstore"
	"github.com/pressly/chainstore/boltstore"
	"github.com/pressly/chainstore/lrumgr"
	"github.com/pressly/chainstore/memstore"
	"github.com/pressly/chainstore/s3store"
	"github.com/pressly/lg"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Bind        string `toml:"bind"`
	CacheMaxAge int    `toml:"cache_max_age"`
	TmpDir      string `toml:"tmp_dir"`
	Profiler    bool   `toml:"profiler"`

	LogLevel string `toml:"log_level"`
	ErrLevel string `toml:"err_level"`

	// [cluster]
	Cluster struct {
		LocalNode string   `toml:"local_node"`
		Nodes     []string `toml:"nodes"`
	} `toml:"cluster"`

	// [limits]
	Limits struct {
		// Throttler settings
		MaxRequests       int    `toml:"max_requests"`
		BacklogSize       int    `toml:"backlog_size"`
		BacklogTimeoutStr string `toml:"backlog_timeout"`
		BacklogTimeout    time.Duration

		// Global request timeout
		RequestTimeoutStr string `toml:"request_timeout"`
		RequestTimeout    time.Duration

		// Imgry limits
		MaxFetchers    int `toml:"max_fetchers"`
		MaxImageSizers int `toml:"max_image_sizers"`
	} `toml:"limits"`

	HostExtraQueryParams map[string]url.Values `toml:"host_extra_query_params"`

	// [db]
	DB struct {
		RedisUri string `toml:"redis_uri"`
	} `toml:"db"`

	// [sentry]
	Sentry struct {
		DSN string `toml:"dsn"`
	} `toml:"sentry"`

	// [chainstore]
	Chainstore struct {
		Path          string `toml:"path"`
		MemCacheSize  int64  `toml:"mem_cache_size"`
		DiskCacheSize int64  `toml:"disk_cache_size"`
		S3Bucket      string `toml:"s3_bucket"`
		S3AccessKey   string `toml:"s3_access_key"`
		S3SecretKey   string `toml:"s3_secret_key"`
		S3Region      string `toml:"s3_region"`
		KMSKeyID      string `toml:"kms_key_id"`
	} `toml:"chainstore"`

	// [ssl]
	SSL struct {
		Cert string `toml:"cert"`
		Key  string `toml:"key"`
	} `toml:"ssl"`
}

var (
	ErrNoConfigFile = errors.New("no configuration file specified")

	DefaultConfig = Config{}
)

func init() {
	cf := Config{
		Bind:        "0.0.0.0:4446",
		LogLevel:    "INFO",
		CacheMaxAge: 0,
		TmpDir:      "",
		Profiler:    false,
	}

	// Available RAM / Avg RAM a single job requires. (e.g.: 4096 / 50 = 81)
	cf.Limits.MaxRequests = 80

	// (Maximum waiting time for a single job / Avg time a single job requires) * Throttle limit. (e.g.: (30/5)*80 = 480)
	cf.Limits.BacklogSize = 500

	// Maximum waiting time for a single job. (e.g.: 30s)
	cf.Limits.BacklogTimeout = 30 * time.Second

	// Max overall time to respond to the request
	cf.Limits.RequestTimeout = 45 * time.Second

	// Max parallel imgry fetchers
	cf.Limits.MaxFetchers = 100

	// Max parallel image operations
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
	// logging

	level, err := logrus.ParseLevel(strings.ToLower(cf.LogLevel))
	if err != nil {
		return errors.Wrap(err, "failed to parse logrus level")
	}
	if lg.DefaultLogger == nil {
		lg.DefaultLogger = logrus.New()
		lg.DefaultLogger.Formatter = &logrus.JSONFormatter{}
		lg.RedirectStdlogOutput(lg.DefaultLogger)
	}
	lg.DefaultLogger.Level = level

	if cf.Sentry.DSN != "" {
		raven.SetDSN(cf.Sentry.DSN)
		lg.AlertFn = Logger
	}

	// Errorx logging level
	switch cf.ErrLevel {
	case "INFO":
		errorx.SetVerbosity(errorx.Info)
	case "VERBOSE":
		errorx.SetVerbosity(errorx.Verbose)
	case "DEBUG":
		errorx.SetVerbosity(errorx.Debug)
	case "TRACE":
		errorx.SetVerbosity(errorx.Trace)
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

	// Build the stores and setup the chain
	memStore := memstore.New(cf.Chainstore.MemCacheSize * 1024 * 1024)
	diskStore := lrumgr.New(cf.Chainstore.DiskCacheSize*1024*1024, boltstore.New(cf.Chainstore.Path+"store.db", "imgry"))

	var store chainstore.Store

	if cf.Chainstore.S3AccessKey != "" && cf.Chainstore.S3SecretKey != "" {
		s3Store := s3store.New(s3store.Config{
			S3Bucket:    cf.Chainstore.S3Bucket,
			S3AccessKey: cf.Chainstore.S3AccessKey,
			S3SecretKey: cf.Chainstore.S3SecretKey,
			S3Region:    cf.Chainstore.S3Region,
			KMSKeyID:    cf.Chainstore.KMSKeyID,
		})
		// store = chainstore.New(memStore, chainstore.Async(diskStore, s3Store))
		store = chainstore.New(memStore, chainstore.Async(nil, s3Store))
	} else {
		store = chainstore.New(memStore, chainstore.Async(nil, diskStore))
	}

	if err := store.Open(); err != nil {
		return nil, err
	}
	return store, nil
}
