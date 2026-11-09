package tables

import (
	adminDB "github.com/GoAdminGroup/go-admin/modules/db"
	adminModels "github.com/GoAdminGroup/go-admin/plugins/admin/models"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/parameter"
	"github.com/GoAdminGroup/go-admin/template/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"my-base/app/models"
	"strconv"
	"testing"
	"time"
)

// TestSystemPlanDetailScope 验证编辑页不能通过替换 ID 越权读取计划或通知。
func TestSystemPlanDetailScope(t *testing.T) {
	orm, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := orm.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	if err = orm.AutoMigrate(&models.SystemPlan{}, &models.SystemNotification{}); err != nil {
		t.Fatal(err)
	}
	p := models.SystemPlan{CreatedBy: 1, AssigneeID: 2, Title: "私有计划", ExecuteAt: time.Now().UTC(), Status: "pending"}
	if err = orm.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	n := models.SystemNotification{SystemPlanID: p.ID, RecipientUserID: 2, Channel: "email", NextRetryAt: time.Now().UTC(), Status: "pending"}
	if err = orm.Create(&n).Error; err != nil {
		t.Fatal(err)
	}
	conn := &adminDB.Sqlite{Base: adminDB.Base{GormList: map[string]*gorm.DB{"default": orm}}}
	for _, notification := range []bool{false, true} {
		id := p.ID
		if notification {
			id = n.ID
		}
		param := parameter.BaseParam().WithPKs(strconv.Itoa(int(id)))
		for _, userID := range []int64{1, 2, 3} {
			user := adminModels.UserModel{Id: userID, Base: adminModels.Base{Conn: conn}}
			rows, count := systemPlanDetailData(user, notification)(param)
			if userID == 3 {
				if count != 0 || len(rows) != 0 {
					t.Fatal("越权读取编辑页")
				}
			} else if count != 1 {
				t.Fatalf("用户 %d 无法读取本人记录", userID)
			}
		}
	}
}

// TestSystemPlanTimeRoundTrip 验证上海输入按 UTC 保存并能正确回显。
func TestSystemPlanTimeRoundTrip(t *testing.T) {
	for _, input := range []string{"2026-11-10 09:30:00", "2026-11-10 09:30", "2026-11-10T09:30", "2026-11-10T09:30:00", "2026-11-10T09:30:00+08:00", "2026-11-10T01:30:00Z"} {
		t.Run(input, func(t *testing.T) {
			value, err := parseSystemPlanTime(input)
			if err != nil {
				t.Fatal(err)
			}
			if value.Format(time.RFC3339) != "2026-11-10T01:30:00Z" {
				t.Fatal(value)
			}
			shown := displaySystemPlanTime(types.FieldModel{Value: value.Format(time.RFC3339)})
			if shown != "2026-11-10 09:30:00" {
				t.Fatal(shown)
			}
		})
	}
	for _, input := range []string{"", "2026-02-30T09:30", "2026-11-10T25:30"} {
		if _, err := parseSystemPlanTime(input); err == nil {
			t.Fatalf("不应接受无效时间 %q", input)
		}
	}
}
