package server

import (
	"net/http"

	"github.com/goware/lg"
	"github.com/pressly/chainstore"
	"github.com/pressly/imgry"
	"github.com/pressly/imgry/imagick"
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

func (srv *Server) NewRouter() http.Handler {
	return NewRouter()
}

func (srv *Server) Close() {
	lg.Info("server shutting down...")
	srv.ImageEngine.Terminate()
	srv.DB.Close()
	srv.Chainstore.Close()
}
