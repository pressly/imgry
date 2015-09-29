package server

import (
	"net/http"

	"github.com/pressly/chainstore"
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
	// TODO: Engine ...
}

func New(conf *Config) *Server {
	app = &Server{Config: conf}
	return app
}

func (srv *Server) Configure() (err error) {
	if err := srv.Config.SetupRuntime(); err != nil {
		return err
	}

	srv.Config.SetupLogging()

	if srv.Config.Server.TmpDir != "" {
		// clear temp directory on start if specified
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

	return nil
}

func (srv *Server) NewRouter() http.Handler {
	return NewRouter()
}

func (srv *Server) Close() {
	srv.DB.Close()
	srv.Chainstore.Close()
}
