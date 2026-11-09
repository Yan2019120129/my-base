package tables

import (
	"errors"
	"fmt"
	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/auth"
	"github.com/GoAdminGroup/go-admin/modules/db"
	values "github.com/GoAdminGroup/go-admin/plugins/admin/modules/form"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/GoAdminGroup/go-admin/template/types/form"
	"gorm.io/gorm"
	"my-base/app/service"
	"my-base/app/service/dto"
	base "my-base/code/service"
	"strconv"
)

// GetSystemNotificationTable 展示通知发送记录，编辑入口仅允许人工重试。
func GetSystemNotificationTable(ctx *context.Context) table.Table {
	t := table.NewDefaultTable(ctx, table.DefaultConfigWithDriver("mysql").SetPrimaryKey("id", db.Bigint))
	user := auth.Auth(ctx)
	t.GetDetail().SetGetDataFn(systemPlanDetailData(user, true))
	info := t.GetInfo().HideFilterArea().HideNewButton().HideDeleteButton().HideExportButton().HideDetailButton()
	info.AddField("ID", "id", db.Bigint)
	options := types.FieldOptions{}
	names := map[string]string{}
	query := user.Table("system_plans").Select("id", "title").WhereRaw("deleted_at IS NULL")
	if !user.IsSuperAdmin() {
		query = query.WhereRaw("(created_by = " + strconv.FormatInt(user.Id, 10) + " OR assignee_id = " + strconv.FormatInt(user.Id, 10) + ")")
	}
	if rows, err := query.All(); err == nil {
		for _, row := range rows {
			id := fmt.Sprint(row["id"])
			label := fmt.Sprint(row["title"]) + " (#" + id + ")"
			names[id] = label
			options = append(options, types.FieldOption{Text: label, Value: id})
		}
	}
	info.AddField("关联计划", "system_plan_id", db.Bigint).FieldFilterable().FieldFilterOptions(options).FieldFilterOptionExt(map[string]interface{}{"minimumResultsForSearch": 0, "tags": false}).FieldDisplay(func(v types.FieldModel) interface{} {
		if name, ok := names[v.Value]; ok {
			return name
		}
		return "关联计划不可用"
	})
	info.AddField("标题", "subject", db.Varchar)
	info.AddField("渠道", "channel", db.Varchar)
	info.AddField("接收地址", "recipient", db.Varchar)
	info.AddField("发送状态", "status", db.Varchar).FieldFilterable()
	info.AddField("尝试次数", "attempts", db.Int)
	info.AddField("发送时间", "sent_at", db.Datetime).FieldDisplay(displaySystemPlanTime)
	info.AddField("下次重试", "next_retry_at", db.Datetime).FieldDisplay(displaySystemPlanTime)
	info.AddField("错误摘要", "last_error", db.Text)
	info.SetTable("system_notifications").SetTitle("通知记录").SetDescription("编辑入口仅用于重试失败通知")
	if !user.IsSuperAdmin() {
		id := strconv.FormatInt(user.Id, 10)
		info.WhereRaw("system_plan_id IN (SELECT id FROM system_plans WHERE deleted_at IS NULL AND (created_by = " + id + " OR assignee_id = " + id + "))")
	}
	info.WhereRaw("deleted_at IS NULL")
	info.SetDeleteFnWithDB(func(_ *gorm.DB, _ []string) error { return errors.New("通知记录不允许删除") })
	f := t.GetForm()
	f.AddField("ID", "id", db.Bigint, form.Default).FieldDisableWhenUpdate()
	f.AddField("标题", "subject", db.Varchar, form.Text).FieldDisplayButCanNotEditWhenUpdate()
	f.AddField("发送状态", "status", db.Varchar, form.Text).FieldDisplayButCanNotEditWhenUpdate()
	f.SetTable("system_notifications").SetTitle("重试通知").SetDescription("提交后重试失败通知，不修改原消息")
	f.SetInsertFnWithDB(func(_ *gorm.DB, _ values.Values) error { return errors.New("通知只能由到期任务生成") })
	f.SetUpdateFnWithDB(func(db *gorm.DB, v values.Values) error {
		id := v.GetIntDefault("id", 0)
		if id <= 0 {
			return errors.New("通知标识无效")
		}
		s := service.SystemNotification{Service: base.Service{Orm: db}}
		return s.Retry(&dto.SystemNotificationRetryInput{ID: uint(id), UserID: user.Id, IsSuperAdmin: user.IsSuperAdmin()})
	})
	return t
}
