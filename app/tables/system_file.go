package tables

import (
	stdContext "context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"my-base/app/models"
	tablehtml "my-base/app/tables/html"
	"my-base/configs"
	fileService "my-base/module/file"

	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/auth"
	"github.com/GoAdminGroup/go-admin/modules/db"
	"github.com/GoAdminGroup/go-admin/modules/logger"
	form1 "github.com/GoAdminGroup/go-admin/plugins/admin/modules/form"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/GoAdminGroup/go-admin/template/types/form"
	"gorm.io/gorm"
)

func GetSystemFileTable(ctx *context.Context) table.Table {
	config := table.NewDefaultTable(ctx, table.DefaultConfigWithDriver("mysql").SetPrimaryKey("id", db.Bigint))
	systemFileHTML := tablehtml.GetSystemFile()

	info := config.GetInfo().HideFilterArea()
	ConfigureFileNameDisplay(info)
	info.AddField("Id", "id", db.Bigint).
		FieldWidth(50).
		FieldFilterable()
	info.AddField("预览", "public_url", db.Varchar).
		FieldWidth(120).
		FieldDisplay(func(value types.FieldModel) interface{} {
			return systemFileHTML.Preview(value.Value, value.Row["mime_type"])
		})
	info.AddField("原始文件名", "original_name", db.Varchar).
		FieldWidth(150).
		FieldDisplay(DisplayFileName)
	info.AddField("分类", "category", db.Varchar).FieldFilterable()
	info.AddField("大小", "file_size", db.Bigint).
		FieldWidth(200).
		FieldDisplay(func(value types.FieldModel) interface{} {
			size, _ := strconv.ParseInt(value.Value, 10, 64)
			return fmt.Sprintf("%s (%d B)", humanFileSize(size), size)
		})
	info.AddField("MIME", "mime_type", db.Varchar).
		FieldWidth(200)
	//info.AddField("URL", "public_url", db.Varchar).FieldWidth(120).FieldDisplay(func(value types.FieldModel) interface{} {
	//	return systemFileHTML.QuickShareLink(fmt.Sprint(value.Row["id"]), value.Value)
	//})
	info.AddField("下载", "file_hash", db.Varchar).
		FieldWidth(70).
		FieldDisplay(func(value types.FieldModel) interface{} {
			return systemFileHTML.DownloadButton(fmt.Sprint(value.Row["id"]))
		})
	info.AddColumn("分享", func(value types.FieldModel) interface{} {
		return systemFileHTML.ShareButton(fmt.Sprint(value.Row["id"]))
	}).FieldWidth(70)
	info.AddField("状态", "status", db.Int).
		FieldWidth(65).
		FieldDisplay(func(value types.FieldModel) interface{} {
			return systemFileHTML.Status(value.Value)
		})
	info.AddField("Created_at", "created_at", db.Datetime).
		FieldWidth(180).
		FieldDisplay(func(value types.FieldModel) interface{} {
			t, err := time.Parse(time.RFC3339Nano, value.Value)
			if err != nil {
				return value.Value
			}
			return t.Format(time.DateTime)
		})
	info.AddField("Updated_at", "updated_at", db.Datetime).
		FieldWidth(180).
		FieldDisplay(func(value types.FieldModel) interface{} {
			t, err := time.Parse(time.RFC3339Nano, value.Value)
			if err != nil {
				return value.Value
			}
			return t.Format(time.DateTime)
		})
	info.SetTable("system_files").SetTitle("系统文件").SetDescription("系统文件").
		SetDeleteFnWithDB(func(gormDB *gorm.DB, ids []string) error {
			return deleteSystemFilesByDB(gormDB, ids)
		})
	info.SetFooterHtml(systemFileHTML.ShareModal())
	info.Where("status", "!=", -1)
	user := auth.Auth(ctx)
	if !user.IsSuperAdmin() {
		info.Where("is_public", "=", 1)
	}
	formList := config.GetForm()
	formList.AddField("Id", "id", db.Bigint, form.Default)
	tusUploadHTML := tablehtml.GetTusUploadHTML()
	formList.AddField("上传文件", "storage_path", db.Varchar, form.Custom).
		FieldCustomContent(tusUploadHTML.TusUploadContent()).
		FieldCustomJs(tusUploadHTML.TusUploadScript(configs.GetTusUpload().Endpoint)).
		FieldCustomCss(tusUploadHTML.TusUploadStyle()).
		FieldHelpMsg(tusUploadHTML.UploadHelp())
	formList.AddField("原始文件名", "original_name", db.Varchar, form.Text)
	formList.AddField("文件名", "file_name", db.Varchar, form.Text).FieldHide()
	formList.AddField("扩展名", "file_ext", db.Varchar, form.Text).FieldHide()
	formList.AddField("MIME", "mime_type", db.Varchar, form.Text).FieldHide()
	formList.AddField("大小", "file_size", db.Bigint, form.Number).FieldHide()
	formList.AddField("Hash", "file_hash", db.Varchar, form.Text).FieldHide()
	formList.AddField("存储驱动", "storage_driver", db.Varchar, form.Text).FieldHide()
	formList.AddField("访问URL", "public_url", db.Varchar, form.Text).FieldHide()
	formList.AddField("分类", "category", db.Varchar, form.Text).FieldDefault("default")
	formList.AddField("公开访问", "is_public", db.Tinyint, form.Switch).
		FieldOptions(types.FieldOptions{{Text: "否", Value: "0"}, {Text: "是", Value: "1"}}).
		FieldDefault("1")
	formList.AddField("状态", "status", db.Int, form.SelectSingle).
		FieldOptions(types.FieldOptions{{Text: "正常", Value: "1"}, {Text: "禁用", Value: "0"}}).
		FieldDefault("1")
	formList.SetTable("system_files").SetTitle("系统文件").SetDescription("系统文件")
	formList.SetInsertFnWithDB(func(gormDB *gorm.DB, values form1.Values) error {
		return insertSystemFileByDB(gormDB, values, user.Id)
	})
	formList.SetUpdateFnWithDB(func(gormDB *gorm.DB, values form1.Values) error {
		return updateSystemFileByDB(gormDB, values, user.Id)
	})

	return config
}

