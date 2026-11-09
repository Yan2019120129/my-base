package tables

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"my-base/configs"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/auth"
	adminConfig "github.com/GoAdminGroup/go-admin/modules/config"
	"github.com/GoAdminGroup/go-admin/modules/db"
	values "github.com/GoAdminGroup/go-admin/plugins/admin/modules/form"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/GoAdminGroup/go-admin/template/types/form"
	"gorm.io/gorm"
	"my-base/app/service"
	"my-base/app/service/dto"
	base "my-base/code/service"
)

// GetSystemPlanTable 提供计划管理，所有写操作经过业务服务。
func GetSystemPlanTable(ctx *context.Context) table.Table {
	t := table.NewDefaultTable(ctx, table.DefaultConfigWithDriver("mysql").SetPrimaryKey("id", db.Bigint))
	user := auth.Auth(ctx)
	t.GetDetail().SetGetDataFn(systemPlanDetailData(user, false))
	info := t.GetInfo().HideFilterArea().HideDeleteButton().HideExportButton().HideDetailButton()
	info.AddField("ID", "id", db.Bigint).FieldFilterable()
	info.AddField("标题", "title", db.Varchar).FieldFilterable()
	userNames := map[string]string{}
	if rows, err := user.Table(adminConfig.GetAuthUserTable()).Select("id", "username").All(); err == nil {
		for _, row := range rows {
			userNames[fmt.Sprint(row["id"])] = fmt.Sprint(row["username"])
		}
	}
	displayUser := func(v types.FieldModel) interface{} {
		if name, ok := userNames[v.Value]; ok {
			return name
		}
		return "用户不可用"
	}
	info.AddField("创建人", "created_by", db.Bigint).FieldDisplay(displayUser)
	info.AddField("执行用户", "assignee_id", db.Bigint).FieldDisplay(displayUser)
	info.AddField("接收邮箱", "recipient", db.Varchar)
	info.AddField("执行时间（上海）", "execute_at", db.Datetime).FieldDisplay(displaySystemPlanTime)
	info.AddField("状态", "status", db.Varchar).FieldDisplay(func(v types.FieldModel) interface{} {
		switch v.Value {
		case "pending":
			return "待执行"
		case "completed":
			return "已完成"
		case "cancelled":
			return "已取消"
		}
		return v.Value
	})
	info.AddColumn("是否逾期", func(v types.FieldModel) interface{} {
		if fmtStatus, ok := v.Row["status"]; ok && fmtStatus == "pending" {
			for _, layout := range []string{time.RFC3339Nano, time.DateTime} {
				if due, err := time.Parse(layout, fmt.Sprint(v.Row["execute_at"])); err == nil && due.Before(time.Now()) {
					return "已逾期"
				}
			}
		}
		return "-"
	})
	info.AddField("通知生成时间", "notification_created_at", db.Datetime).FieldDisplay(displaySystemPlanTime)
	info.SetTable("system_plans").SetTitle("计划管理").SetDescription("单次执行计划，编辑中选择完成或取消")
	if !user.IsSuperAdmin() {
		info.WhereRaw("(created_by = " + strconv.FormatInt(user.Id, 10) + " OR assignee_id = " + strconv.FormatInt(user.Id, 10) + ")")
	}
	info.WhereRaw("deleted_at IS NULL")
	info.SetDeleteFnWithDB(func(_ *gorm.DB, _ []string) error { return errors.New("请使用取消操作，禁止删除计划") })
	f := t.GetForm()
	f.AddField("ID", "id", db.Bigint, form.Default).FieldDisableWhenCreate().FieldDisableWhenUpdate()
	f.AddField("标题", "title", db.Varchar, form.Text)
	f.AddField("内容", "content", db.Text, form.TextArea)
	if user.IsSuperAdmin() {
		options := types.FieldOptions{}
		if rows, err := user.Table(adminConfig.GetAuthUserTable()).Select("id", "username").All(); err == nil {
			for _, row := range rows {
				options = append(options, types.FieldOption{Text: fmt.Sprint(row["username"]), Value: fmt.Sprint(row["id"])})
			}
		}
		f.AddField("执行用户", "assignee_id", db.Bigint, form.SelectSingle).FieldOptions(options).FieldOptionExt(map[string]interface{}{"minimumResultsForSearch": 0, "tags": false}).FieldDefault(strconv.FormatInt(user.Id, 10))
	}
	domains, _ := json.Marshal(configs.GetAllowedRecipientDomains())
	f.AddField("接收邮箱", "recipient", db.Varchar, form.Text).FieldHelpMsg("目前仅支持配置中的邮箱后缀，默认 @qq.com").FieldCustomJs(template.JS(strings.ReplaceAll(planRecipientScript, "__DOMAINS__", string(domains))))
	f.AddField("执行时间（上海）", "execute_at", db.Datetime, form.Datetime).FieldDisplay(displaySystemPlanTime)
	f.AddField("操作", "action", db.Varchar, form.SelectSingle).FieldOptions(types.FieldOptions{{Text: "保存修改", Value: "save"}, {Text: "确认完成", Value: "completed"}, {Text: "取消计划", Value: "cancelled"}}).FieldDefault("save").FieldDisableWhenCreate()
	f.SetTable("system_plans").SetTitle("计划管理").SetDescription("邮箱由计划指定；执行时间使用 Asia/Shanghai")
	f.SetInsertFnWithDB(func(db *gorm.DB, v values.Values) error {
		in := dto.SystemPlanInput{UserID: user.Id, IsSuperAdmin: user.IsSuperAdmin(), Title: v.Get("title"), Content: v.Get("content"), Recipient: v.Get("recipient"), AssigneeID: int64(v.GetIntDefault("assignee_id", int(user.Id)))}
		due, err := parseSystemPlanTime(v.Get("execute_at"))
		if err != nil {
			return err
		}
		in.ExecuteAt = due
		s := service.SystemPlan{Service: base.Service{Orm: db}}
		return s.Save(&in)
	})
	f.SetUpdateFnWithDB(func(db *gorm.DB, v values.Values) error {
		id := v.GetIntDefault("id", 0)
		if id <= 0 {
			return errors.New("计划标识无效")
		}
		in := dto.SystemPlanInput{ID: uint(id), UserID: user.Id, IsSuperAdmin: user.IsSuperAdmin(), Action: v.Get("action"), Title: v.Get("title"), Content: v.Get("content"), Recipient: v.Get("recipient"), AssigneeID: int64(v.GetIntDefault("assignee_id", int(user.Id)))}
		s := service.SystemPlan{Service: base.Service{Orm: db}}
		if in.Action == "completed" || in.Action == "cancelled" {
			return s.Transition(&in)
		}
		if in.Action != "save" {
			return errors.New("无效的操作")
		}
		due, err := parseSystemPlanTime(v.Get("execute_at"))
		if err != nil {
			return err
		}
		in.ExecuteAt = due
		return s.Save(&in)
	})
	return t
}

