package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"my-base/app/models"
	"my-base/app/service/dto"
	"my-base/code/service"

	"gorm.io/gorm"
)

// Dashboard 使用现有数据库连接提供仪表盘聚合查询。
type Dashboard struct{ service.Service }

// GetOverview 按列表权限统计可见正常文件，并返回有限数量的最近记录。
func (e *Dashboard) GetOverview(input *dto.DashboardQuery) (*dto.DashboardOverview, error) {
	if input == nil {
		return nil, errors.New("仪表盘查询参数不能为空")
	}
	days := input.Days
	if days != 30 {
		days = 7
	}
	zone := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Now().In(zone)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, zone)
	result := &dto.DashboardOverview{UpdatedAt: now, Days: days}
	for i := 0; i < days; i++ {
		result.Trend = append(result.Trend, dto.DashboardTrend{Date: today.AddDate(0, 0, i-days+1).Format("01-02")})
	}
	if !input.CanViewFiles {
		return result, nil
	}
	if e.Orm == nil {
		return nil, errors.New("数据库连接不可用")
	}
	// 每次返回独立查询，避免聚合条件影响后续查询。
	files := func() *gorm.DB {
		query := e.Orm.Model(&models.SystemFile{}).Where("status = ?", 1)
		if !input.IsSuperAdmin {
			query = query.Where("is_public = ?", true)
		}
		return query
	}
	if err := files().Select(`COUNT(*) AS file_count, COALESCE(SUM(file_size), 0) AS total_size,
 COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? THEN 1 ELSE 0 END), 0) AS today_count,
 COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? THEN file_size ELSE 0 END), 0) AS today_size`,
		today, today.AddDate(0, 0, 1), today, today.AddDate(0, 0, 1)).Scan(&result.DashboardFileMetrics).Error; err != nil {
		return nil, err
	}

	// 用明确的上海时区边界分桶，避免依赖数据库服务器的日期函数和时区。
	var buckets strings.Builder
	buckets.WriteString("CASE")
	args := make([]interface{}, 0, days*2)
	start := today.AddDate(0, 0, 1-days)
	for i := 0; i < days; i++ {
		fmt.Fprintf(&buckets, " WHEN created_at >= ? AND created_at < ? THEN %d", i)
		args = append(args, start.AddDate(0, 0, i), start.AddDate(0, 0, i+1))
	}
	buckets.WriteString(" ELSE -1 END AS day_index, COUNT(*) AS count")
	var counts []struct {
		DayIndex int
		Count    int64
	}
	if err := files().Where("created_at >= ? AND created_at < ?", start, today.AddDate(0, 0, 1)).
		Select(buckets.String(), args...).Group("day_index").Scan(&counts).Error; err != nil {
		return nil, err
	}
	for _, count := range counts {
		if count.DayIndex >= 0 && count.DayIndex < days {
			result.Trend[count.DayIndex].Count = count.Count
		}
	}

	category := `CASE WHEN mime_type LIKE 'image/%' THEN '图片' WHEN mime_type LIKE 'video/%' THEN '视频' WHEN mime_type LIKE 'audio/%' THEN '音频' WHEN mime_type LIKE 'text/%' OR mime_type = 'application/pdf' OR mime_type LIKE '%officedocument%' OR mime_type LIKE '%msword%' OR mime_type LIKE '%ms-excel%' OR mime_type LIKE '%ms-powerpoint%' THEN '文档' ELSE '其他' END`
	if err := files().Select(category + " AS name, COUNT(*) AS count, COALESCE(SUM(file_size), 0) AS size").Group(category).Order("size DESC, name ASC").Scan(&result.Types).Error; err != nil {
		return nil, err
	}
	if err := files().Select("id, original_name, mime_type, file_size, created_at").Order("created_at DESC, id DESC").Limit(6).Scan(&result.RecentFiles).Error; err != nil {
		return nil, err
	}
	if !input.CanViewShares {
		return result, nil
	}
	// 分享必须关联可见的正常文件，且未软删除、未撤销、未到期。
	shares := func() *gorm.DB {
		return e.Orm.Model(&models.SystemFileShare{}).
			Where("revoked_at IS NULL AND expires_at > ?", now).
			Where("system_file_id IN (?)", files().Select("id"))
	}
	if err := shares().Count(&result.ActiveShares).Error; err != nil {
		return nil, err
	}
	until := now.Add(24 * time.Hour)
	if err := shares().Where("expires_at <= ?", until).Count(&result.ExpiringCount).Error; err != nil {
		return nil, err
	}
	if err := shares().Select("system_file_shares.id, system_files.original_name, system_file_shares.expires_at").
		Joins("JOIN system_files ON system_files.id = system_file_shares.system_file_id").
		Where("expires_at <= ?", until).Order("expires_at ASC, system_file_shares.id ASC").Limit(6).Scan(&result.ExpiringShares).Error; err != nil {
		return nil, err
	}
	return result, nil
}
