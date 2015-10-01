package main

import (
	"flag"
	"log"
	"os"
	"syscall"

	"github.com/goware/lg"
	"github.com/pressly/imgry"
	"github.com/pressly/imgry/server"
	"github.com/zenazn/goji/graceful"
)

var (
	flags    = flag.NewFlagSet("imgry", flag.ExitOnError)
	confFile = flags.String("config", "", "path to config file")
)

func main() {
	var err error
	flags.Parse(os.Args[1:])

	conf, err := server.NewConfigFromFile(*confFile, os.Getenv("CONFIG"))
	if err != nil {
		log.Fatal(err)
	}

	srv := server.New(conf)
	if err := srv.Configure(); err != nil {
		log.Fatal(err)
	}

	lg.Infof("** Imgry Server v%s at %s **", imgry.VERSION, srv.Config.Bind)
	lg.Infof("** Engine: %s", srv.ImageEngine.Version())

	graceful.AddSignal(syscall.SIGINT, syscall.SIGTERM)
	graceful.PreHook(func() { srv.Close() })

	err = graceful.ListenAndServe(srv.Config.Bind, srv.NewRouter())
	if err != nil {
		lg.Fatal(err.Error())
	}
	graceful.Wait()
}
