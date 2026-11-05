package pages

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"my-base/app/service"
	"my-base/app/service/dto"
	"my-base/code"

	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/auth"
	"github.com/GoAdminGroup/go-admin/modules/config"
	"github.com/GoAdminGroup/go-admin/modules/db"
	"github.com/GoAdminGroup/go-admin/modules/logger"
	"github.com/GoAdminGroup/go-admin/template/chartjs"
	"github.com/GoAdminGroup/go-admin/template/types"
)

// dashboardHTML 将仪表盘样式和模板随程序编译，避免依赖运行目录。
//
//go:embed dashboard.html
var dashboardHTML string

// dashboardPage 保存展示状态，链接均来自后台配置而非用户输入。
type dashboardPage struct {
	Overview  *dto.DashboardOverview
	Query     dto.DashboardQuery
	HomeURL   string
	FilesURL  string
	SharesURL string
	Error     string
	Chart     template.HTML
}

// GetDashBoard 复用后台连接池，按当前用户权限加载并渲染工作台。
func GetDashBoard(ctx *context.Context, conn db.Connection) (types.Panel, error) {
	user := auth.Auth(ctx)
	days := 7
	if ctx.Query("days") == "30" {
		days = 30
	}
	page := dashboardPage{
		Query:   dto.DashboardQuery{Days: days, IsSuperAdmin: user.IsSuperAdmin()},
		HomeURL: config.Url("/"), FilesURL: config.Url("/info/system_files"), SharesURL: config.Url("/info/system_file_shares"),
	}
	page.Query.CanViewFiles = auth.CheckPermissions(user, page.FilesURL, http.MethodGet, nil)
	page.Query.CanViewShares = page.Query.CanViewFiles && auth.CheckPermissions(user, page.SharesURL, http.MethodGet, nil)
	// 没有文件访问权限时仍展示说明，但不执行数据库统计。
	s := service.Dashboard{}
	var err error
	if page.Query.CanViewFiles {
		s.Orm, err = conn.GetGorm(code.DefaultGoAdminConnectionName)
	}
	if err == nil {
		if s.Orm != nil {
			s.Orm = s.Orm.WithContext(ctx.Request.Context())
		}
		page.Overview, err = s.GetOverview(&page.Query)
	}
	if err != nil {
		logger.Error("仪表盘统计查询失败: ", err)
		page.Error = "统计数据暂时无法加载，请稍后重试。"
	} else if page.Query.CanViewFiles {
		labels := make([]string, 0, days)
		values := make([]float64, 0, days)
		for _, point := range page.Overview.Trend {
			labels = append(labels, point.Date)
			values = append(values, float64(point.Count))
		}
		chart := chartjs.Line().SetID("dashboard-upload-trend").SetHeight(220).SetLabels(labels).
			AddDataSet("新增文件数量").DSData(values).DSFill(false).DSBorderColor("#3975ed").DSLineTension(0.15)
		chart.SetOptionLegendDisplay(false)
		page.Chart = chart.GetContent()
	}
	content, err := renderDashboard(page)
	return types.Panel{Title: "工作台", Description: "文件与分享概览", Content: content}, err
}

// renderDashboard 使用自动转义模板渲染数据库中的文件名等非可信文本。
func renderDashboard(page dashboardPage) (template.HTML, error) {
	funcs := template.FuncMap{
		"size":  dashboardFileSize,
		"clock": func(t time.Time) string { return t.In(time.FixedZone("Asia/Shanghai", 8*60*60)).Format("01-02 15:04") },
		"percent": func(part, total int64) string {
			if total <= 0 {
				return "0"
			}
			return fmt.Sprintf("%.1f", float64(part)/float64(total)*100)
		},
	}
	tmpl, err := template.New("dashboard").Funcs(funcs).Parse(dashboardHTML)
	if err != nil {
		return "", err
	}
	var output bytes.Buffer
	if err := tmpl.Execute(&output, page); err != nil {
		return "", err
	}
	return template.HTML(output.String()), nil
}

// dashboardFileSize 将字节数转换为便于阅读的二进制容量单位。
func dashboardFileSize(value int64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	size := float64(value)
	index := 0
	for size >= 1024 && index < len(units)-1 {
		size /= 1024
		index++
	}
	if index == 0 {
		return fmt.Sprintf("%d B", value)
	}
	return fmt.Sprintf("%.1f %s", size, units[index])
}
