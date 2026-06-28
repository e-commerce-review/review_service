package biz

import (
	"context"
	"log/slog"
	v1 "review_service/api/review/v1"
	"review_service/internal/data/model"
	"review_service/pkg/snowflake"
)

type ReviewRepo interface {
	SaveReview(context.Context, *model.ReviewInfo) (*model.ReviewInfo, error)
	GetReviewByOrderID(context.Context, int64) ([]*model.ReviewInfo, error)
}

type ReviewUsecase struct {
	repo ReviewRepo
	log  *slog.Logger
}

func NewReviewUsecase(repo ReviewRepo, logger *slog.Logger) *ReviewUsecase {
	return &ReviewUsecase{
		repo: repo,
		log:  logger,
	}
}

func (uc *ReviewUsecase) CreateReview(ctx context.Context, review *model.ReviewInfo) (*model.ReviewInfo, error) {
	uc.log.DebugContext(ctx, "[biz] CreateReview", slog.Any("req", review))
	reviews, err := uc.repo.GetReviewByOrderID(ctx, review.OrderID)
	if err != nil {
		return nil, v1.ErrorDbFailed("query database failed")
	}
	if len(reviews) > 0 {
		return nil, v1.ErrorOrderReviewed("%d already reviewed", review.OrderID)
	}
	review.ReviewID = snowflake.GenID()
	return uc.repo.SaveReview(ctx, review)
}
