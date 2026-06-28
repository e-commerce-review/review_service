package biz

import (
	"context"
	"log/slog"
	"review_service/internal/data/model"
)

type ReviewRepo interface {
	SaveReview(context.Context, *model.ReviewInfo) (*model.ReviewInfo, error)
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
	return uc.repo.SaveReview(ctx, review)
}
