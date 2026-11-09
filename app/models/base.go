package models

import (
	"gorm.io/gorm"
)

var (
	ModelManage = &Manage{
		table: make(map[string]*modelInfo),
	}
)

type modelInfo struct {
	name    string // 表名
	model   any    // 模型
	comment string
}

type Manage struct {
	table map[string]*modelInfo // 模型
}

// SetModel 放置模型
func (m *Manage) SetModel(name string, model any, comment string) *Manage {
	m.table[name] = &modelInfo{
		name:    name,
		model:   model,
		comment: comment,
	}
	return m
}

// Create 创建表
func (m *Manage) Create(db *gorm.DB, names ...string) error {
	tables := make([]*modelInfo, 0)
	for _, name := range names {
		if p, ok := m.table[name]; ok {
			tables = append(tables, p)
		}
	}

	if len(names) == 0 {
		for _, v := range m.table {
			tables = append(tables, v)
		}
	}

	if len(tables) > 0 {
		for _, v := range tables {
			if err := db.Set("gorm:table_options", "COMMENT='"+v.comment+";'").AutoMigrate(v.model); err != nil {
				return err
			}
		}
	}
	return nil
}

// Delete 删除表
func (m *Manage) Delete(db *gorm.DB, names ...string) error {
	var tables []any
	for _, name := range names {
		if p, ok := m.table[name]; ok {
			tables = append(tables, p)
		}
	}

	if len(names) == 0 {
		for _, v := range m.table {
			tables = append(tables, v)
		}
	}

	if len(tables) > 0 {
		if err := db.Migrator().DropTable(tables...); err != nil {
			return err
		}
	}
	return nil
}

// Reset 重置表
func (m *Manage) Reset(db *gorm.DB, names ...string) error {
	err := m.Delete(db, names...)
	if err != nil {
		return err
	}
	err = m.Create(db, names...)
	if err != nil {
		return err
	}
	return nil
}
