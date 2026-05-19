//go:build ignore
// +build ignore

package data

import (
	"context"
	"database/sql"
	v2 "odin/api/loli/service/v2"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	GetAchieveInfoSQL       = `SELECT JSON_EXTRACT(achieve_info, '$') FROM users WHERE id = ?`
	GetAchieveSeriesByIdSQL = `SELECT JSON_EXTRACT(achieve_info, CONCAT('$.series."', ?, '"')) FROM users WHERE id = ?`
	SetAchieveSeriesByIdSQL = `UPDATE users SET achieve_info = JSON_SET(achieve_info, CONCAT('$.series."', ?, '"'), CAST(? AS JSON)) WHERE id = ?`
	UpdateAchieveSummarySQL = `UPDATE users SET achieve_info = JSON_SET(achieve_info, '$.summary', CAST(? AS JSON)) WHERE id = ?`
	GetAchieveSummarySQL    = `SELECT achieve_info->'$.summary' FROM users WHERE id = ?`
	AddAchieveExperienceSQL = `UPDATE users SET achieve_info = JSON_SET(achieve_info, '$.summary.experience',CAST((JSON_EXTRACT(achieve_info, '$.summary.experience') + ?) AS SIGNED)) WHERE id = ?`
)

func (r *userRepo) GetAchieveInfo(ctx context.Context, yid int64) (*v2.AchieveInfo, error) {
	gormDB := r.tryExtractTx(ctx)

	var s sql.NullString
	if err := gormDB.Raw(GetAchieveInfoSQL, yid).Row().Scan(&s); err != nil {
		return nil, errors.Wrapf(err, "gorm GetAllAchievesSQL/Scan error")
	}
	achieves := &v2.AchieveInfo{}
	if err := protojson.Unmarshal([]byte(s.String), achieves); err != nil {
		return nil, errors.Wrapf(err, "protojson unmarshal Achieves error")
	}
	if err := r.setAchieveInfoCache(ctx, yid, achieves); err != nil && !errors.Is(err, ErrCacheDisabled) {
		r.log.Warnf("setAchieveInfoCache error=%v", err)
	}
	return achieves, nil
}

func (r *userRepo) GetAchieveSeriesById(ctx context.Context, yid int64, aid int32) (*v2.AchieveInfo_Series, error) {
	gormDB := r.tryExtractTx(ctx)

	var (
		achieve       *v2.AchieveInfo_Series
		isCacheFailed bool
		err           error
	)

	if achieve, err = r.getAchieveTopicByIdCache(ctx, yid, aid); err != nil {
		isCacheFailed = true
		if !errors.Is(err, ErrCacheDisabled) {
			r.log.Warnf("getAchieveTopicByIdCache error=%v", err)
		}
	}

	if isCacheFailed {
		var s sql.NullString
		if err := gormDB.Raw(GetAchieveSeriesByIdSQL, aid, yid).Row().Scan(&s); err != nil {
			return nil, errors.Wrapf(err, "gorm GetAchieveSeriesByIdSQL/Scan error")
		}
		if !s.Valid {
			return nil, nil
		}
		achieve = &v2.AchieveInfo_Series{}
		if err := protojson.Unmarshal([]byte(s.String), achieve); err != nil {
			return nil, errors.Wrapf(err, "protojson umahrshal AchieveTopic error")
		}
	}

	return achieve, nil
}

func (r *userRepo) SetAchieveSeriesById(ctx context.Context, yid int64, aid int32, achieve *v2.AchieveInfo_Series) error {
	gormDB := r.tryExtractTx(ctx)

	jbAchieve, err := pMarshaler.Marshal(achieve)
	if err != nil {
		return errors.Wrapf(err, "pMarshaler marshal AchieveTopic error")
	}

	if err := gormDB.Exec(SetAchieveSeriesByIdSQL, aid, string(jbAchieve), yid).Error; err != nil {
		return errors.Wrapf(err, "gorm SetAchieveSeriesById err")
	}

	if err := r.setAchieveTopicByIdCache(ctx, yid, aid, string(jbAchieve)); err != nil && !errors.Is(err, ErrCacheDisabled) {
		r.log.Warnf("getAchieveTopicByIdCache error=%v", err)
	}

	return nil
}

func (r *userRepo) GetAchieveSummary(ctx context.Context, yid int64) (*v2.AchieveInfo_Summary, error) {
	gormDB := r.tryExtractTx(ctx)

	var s sql.NullString
	if err := gormDB.Raw(GetAchieveSummarySQL, yid).Row().Scan(&s); err != nil {
		return nil, errors.Wrapf(err, "gorm GetAchieveSummarySQL/Scan error")
	}
	achieveProgress := &v2.AchieveInfo_Summary{}
	if err := protojson.Unmarshal([]byte(s.String), achieveProgress); err != nil {
		return nil, errors.Wrapf(err, "protojson.Unmarshal AchieveSummary error")
	}
	return achieveProgress, nil
}

func (r *userRepo) SetAchieveSummary(ctx context.Context, yid int64, achieveProgress *v2.AchieveInfo_Summary) error {
	gormDB := r.tryExtractTx(ctx)

	jbAchieveSummary, err := pMarshaler.Marshal(achieveProgress)
	if err != nil {
		return errors.Wrapf(err, "pMarshaler.Marshal AchieveSummary error")
	}
	err = gormDB.Exec(UpdateAchieveSummarySQL, string(jbAchieveSummary), yid).Error
	return errors.Wrapf(err, "gorm SetAchieveSeriesByIdSQL err")
}

func (r *userRepo) AddAchieveExperience(ctx context.Context, yid int64, exp int32) error {
	gormDB := r.tryExtractTx(ctx)

	if err := gormDB.Exec(AddAchieveExperienceSQL, exp, yid).Error; err != nil {
		return errors.Wrapf(err, "AddAchieveExperienceSQL error")
	}
	return nil
}
