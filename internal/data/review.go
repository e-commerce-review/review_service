package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"review_service/internal/biz"
	"review_service/internal/data/model"
	"review_service/internal/data/query"
	"review_service/pkg/snowflake"

	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
		Where(r.data.query.ReviewReplyInfo.ReviewID.Eq(reviewID)).First()
}

func (r *reviewRepo) AppealReview(ctx context.Context, param *biz.AppealParam) (*model.ReviewAppealInfo, error) {
	// 先查询有没有申诉
	ret, err := r.data.query.ReviewAppealInfo.
		WithContext(ctx).
		Where(
			query.ReviewAppealInfo.ReviewID.Eq(param.ReviewID),
			query.ReviewAppealInfo.StoreID.Eq(param.StoreID),
		).First()
	r.log.DebugContext(ctx, "[data] AppealReview query", "ret", ret, "err", err)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		// 其他查询错误
		return nil, err
	}
	if err == nil && ret.Status > 10 {
		return nil, errors.New("该评价已有审核过的申诉记录")
	}
	// 查询不到审核过的申诉记录
	// 1. 有申诉记录但是处于待审核状态，需要更新
	// if ret != nil{
	// 	// update
	// }else{
	// 	// insert
	// }
	// 2. 没有申诉记录，需要创建
	appeal := &model.ReviewAppealInfo{
		ReviewID:  param.ReviewID,
		StoreID:   param.StoreID,
		Status:    10,
		Reason:    param.Reason,
		Content:   param.Content,
		PicInfo:   param.PicInfo,
		VideoInfo: param.VideoInfo,
	}
	if ret != nil {
		appeal.AppealID = ret.AppealID
	} else {
		appeal.AppealID = snowflake.GenID()
	}
	err = r.data.query.ReviewAppealInfo.
		WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "review_id"}, // ON DUPLICATE KEY
			},
			DoUpdates: clause.Assignments(map[string]interface{}{ // UPDATE
				"status":     appeal.Status,
				"content":    appeal.Content,
				"reason":     appeal.Reason,
				"pic_info":   appeal.PicInfo,
				"video_info": appeal.VideoInfo,
			}),
		}).
		Create(appeal) // INSERT
	r.log.DebugContext(ctx, "[data] AppealReview", "err", err)
	return appeal, err
}

func (r *reviewRepo) AuditAppeal(ctx context.Context, param *biz.AuditAppealParam) error {
	err := r.data.query.Transaction(func(tx *query.Query) error {
		// 申诉表
		if _, err := tx.ReviewAppealInfo.
			WithContext(ctx).
			Where(r.data.query.ReviewAppealInfo.AppealID.Eq(param.AppealID)).
			Updates(map[string]interface{}{
				"status":  param.Status,
				"op_user": param.OpUser,
			}); err != nil {
			return err
		}
		// 评价表
		if param.Status == 20 { // 申诉通过则需要隐藏评价
			if _, err := tx.ReviewInfo.WithContext(ctx).
				Where(tx.ReviewInfo.ReviewID.Eq(param.ReviewID)).
				Update(tx.ReviewInfo.Status, 40); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func (r *reviewRepo) AuditReview(ctx context.Context, param *biz.AuditParam) error {
	_, err := r.data.query.ReviewInfo.
		WithContext(ctx).
		Where(r.data.query.ReviewInfo.ReviewID.Eq(param.ReviewID)).
		Updates(map[string]interface{}{
			"status":     param.Status,
			"op_user":    param.OpUser,
			"op_reason":  param.OpReason,
			"op_remarks": param.OpRemarks,
		})
	return err
}

func (r *reviewRepo) ListReviewByUserID(ctx context.Context, userID int64, offset, limit int) ([]*model.ReviewInfo, error) {
	return r.data.query.ReviewInfo.
		WithContext(ctx).
		Where(r.data.query.ReviewInfo.UserID.Eq(userID)).
		Order(r.data.query.ReviewInfo.ID.Desc()).
		Limit(limit).
		Offset(offset).
		Find()
}

// ListReviewByStoreID 根据storeID 分页查询评价
func (r *reviewRepo) ListReviewByStoreID(ctx context.Context, storeID int64, offset, limit int) ([]*biz.ReviewInfo, error) {
	// 去ES里面查询评价
	resp, err := r.data.es.Search().
		Index("review").
		From(offset).
		Size(limit).
		Query(&types.Query{
			Bool: &types.BoolQuery{
				Filter: []types.Query{
					{
						Term: map[string]types.TermQuery{
							"store_id": {Value: storeID},
						},
					},
				},
			},
		}).
		Do(ctx)
	fmt.Printf("--> es search: %v %v\n", resp, err)
	if err != nil {
		return nil, err
	}
	fmt.Printf("es result total:%v\n", resp.Hits.Total.Value)
	list := make([]*biz.ReviewInfo, 0, resp.Hits.Total.Value)

	for _, hit := range resp.Hits.Hits {
		tmp := &biz.ReviewInfo{}
		if err := json.Unmarshal(hit.Source_, tmp); err != nil {
			r.log.ErrorContext(ctx, "json.unmarshal failed", "err", err)
			continue
		}
		list = append(list, tmp)
	}

	return list, nil
}