func insertSystemFileByDB(gormDB *gorm.DB, values form1.Values, uploaderID int64) error {
	prepared, err := prepareTusSystemFile(values, uploaderID)
	if err != nil {
		return err
	}
	if err := gormDB.Create(&prepared.File).Error; err != nil {
		return errors.Join(err, prepared.Rollback())
	}
	if err := prepared.Commit(); err != nil {
		logger.Errorf("commit tus upload metadata failed: %v", err)
	}
	return nil
}

func updateSystemFileByDB(gormDB *gorm.DB, values form1.Values, uploaderID int64) error {
	id := values.GetIntDefault("id", 0)
	if id <= 0 {
		return errors.New("invalid id")
	}

	if strings.TrimSpace(values.Get("tus_upload_id")) != "" {
		prepared, err := prepareTusSystemFile(values, uploaderID)
		if err != nil {
			return err
		}
		result := gormDB.Model(&models.SystemFile{}).Where("id = ?", id).Updates(systemFileUpdateValues(&prepared.File))
		if result.Error != nil {
			return errors.Join(result.Error, prepared.Rollback())
		}
		if result.RowsAffected != 1 {
			return errors.Join(errors.New("system file not found"), prepared.Rollback())
		}
		if err := prepared.Commit(); err != nil {
			logger.Errorf("commit tus upload metadata failed: %v", err)
		}
		return nil
	}

	updates := map[string]interface{}{
		"original_name": values.Get("original_name"),
		"category":      systemFileCategory(values),
		"is_public":     values.Get("is_public") != "0",
		"status":        systemFileStatus(values),
		"updated_at":    time.Now(),
	}
	return gormDB.Model(&models.SystemFile{}).Where("id = ?", id).Updates(updates).Error
}

func deleteSystemFilesByDB(gormDB *gorm.DB, ids []string) error {
	if len(ids) == 0 {
		return errors.New("invalid id")
	}
	return gormDB.Model(&models.SystemFile{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{"status": -1, "deleted_at": time.Now()}).Error
}

func prepareTusSystemFile(values form1.Values, uploaderID int64) (*fileService.PreparedTusUpload, error) {
	service, err := fileService.NewTusUploadService(configs.GetTusUpload())
	if err != nil {
		return nil, err
	}
	ctx, cancel := stdContext.WithTimeout(stdContext.Background(), 15*time.Second)
	defer cancel()
	return service.PrepareSystemFile(ctx, &fileService.TusSystemFileInput{
		UploadID:   strings.TrimSpace(values.Get("tus_upload_id")),
		Category:   systemFileCategory(values),
		IsPublic:   values.Get("is_public") != "0",
		Status:     systemFileStatus(values),
		UploaderID: uploaderID,
	})
}

// systemFileUpdateValues 返回替换上传文件时需要完整更新的字段。
func systemFileUpdateValues(fileModel *models.SystemFile) map[string]interface{} {
	return map[string]interface{}{
		"original_name":  fileModel.OriginalName,
		"file_name":      fileModel.FileName,
		"file_ext":       fileModel.FileExt,
		"mime_type":      fileModel.MimeType,
		"file_size":      fileModel.FileSize,
		"file_hash":      fileModel.FileHash,
		"storage_driver": fileModel.StorageDriver,
		"storage_path":   fileModel.StoragePath,
		"public_url":     fileModel.PublicURL,
		"category":       fileModel.Category,
		"is_public":      fileModel.IsPublic,
		"status":         fileModel.Status,
		"uploader_id":    fileModel.UploaderID,
		"updated_at":     time.Now(),
	}
}

func systemFileCategory(values form1.Values) string {
	category := strings.TrimSpace(values.Get("category"))
	if category == "" {
		return "default"
	}
	return category
}

func systemFileStatus(values form1.Values) int {
	status := values.GetIntDefault("status", 1)
	if status != 0 {
		return 1
	}
	return 0
}

func humanFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	value := float64(size)
	idx := -1
	for value >= 1024 && idx < len(units)-1 {
		value /= 1024
		idx++
	}
	if idx < 0 {
		return fmt.Sprintf("%d B", size)
	}
	return fmt.Sprintf("%.2f %s", value, units[idx])
}
