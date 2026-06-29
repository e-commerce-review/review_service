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
	GetReview(context.Context, int64) (*model.ReviewInfo, error)
	AuditReview(context.Context, *AuditParam) error
	SaveReply(context.Context, *model.ReviewReplyInfo) (*model.ReviewReplyInfo, error)
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

// GetReview 根据评价ID获取评价
func (uc *ReviewUsecase) GetReview(ctx context.Context, reviewID int64) (*model.ReviewInfo, error) {
	uc.log.DebugContext(ctx, "[biz] GetReview", slog.Any("reviewID", reviewID))
	return uc.repo.GetReview(ctx, reviewID)
}

// AuditReview 审核评价
func (uc *ReviewUsecase) AuditReview(ctx context.Context, param *AuditParam) error {
	uc.log.DebugContext(ctx, "[biz] AuditReview", slog.Any("param", param))
	return uc.repo.AuditReview(ctx, param)
}

func (uc *ReviewUsecase) CreateReply(ctx context.Context, param *ReplyParam) (*model.ReviewReplyInfo, error) {
	uc.log.DebugContext(ctx, "[biz] CreateReply", slog.Any("param", param))
	reply := &model.ReviewReplyInfo{
		ReplyID:   snowflake.GenID(),
		ReviewID:  param.ReviewID,
		StoreID:   param.StoreID,
		Content:   param.Content,
		PicInfo:   param.PicInfo,
		VideoInfo: param.VideoInfo,
	}
	return uc.repo.SaveReply(ctx, reply)
}
