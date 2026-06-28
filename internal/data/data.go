package data

import (
	"errors"
	"log/slog"
	"review_service/internal/conf"
	"review_service/internal/data/query"
	"strings"

	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var ProviderSet = wire.NewSet(NewData, NewReviewRepo, NewDB)

// Data .
type Data struct {
	query *query.Query
	log   *slog.Logger
}

// NewData .
func NewData(db *gorm.DB, logger *slog.Logger) (*Data, func(), error) {
	cleanup := func() {
		logger.Info("closing the data resources")
	}
	query.SetDefault(db)

	return &Data{query: query.Q, log: logger}, cleanup, nil
}

func NewDB(cfg *conf.Data) (*gorm.DB, error) {
	switch strings.ToLower(cfg.Database.GetDriver()) {
	case "mysql":
		return gorm.Open(mysql.Open(cfg.Database.GetSource()))
	}
	return nil, errors.New("connect db fail unsupported db driver")
}
