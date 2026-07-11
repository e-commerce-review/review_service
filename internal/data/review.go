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
	"strconv"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
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
			return err
		}
		if _, err := tx.ReviewInfo.WithContext(ctx).
			Where(tx.ReviewInfo.ReviewID.Eq(reply.ReviewID)).
			Update(tx.ReviewInfo.HasReply, 1); err != nil {
			r.log.ErrorContext(ctx, "saveReply update reply failed", "err", err)
			return err
		}
		return nil
	})
	return reply, err
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
		_, err = r.data.query.ReviewAppealInfo.
			WithContext(ctx).
			Where(r.data.query.ReviewAppealInfo.AppealID.Eq(ret.AppealID)).
			Updates(map[string]interface{}{
				"status":     appeal.Status,
				"content":    appeal.Content,
				"reason":     appeal.Reason,
				"pic_info":   appeal.PicInfo,
				"video_info": appeal.VideoInfo,
			})
	} else {
		appeal.AppealID = snowflake.GenID()
		err = r.data.query.ReviewAppealInfo.
			WithContext(ctx).
			Create(appeal)
	}
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
				"status":     param.Status,
				"op_user":    param.OpUser,
				"op_remarks": param.OpRemarks,
			}); err != nil {
			return err
		}
		// 评价表
		if param.Status == 20 { // 申诉通过则需要隐藏评价
			if _, err := tx.ReviewInfo.WithContext(ctx).
				Where(tx.ReviewInfo.ReviewID.Eq(param.ReviewID)).
				Updates(map[string]interface{}{
					"status":     40,
					"op_user":    param.OpUser,
					"op_reason":  param.OpReason,
					"op_remarks": param.OpRemarks,
				}); err != nil {
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
	return r.getData(ctx, storeID, offset, limit)
}

var g singleflight.Group

func (r *reviewRepo) getData(ctx context.Context, storeID int64, offset, limit int) ([]*biz.ReviewInfo, error) {
	key := fmt.Sprintf("review_info:%d:%d:%d", storeID, offset, limit)
	v, err, shared := g.Do(key, func() (any, error) {
		data, err := r.getDataFromCache(ctx, key)
		r.log.DebugContext(ctx, "r.getDataFromCache", "data", data, "err", err)
		if err == nil {
			return data, nil
		}
		if errors.Is(err, redis.Nil) {
			data, err := r.getDataFromES(ctx, key)
			if err == nil {
				return data, r.setCache(ctx, key, data)
			}
			return nil, err
		}
		return nil, err
	})
	r.log.DebugContext(ctx, "single flight", "v", v, "err", err, "shared", shared)
	if err != nil {
		return nil, err
	}
	hm := new(types.HitsMetadata)
	if err := json.Unmarshal(v.([]byte), hm); err != nil {
		return nil, err
	}
	list := make([]*biz.ReviewInfo, 0, hm.Total.Value)
	for _, hit := range hm.Hits {
		tmp := &biz.ReviewInfo{}
		if err := json.Unmarshal(hit.Source_, tmp); err != nil {
			r.log.ErrorContext(ctx, "json.unmarshal failed", "err", err)
			continue
		}
		list = append(list, tmp)

	}
	return list, nil
}

func (r *reviewRepo) getDataFromCache(ctx context.Context, key string) ([]byte, error) {
	r.log.DebugContext(ctx, "getDataFromCache", "key", key)
	return r.data.rdb.Get(ctx, key).Bytes()
}

func (r *reviewRepo) getDataFromES(ctx context.Context, key string) ([]byte, error) {
	values := strings.Split(key, ":")
	if len(values) < 4 {
		return nil, errors.New("invalid key")
	}
	index, storeID, offsetStr, limitStr := values[0], values[1], values[2], values[3]
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return nil, errors.New("invalid offset")
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return nil, errors.New("invalid limit")
	}
	resp, err := r.data.es.Search().
		Index(index).
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
	if err != nil {
		return nil, err
	}
	return json.Marshal(resp.Hits)
}

func (r *reviewRepo) setCache(ctx context.Context, key string, data []byte) error {
	return r.data.rdb.Set(ctx, key, data, time.Second*10).Err()
}
