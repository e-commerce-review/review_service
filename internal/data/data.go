package data

import (
	"errors"
	"log/slog"
	"review_service/internal/conf"
	"review_service/internal/data/query"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var ProviderSet = wire.NewSet(NewData, NewReviewRepo, NewDB, NewESClient)

// Data .
type Data struct {
	query *query.Query
	log   *slog.Logger
	es    *elasticsearch.TypedClient
}

// NewData .
func NewData(db *gorm.DB, esclient *elasticsearch.TypedClient, logger *slog.Logger) (*Data, func(), error) {
	cleanup := func() {
		logger.Info("closing the data resources")
	}
	query.SetDefault(db)

	return &Data{query: query.Q, log: logger, es: esclient}, cleanup, nil
}

func NewESClient(cfg *conf.Elasticsearch) (*elasticsearch.TypedClient, error) {
	esCfg := elasticsearch.Config{
		Addresses: cfg.GetAddresses(),
	}
	return elasticsearch.NewTypedClient(esCfg)
}

func NewDB(cfg *conf.Data) (*gorm.DB, error) {
	switch strings.ToLower(cfg.Database.GetDriver()) {
	case "mysql":
		return gorm.Open(mysql.Open(cfg.Database.GetSource()))
	}
	return nil, errors.New("connect db fail unsupported db driver")
}
