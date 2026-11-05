package service

import (
	"testing"
	"time"

	"my-base/app/models"
	"my-base/app/service/dto"
	codeService "my-base/code/service"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestDashboardOverview 验证权限、软删除、分享有效性和上海日期分桶。
func TestDashboardOverview(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := database.AutoMigrate(&models.SystemFile{}, &models.SystemFileShare{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().In(time.FixedZone("Asia/Shanghai", 8*60*60))
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	records := []models.SystemFile{
		{Model: gorm.Model{CreatedAt: today}, OriginalName: "公开图片", FileName: "a", StoragePath: "a", Status: 1, IsPublic: true, MimeType: "image/png", FileSize: 100},
		{Model: gorm.Model{CreatedAt: today.Add(-time.Second)}, OriginalName: "私有文档", FileName: "b", StoragePath: "b", Status: 1, IsPublic: false, MimeType: "application/pdf", FileSize: 200},
		{Model: gorm.Model{CreatedAt: today.Add(-time.Second)}, OriginalName: "昨天文档", FileName: "c", StoragePath: "c", Status: 1, IsPublic: true, MimeType: "application/pdf", FileSize: 300},
		{OriginalName: "禁用文件", FileName: "d", StoragePath: "d", Status: 2, IsPublic: true, FileSize: 400},
		{OriginalName: "已删除文件", FileName: "e", StoragePath: "e", Status: 1, IsPublic: true, FileSize: 500},
	}
	for i := range records {
		if err := database.Create(&records[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	// 模型默认公开，使用显式更新构造私有文件。
	if err := database.Model(&records[1]).Update("is_public", false).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Delete(&records[4]).Error; err != nil {
		t.Fatal(err)
	}
	revoked := now.Add(-time.Minute)
	shares := []models.SystemFileShare{
		{SystemFileID: records[0].ID, Token: "active", ExpiresAt: now.Add(time.Hour)},
		{SystemFileID: records[0].ID, Token: "later", ExpiresAt: now.Add(48 * time.Hour)},
		{SystemFileID: records[1].ID, Token: "private", ExpiresAt: now.Add(time.Hour)},
		{SystemFileID: records[0].ID, Token: "expired", ExpiresAt: now.Add(-time.Hour)},
		{SystemFileID: records[0].ID, Token: "revoked", ExpiresAt: now.Add(time.Hour), RevokedAt: &revoked},
		{SystemFileID: records[3].ID, Token: "disabled", ExpiresAt: now.Add(time.Hour)},
		{SystemFileID: records[4].ID, Token: "deleted-file", ExpiresAt: now.Add(time.Hour)},
		{SystemFileID: 99999, Token: "missing-file", ExpiresAt: now.Add(time.Hour)},
		{SystemFileID: records[0].ID, Token: "deleted-share", ExpiresAt: now.Add(time.Hour)},
	}
	for i := range shares {
		if err := database.Create(&shares[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := database.Delete(&shares[8]).Error; err != nil {
		t.Fatal(err)
	}
	s := Dashboard{Service: codeService.Service{Orm: database}}
	for _, super := range []bool{false, true} {
		data, err := s.GetOverview(&dto.DashboardQuery{Days: 30, IsSuperAdmin: super, CanViewFiles: true, CanViewShares: true})
		if err != nil {
			t.Fatal(err)
		}
		expectedCount, expectedSize, expectedShares, expectedExpiring := int64(2), int64(400), int64(2), int64(1)
		if super {
			expectedCount = 3
			expectedSize = 600
			expectedShares = 3
			expectedExpiring = 2
		}
		if data.FileCount != expectedCount || data.TotalSize != expectedSize || data.TodayCount != 1 || data.TodaySize != 100 {
			t.Fatalf("文件统计不正确，超级管理员=%v: %+v", super, data)
		}
		if data.ActiveShares != expectedShares || data.ExpiringCount != expectedExpiring || len(data.ExpiringShares) != int(expectedExpiring) {
			t.Fatalf("分享统计不正确: %+v", data)
		}
		if len(data.Trend) != 30 || data.Trend[29].Count != 1 || data.Trend[28].Count != expectedCount-1 || data.Trend[0].Count != 0 {
			t.Fatalf("日期分桶错误: %+v", data.Trend)
		}
		if len(data.RecentFiles) != int(expectedCount) || len(data.Types) != 2 {
			t.Fatalf("文件列表或分类错误: %+v", data)
		}
	}
	// 无分享权限时不读取分享表；无文件权限时无需数据库连接。
	if err := database.Migrator().DropTable(&models.SystemFileShare{}); err != nil {
		t.Fatal(err)
	}
	data, err := s.GetOverview(&dto.DashboardQuery{Days: 99, CanViewFiles: true})
	if err != nil || data.Days != 7 || data.ActiveShares != 0 || len(data.ExpiringShares) != 0 {
		t.Fatalf("分享权限或默认周期错误: %+v, %v", data, err)
	}
	empty := Dashboard{}
	data, err = empty.GetOverview(&dto.DashboardQuery{CanViewShares: true})
	if err != nil || data.FileCount != 0 || len(data.RecentFiles) != 0 {
		t.Fatalf("未授权查询不应访问数据库: %+v, %v", data, err)
	}
	if _, err := empty.GetOverview(nil); err == nil {
		t.Fatal("应拒绝空查询参数")
	}
	if _, err := empty.GetOverview(&dto.DashboardQuery{CanViewFiles: true}); err == nil {
		t.Fatal("应返回连接缺失错误")
	}
	if err := database.Migrator().DropTable(&models.SystemFile{}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetOverview(&dto.DashboardQuery{CanViewFiles: true}); err == nil {
		t.Fatal("数据库失败不得返回零统计")
	}
}
