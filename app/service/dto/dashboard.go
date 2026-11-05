package dto

import "time"

// DashboardQuery 定义仪表盘的统计周期和当前用户的数据权限。
type DashboardQuery struct {
	Days          int
	IsSuperAdmin  bool
	CanViewFiles  bool
	CanViewShares bool
}

// DashboardOverview 保存仪表盘统计与列表，不包含存储路径和分享令牌。
type DashboardOverview struct {
	DashboardFileMetrics
	UpdatedAt      time.Time
	Days           int
	ActiveShares   int64
	ExpiringCount  int64
	Trend          []DashboardTrend
	Types          []DashboardFileType
	RecentFiles    []DashboardFile
	ExpiringShares []DashboardShare
}

// DashboardFileMetrics 保存可由单次聚合查询得到的文件指标。
type DashboardFileMetrics struct {
	FileCount  int64
	TotalSize  int64
	TodayCount int64
	TodaySize  int64
}

// DashboardTrend 表示上海时区某一天的新增文件数量。
type DashboardTrend struct {
	Date  string
	Count int64
}

// DashboardFileType 表示一类文件的数量与总大小。
type DashboardFileType struct {
	Name  string
	Count int64
	Size  int64
}

// DashboardFile 保存最近上传列表需要的文件信息。
type DashboardFile struct {
	ID           uint
	OriginalName string
	MimeType     string
	FileSize     int64
	CreatedAt    time.Time
}

// DashboardShare 保存即将到期的分享信息。
type DashboardShare struct {
	ID           uint
	OriginalName string
	ExpiresAt    time.Time
}
