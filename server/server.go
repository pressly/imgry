package server

import (
	"net/http"

	"github.com/goware/cors"
	"github.com/pressly/chainstore"
	"github.com/pressly/chi"
	"github.com/pressly/chi/middleware"
	"github.com/pressly/consistentrd"
	"github.com/pressly/imgry"
	"github.com/pressly/imgry/imagick"
	"github.com/pressly/lg"
	"github.com/sirupsen/logrus"
)

var (
	app     *Server
	respond = NewResponder()
)

type Server struct {
	Config      *Config
	DB          *DB
	Chainstore  chainstore.Store
	Fetcher     *Fetcher
	ImageEngine imgry.Engine
}

func New(conf *Config) *Server {
	app = &Server{Config: conf}
	return app
}

func (srv *Server) Configure() (err error) {
	if err := srv.Config.Apply(); err != nil {
		return err
	}

	srv.DB, err = srv.Config.GetDB()
	if err != nil {
		return err
	}

	srv.Chainstore, err = srv.Config.GetChainstore()
	if err != nil {
		return err
	}

	srv.Fetcher = NewFetcher()

	tmpDir := srv.Config.TmpDir
	srv.ImageEngine = imagick.Engine{}
	if err := srv.ImageEngine.Initialize(tmpDir); err != nil {
		return err
	}

	return nil
}

// Close signals to the server that should deny new requests
// and finish up requests in progress.
func (srv *Server) Close() {
	lg.Info("closing server..")
}

// Shutdown will release other resources and halt the server.
func (srv *Server) Shutdown() {
	srv.ImageEngine.Terminate()
	srv.DB.Close()
	srv.Chainstore.Close()
	lg.Info("server shutdown.")
}

func (srv *Server) NewRouter() http.Handler {
	cf := srv.Config

	conrd, err := consistentrd.New(cf.Cluster.LocalNode, cf.Cluster.Nodes)
	if err != nil {
		panic(err)
	}
	_ = conrd

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	if cf.Sentry.DSN != "" {
		lg.DefaultLogger.Formatter = &logrus.JSONFormatter{}
		r.Use(lg.SanitizingRequestLogger(lg.DefaultLogger, map[string]string{
			"jwt":   "[token-redacted]",
			"state": "[token-redacted]",
			"email": "[email-redacted]",
		}))
		r.Use(CapturePanic())
	} else {
		lg.DefaultLogger.Formatter = &logrus.TextFormatter{}
		r.Use(lg.RequestLogger(lg.DefaultLogger))
		r.Use(lg.PrintPanics)
	}

	r.Use(middleware.ThrottleBacklog(cf.Limits.MaxRequests, cf.Limits.BacklogSize, cf.Limits.BacklogTimeout))
	r.Use(middleware.CloseNotify)
	r.Use(middleware.Timeout(cf.Limits.RequestTimeout))
	// r.Use(httpcoala.Route("HEAD", "GET"))

	r.Use(middleware.Heartbeat("/ping"))

	if srv.Config.Profiler {
		r.Mount("/debug", middleware.Profiler())
	}

	r.Get("/", Index)
	r.Get("/info", GetImageInfo)

	r.Route("/:bucket", func(r chi.Router) {
		r.Use(BucketURLCtx)

		r.Post("/", BucketImageUpload)
		r.Get("/add", BucketAddItems)

		r.Group(func(r chi.Router) {
			cors := cors.New(cors.Options{
				AllowedOrigins:   []string{"*"},
				AllowedMethods:   []string{"GET", "OPTIONS"},
				AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
				ExposedHeaders:   []string{"Link"},
				AllowCredentials: true,
				MaxAge:           300, // Maximum value not ignored by any of major browsers
			})
			r.Use(cors.Handler)

			r.Get("/", BucketGetIndex)
			r.Get("/fetch", BucketFetchItem)
		})

		r.Get("/:key", BucketGetItem)
		r.Delete("/:key", BucketDeleteItem)
	})

	return r
}

func Index(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte(`.`))
}