// parseSystemPlanTime 将后台上海时间或带时区时间转换为 UTC。
func parseSystemPlanTime(value string) (time.Time, error) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Time{}, err
	}
	for _, layout := range []string{time.RFC3339Nano, time.DateTime, "2006-01-02 15:04"} {
		if value, err := time.ParseInLocation(layout, strings.TrimSpace(value), location); err == nil {
			return value.UTC(), nil
		}
	}
	return time.Time{}, errors.New("执行时间格式不正确")
}

// displaySystemPlanTime 按上海时区展示数据库时间。
func displaySystemPlanTime(v types.FieldModel) interface{} {
	if v.Value == "" {
		return "-"
	}
	// 无时区的数据库 DATETIME 按 UTC 解释。
	var parsed time.Time
	var err error
	for _, layout := range []string{time.RFC3339Nano, time.DateTime} {
		parsed, err = time.Parse(layout, v.Value)
		if err == nil {
			break
		}
	}
	if err != nil {
		return v.Value
	}
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return v.Value
	}
	return parsed.In(location).Format(time.DateTime)
}

// planRecipientScript 将邮箱账号与固定后缀组合，状态操作不受邮箱校验影响。
const planRecipientScript = `
(function(){
 var original=document.querySelector('input[name="recipient"]'); if(!original || original.dataset.enhanced) return;
 original.dataset.enhanced='1'; original.type='hidden';
 var domains=__DOMAINS__, account=document.createElement('input'), suffix=document.createElement('select');
 account.className='form-control'; account.placeholder='请输入邮箱账号';
 domains.forEach(function(d){var o=document.createElement('option');o.value=d;o.textContent='@'+d;suffix.appendChild(o);});
 suffix.className='form-control'; if(domains.length===1) suffix.disabled=true;
 var wrap=document.createElement('div');wrap.style.display='flex';wrap.appendChild(account);wrap.appendChild(suffix);original.parentNode.insertBefore(wrap,original);
 var warning=document.createElement('small');warning.className='text-danger';wrap.after(warning);
 var old=original.value, index=old.lastIndexOf('@');
 if(index>=0 && domains.indexOf(old.slice(index+1).toLowerCase())>=0){ account.value=old.slice(0,index);suffix.value=old.slice(index+1).toLowerCase(); }
 else if(old){ warning.textContent='原地址：'+old+'；保存修改前请更换为支持的邮箱'; }
 function sync(){var value=account.value.trim(), at=value.lastIndexOf('@');
 if(at>=0){var d=value.slice(at+1).toLowerCase();if(domains.indexOf(d)<0){warning.textContent='不支持该邮箱后缀';return false;}suffix.value=d;value=value.slice(0,at);account.value=value;}
 if(!value || !/^[^\s@]+$/.test(value)){warning.textContent='请输入有效邮箱账号';return false;}
 original.value=value+'@'+suffix.value;warning.textContent='';return true;}
 account.addEventListener('input',sync);suffix.addEventListener('change',sync);
 original.form.addEventListener('submit',function(event){var action=this.querySelector('[name="action"]');if(action && (action.value==='completed'||action.value==='cancelled'))return;if(!sync()){event.preventDefault();event.stopImmediatePropagation();account.focus();}},true);
})();`
