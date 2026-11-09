package tables

import (
	adminModels "github.com/GoAdminGroup/go-admin/plugins/admin/models"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/parameter"
	"github.com/GoAdminGroup/go-admin/template/types"
	"my-base/code"
	"strconv"
	"time"
)

// systemPlanDetailData 限制后台编辑页的单条读取，避免绕过列表的数据权限。
func systemPlanDetailData(user adminModels.UserModel, notifications bool) types.GetDataFn {
	return func(param parameter.Parameters) ([]map[string]interface{}, int) {
		id, err := strconv.ParseUint(param.PK(), 10, 64)
		if err != nil || id == 0 || user.Id <= 0 || user.Conn == nil {
			return nil, 0
		}
		orm, err := user.Conn.GetGorm(code.DefaultGoAdminConnectionName)
		if err != nil {
			return nil, 0
		}
		query := orm.Table("system_plans").Where("system_plans.id = ? AND system_plans.deleted_at IS NULL", id)
		if notifications {
			query = orm.Table("system_notifications").Select("system_notifications.*").Joins("JOIN system_plans ON system_plans.id = system_notifications.system_plan_id").Where("system_notifications.id = ? AND system_notifications.deleted_at IS NULL AND system_plans.deleted_at IS NULL", id)
		}
		if !user.IsSuperAdmin() {
			query = query.Where("system_plans.created_by = ? OR system_plans.assignee_id = ?", user.Id, user.Id)
		}
		var rows []map[string]interface{}
		if err := query.Find(&rows).Error; err != nil {
			return nil, 0
		}
		// 自定义数据源按 JSON 类型读取，统一转换驱动返回的时间与字节。
		for _, row := range rows {
			for key, value := range row {
				switch v := value.(type) {
				case time.Time:
					row[key] = v.UTC().Format(time.RFC3339Nano)
				case []byte:
					row[key] = string(v)
				}
			}
		}
		return rows, len(rows)
	}
}
