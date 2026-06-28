package data

import (
	"context"
	"log/slog"
	"review_service/internal/biz"
	"review_service/internal/data/model"
)

type reviewRepo struct {
	data *Data
	log  *slog.Logger
}

// NewReviewRepo .
func NewReviewRepo(data *Data, logger *slog.Logger) biz.ReviewRepo {
	return &reviewRepo{
		data: data,
		log:  logger,
	}
}

func (r *reviewRepo) SaveReview(ctx context.Context, review *model.ReviewInfo) (*model.ReviewInfo, error) {
	err := r.data.query.ReviewInfo.
		WithContext(ctx).
		Save(review)
	return review, err
}
