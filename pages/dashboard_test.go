package pages

import (
	"strings"
	"testing"
	"time"

	"my-base/app/service/dto"
)

// TestRenderDashboard 验证文件名转义、权限提示及加载失败状态。
func TestRenderDashboard(t *testing.T) {
	malicious := `<script>alert("文件名")</script>`
	page := dashboardPage{
		Query: dto.DashboardQuery{CanViewFiles: true, CanViewShares: true, Days: 7},
		Overview: &dto.DashboardOverview{Days: 7, DashboardFileMetrics: dto.DashboardFileMetrics{TotalSize: 1024}, UpdatedAt: time.Now(),
			Types:          []dto.DashboardFileType{{Name: "图片", Size: 1024, Count: 1}},
			RecentFiles:    []dto.DashboardFile{{ID: 1, OriginalName: malicious}},
			ExpiringShares: []dto.DashboardShare{{ID: 1, OriginalName: malicious}}},
		HomeURL: "/admin", FilesURL: "/admin/info/system_files", SharesURL: "/admin/info/system_file_shares",
	}
	html, err := renderDashboard(page)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(html), malicious) || !strings.Contains(string(html), "&lt;script&gt;") || strings.Contains(string(html), "ZgotmplZ") {
		t.Fatalf("模板未安全渲染: %s", html)
	}
	if !strings.Contains(string(html), "1.0 KiB") || !strings.Contains(string(html), "width:100.0%") {
		t.Fatal("容量或类型分布未正确显示")
	}
	failed, err := renderDashboard(dashboardPage{Error: "查询失败"})
	if err != nil || !strings.Contains(string(failed), "查询失败") || strings.Contains(string(failed), "文件总数") {
		t.Fatalf("加载失败不应展示零指标: %v", err)
	}
	denied, err := renderDashboard(dashboardPage{})
	if err != nil || !strings.Contains(string(denied), "暂无文件管理权限") {
		t.Fatalf("应显示权限提示: %v", err)
	}
	page.Query.CanViewShares = false
	html, err = renderDashboard(page)
	if err != nil || !strings.Contains(string(html), "暂无分享管理权限") || strings.Contains(string(html), "管理分享 →") {
		t.Fatalf("不应显示无权限的分享入口: %v", err)
	}
}
