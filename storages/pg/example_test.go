package pg_test

import (
	"github.com/zloevil/jet"
	"github.com/zloevil/jet/storages/pg"
)

func ExampleOpen() {
	logFn := func() jet.CLogger { return jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.InfoLevel})) }

	db, err := pg.Open(&pg.DbConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "app",
		Password: "secret",
		DBName:   "app",
	}, logFn)
	if err != nil {
		return
	}
	defer db.Close()

	// db.Instance is the configured *gorm.DB
	_ = db.Instance
}
