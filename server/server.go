package server

import (
	"net/http"

	"github.com/goware/heartbeat"
	"github.com/goware/httpcoala"
	"github.com/goware/lg"
	"github.com/pressly/chainstore"
	"github.com/pressly/chi"
	"github.com/pressly/chi/middleware"
	"github.com/pressly/consistentrd"
	"github.com/pressly/imgry"
	"github.com/pressly/imgry/imagick"
	"golang.org/x/net/context"
)

var (
	app     *Server
	respond = NewResponder()
)

type Server struct {
	Config      *Config
	DB          *DB
	Chainstore  chainstore.Store
	HttpFetcher *HttpFetcher
	ImageEngine imgry.Engine

	Ctx        context.Context
	CancelFunc context.CancelFunc
}

func New(conf *Config) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	app = &Server{Ctx: ctx, CancelFunc: cancel, Config: conf}
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

	if err := srv.Config.SetupLibrato(); err != nil {
		return err
	}

	srv.HttpFetcher = NewHttpFetcher()

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
	srv.CancelFunc()
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

	r := chi.NewRouter()

	r.Use(ParentContext(srv.Ctx))
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(middleware.CloseNotify)
	// r.Use(middleware.Timeout(cf.Limits.RequestTimeout))
	r.Use(httpcoala.Route("HEAD", "GET"))

	r.Use(heartbeat.Route("/ping"))
	r.Use(heartbeat.Route("/favicon.ico"))

	if cf.Airbrake.ApiKey != "" {
		r.Use(AirbrakeRecoverer(cf.Airbrake.ApiKey))
	}

	if srv.Config.Profiler {
		r.Mount("/debug", middleware.NoCache, Profiler())
	}

	r.Get("/", trackRoute("root"), func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("."))
	})

	r.Get("/info", trackRoute("imageInfo"), GetImageInfo)

	r.Route("/:bucket", func(r chi.Router) {
		r.Post("/", BucketImageUpload)
		r.Get("/", conrd.RouteWithParams("url"), trackRoute("bucketV1GetItem"), BucketGetIndex)
		r.Get("/fetch", conrd.RouteWithParams("url"), trackRoute("bucketV1GetItem"), BucketFetchItem)

		// TODO: review
		r.Get("/add", trackRoute("bucketAddItems"), BucketAddItems)
		r.Get("/:key", conrd.Route(), BucketGetItem)
		r.Delete("/:key", conrd.Route(), BucketDeleteItem)
	})

	// DEPRECATED
	r.Get("/fetch", trackRoute("bucketV0GetItem"), BucketV0FetchItem) // for Pressilla v2 apps
	r.Get("/:bucket//fetch", conrd.RouteWithParams("url"), trackRoute("bucketV1GetItem"), BucketFetchItem)
	// --

	return r
}
