package mongodb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/zloevil/jet"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"os"
	"strings"
	"time"
)

type Config struct {
	ConnectionString string
	TimeoutSec       *int
	CertPath         *string
}

type Storage struct {
	Instance *mongo.Client
	lg       jet.CLoggerFunc
}

func Open(config *Config, logger jet.CLoggerFunc) (*Storage, error) {

	s := &Storage{lg: logger}

	l := logger().Cmp("mongo").Mth("open").Dbg("connecting...")

	// setup options
	opts := options.Client().ApplyURI(config.ConnectionString)

	// setup connection timeout if specified
	if config.TimeoutSec != nil {
		opts = opts.SetConnectTimeout(time.Duration(*config.TimeoutSec) * time.Second)
	}

	// tls configuration
	tlsConfig, err := s.makeTlsConfig(config.CertPath)
	if err != nil {
		return nil, err
	}
	if tlsConfig != nil {
		opts.SetTLSConfig(tlsConfig)
	}

	// connect (in v2 mongo.Connect no longer takes a context)
	s.Instance, err = mongo.Connect(opts)
	if err != nil {
		return nil, ErrConnection(err)
	}

	l.Dbg("ok")

	return s, nil
}

func (s *Storage) Close(ctx context.Context) {
	_ = s.Instance.Disconnect(ctx)
}

func (s *Storage) makeTlsConfig(dbCertPath *string) (*tls.Config, error) {

	if dbCertPath == nil || strings.Compare(*dbCertPath, "") == 0 {
		return nil, nil
	}

	rootPEM, err := os.ReadFile(*dbCertPath)
	if err != nil {
		return nil, ErrReadCertFile(err)
	}

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(rootPEM)
	if !ok {
		return nil, ErrAppendCert(err)
	}

	return &tls.Config{
		RootCAs:            roots,
		InsecureSkipVerify: true,
	}, nil
}
