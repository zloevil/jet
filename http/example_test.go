package http_test

import (
	"github.com/zloevil/jet"
	"github.com/zloevil/jet/http"
)

func ExampleNewHttpServer() {
	logFn := func() jet.CLogger { return jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.InfoLevel})) }

	srv := http.NewHttpServer(&http.Config{Port: "8080"}, logFn)
	srv.Listen()
	defer srv.Close()

	// register routes on srv.RootRouter
	_ = srv.RootRouter
}
