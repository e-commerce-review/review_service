package data

import (
	"context"
	"errors"
	"log/slog"
	"review_service/internal/biz"
	"review_service/internal/data/model"
	"review_service/internal/data/query"
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

func (r *reviewRepo) GetReviewByOrderID(ctx context.Context, orderID int64) ([]*model.ReviewInfo, error) {
	return r.data.query.ReviewInfo.WithContext(ctx).Where(r.data.query.ReviewInfo.OrderID.Eq(orderID)).Find()
}

func (r *reviewRepo) GetReview(ctx context.Context, reviewID int64) (*model.ReviewInfo, error) {
	return r.data.query.ReviewInfo.
		WithContext(ctx).
		Where(r.data.query.ReviewInfo.ReviewID.Eq(reviewID)).
		First()
}

func (r *reviewRepo) AuditReview(ctx context.Context, param *biz.AuditParam) error {
	return nil
}

func (r *reviewRepo) SaveReply(ctx context.Context, reply *model.ReviewReplyInfo) (*model.ReviewReplyInfo, error) {
	review, err := r.data.query.ReviewInfo.WithContext(ctx).
		Where(r.data.query.ReviewInfo.ReviewID.Eq(reply.ReviewID)).First()
	if err != nil {
		return nil, err
	}
	if review.HasReply == 1 {
		return nil, errors.New("this review already replied")
	}
	if review.StoreID != reply.StoreID {
		return nil, errors.New("horizontal overstepping")
	}
	err = r.data.query.Transaction(func(tx *query.Query) error {
		if err := tx.ReviewReplyInfo.WithContext(ctx).Save(reply); err != nil {
			r.log.ErrorContext(ctx, "saveReply create reply failed", "err", err)
		}
		if _, err := tx.ReviewInfo.WithContext(ctx).
			Where(tx.ReviewInfo.ReviewID.Eq(reply.ReviewID)).
			Update(tx.ReviewInfo.HasReply, 1); err != nil {
			r.log.ErrorContext(ctx, "saveReply update reply failed", "err", err)
		}
		return nil
	})
	return reply, nil
}

func (r *reviewRepo) GetReviewReply(ctx context.Context, reviewID int64) (*model.ReviewReplyInfo, error) {
	return r.data.query.ReviewReplyInfo.WithContext(ctx).
		Where(r.data.query.ReviewAppealInfo.ReviewID.Eq(reviewID)).First()
}
