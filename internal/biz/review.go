package biz

import (
	"context"
	"log/slog"
	v1 "review_service/api/review/v1"
	"review_service/internal/data/model"
	"review_service/pkg/snowflake"
	"strings"
	"time"
)

type ReviewRepo interface {
	SaveReview(context.Context, *model.ReviewInfo) (*model.ReviewInfo, error)
	GetReviewByOrderID(context.Context, int64) ([]*model.ReviewInfo, error)
	GetReview(context.Context, int64) (*model.ReviewInfo, error)
	AuditReview(context.Context, *AuditParam) error
	SaveReply(context.Context, *model.ReviewReplyInfo) (*model.ReviewReplyInfo, error)
	AppealReview(context.Context, *AppealParam) (*model.ReviewAppealInfo, error)
	AuditAppeal(context.Context, *AuditAppealParam) error
	ListReviewByUserID(context.Context, int64, int, int) ([]*model.ReviewInfo, error)
	ListReviewByStoreID(context.Context, int64, int, int) ([]*ReviewInfo, error)
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

// AppealReview 申诉评价
func (uc ReviewUsecase) AppealReview(ctx context.Context, param *AppealParam) (*model.ReviewAppealInfo, error) {
	uc.log.DebugContext(ctx, "[biz] AppealReview", "param", param)
	return uc.repo.AppealReview(ctx, param)
}

// AuditAppeal 审核申诉
func (uc ReviewUsecase) AuditAppeal(ctx context.Context, param *AuditAppealParam) error {
	uc.log.DebugContext(ctx, "[biz] AuditAppeal", "param", param)
	return uc.repo.AuditAppeal(ctx, param)
}

// ListReviewByUserID 根据userID分页查询评价
func (uc ReviewUsecase) ListReviewByUserID(ctx context.Context, userID int64, page, size int) ([]*model.ReviewInfo, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 50 {
		size = 10
	}
	offset := (page - 1) * size
	limit := size
	uc.log.DebugContext(ctx, "[biz] ListReviewByUserID", "userID", userID)
	return uc.repo.ListReviewByUserID(ctx, userID, offset, limit)
}

// ListReviewByStoreID 根据storeID分页查询评价
func (uc ReviewUsecase) ListReviewByStoreID(ctx context.Context, storeID int64, page, size int) ([]*ReviewInfo, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 50 {
		size = 10
	}
	offset := (page - 1) * size
	limit := size
	uc.log.DebugContext(ctx, "[biz] ListReviewByStoreID", "storeID", storeID)
	return uc.repo.ListReviewByStoreID(ctx, storeID, offset, limit)
}

type ReviewInfo struct {
	*model.ReviewInfo
	CreateAt     time.Time `json:"create_at"`
	UpdateAt     time.Time `json:"update_at"`
	Anonymous    int32     `json:"anonymous,string"`
	Score        int32     `json:"score,string"`
	ServiceScore int32     `json:"service_score,string"`
	ExpressScore int32     `json:"express_score,string"`
	HasMedia     int32     `json:"has_media,string"`
	Status       int32     `json:"status,string"`
	IsDefault    int32     `json:"is_default,string"`
	HasReply     int32     `json:"has_reply,string"`
	ID           int64     `json:"id,string"`
	Version      int32     `json:"version,string"`
	ReviewID     int64     `json:"review_id,string"`
	OrderID      int64     `json:"order_id,string"`
	SkuID        int64     `json:"sku_id,string"`
	SpuID        int64     `json:"spu_id,string"`
	StoreID      int64     `json:"store_id,string"`
	UserID       int64     `json:"user_id,string"`
}

type MyTime time.Time

func (t *MyTime) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	tmp, err := time.Parse(time.DateTime, s)
	if err != nil {
		return err
	}
	*t = MyTime(tmp)
	return nil
}
