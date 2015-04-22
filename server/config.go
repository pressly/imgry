package server

import (
	"errors"
	"fmt"
	stdlog "log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/pressly/go-metrics/librato"
	"github.com/rcrowley/go-metrics"

	"github.com/BurntSushi/toml"
	"github.com/op/go-logging"
	"github.com/pressly/chainstore"
	"github.com/pressly/chainstore/boltstore"
	"github.com/pressly/chainstore/lrumgr"
	"github.com/pressly/chainstore/memstore"
	"github.com/pressly/chainstore/metricsmgr"
	"github.com/pressly/chainstore/s3store"
)

var (
	ErrNoConfigFile = errors.New("no configuration file specified")
)

type Config struct {
	Server     ServerConfig     `toml:"server"`
	Cluster    ClusterConfig    `toml:"cluster"`
	DbConfig   DbConfig         `toml:"db"`
	Chainstore ChainstoreConfig `toml:"chainstore"`
	Librato    LibratoConfig    `toml:"librato"`
}

type ServerConfig struct {
	Addr          string `toml:"addr"`
	MaxProcs      int    `toml:"max_procs"`
	LogLevel      string `toml:"log_level"`
	CacheMaxAge   int    `toml:"cache_max_age"`
	SizingThruput int    `toml:"sizing_thruput"`
}

type ClusterConfig struct {
	LocalNode string   `toml:"local_node"`
	Nodes     []string `toml:"nodes"`
}

type DbConfig struct {
	RedisUri string `toml:"redis_uri"`
}

type ChainstoreConfig struct {
	Path          string `toml:"path"`
	MemCacheSize  int64  `toml:"mem_cache_size"`
	DiskCacheSize int64  `toml:"disk_cache_size"`
	S3Bucket      string `toml:"s3_bucket"`
	S3AccessKey   string `toml:"s3_access_key"`
	S3SecretKey   string `toml:"s3_secret_key"`
}

type LibratoConfig struct {
	Enabled   bool   `toml:"enabled"`
	Email     string `toml:"email"`
	Token     string `toml:"token"`
	Namespace string `toml:"namespace"`
	Source    string `toml:"source"`
}

func NewConfig() *Config {
	return &Config{}
}

func NewConfigFromFile(confFile string, confEnv string) (*Config, error) {
	var cf *Config
	var err error

	if confFile == "" {
		confFile = confEnv
	}

	if _, err = os.Stat(confFile); os.IsNotExist(err) {
		return nil, ErrNoConfigFile
	}

	cf = &Config{}
	if _, err = toml.DecodeFile(confFile, &cf); err != nil {
		return nil, err
	}
	return cf, nil
}

func (cf *Config) SetupRuntime() (err error) {
	if cf.Server.MaxProcs <= 0 {
		cf.Server.MaxProcs = runtime.NumCPU()
	}
	runtime.GOMAXPROCS(cf.Server.MaxProcs)
	return nil
}

type logProxyWriter struct {
	Logger *logging.Logger
}

func (l *logProxyWriter) Write(p []byte) (n int, err error) {
	l.Logger.Info("%s", p)
	return len(p), nil
}

func (cf *Config) SetupLogger(lg *logging.Logger) error {
	logging.SetFormatter(logging.MustStringFormatter("%{level} %{message}"))
	logging.SetBackend(logging.NewLogBackend(os.Stdout, "", stdlog.LstdFlags))

	// Setup the default log level
	cfLevel := cf.Server.LogLevel
	if cfLevel == "" {
		cfLevel = "INFO"
	}
	logLevel, err := logging.LogLevel(cfLevel)
	if err != nil {
		return err
	}
	logging.SetLevel(logLevel, lg.Module)

	// Redirect the standard logger
	stdlog.SetOutput(&logProxyWriter{lg})
	stdlog.SetFlags(0)

	return nil
}

func (cf *Config) GetDB() (*DB, error) {
	db, err := NewDB(cf.DbConfig.RedisUri)
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
			boltstore.New(cf.Chainstore.Path+"store.db", "imgs"),
		),
	)

	var store chainstore.Store

	if cf.Chainstore.S3AccessKey != "" && cf.Chainstore.S3SecretKey != "" {
		s3Store := metricsmgr.New("fn.store.s3", nil,
			s3store.New(cf.Chainstore.S3Bucket, cf.Chainstore.S3AccessKey, cf.Chainstore.S3SecretKey),
		)

		store = chainstore.New(memStore, chainstore.Async(diskStore, s3Store))
	} else {
		store = chainstore.New(memStore, chainstore.Async(diskStore))
	}

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
