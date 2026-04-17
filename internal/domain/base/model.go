package base

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel 基础模型，包含公共字段
type BaseModel struct {
	ID        string         `gorm:"type:varchar(26);primaryKey" json:"id"`
	TenantID  string         `gorm:"type:varchar(26);index" json:"tenant_id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// GetTenantID 实现 crud.TenantOwned 接口
func (m BaseModel) GetTenantID() string {
	return m.TenantID
}

// JSON 自定义 JSON 类型，支持 GORM jsonb 存储。
// nil JSON 值在数据库中存储为 SQL NULL（而非空对象 {}）。
// 如果业务需要区分"未设置"和"空对象"，请使用显式初始化：make(JSON)。
type JSON map[string]any

// Value 实现 driver.Valuer 接口
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan 实现 sql.Scanner 接口
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// GenerateUUID 生成 UUID v7，安全处理错误
func GenerateUUID() string {
	id, err := uuid.NewV7()
	if err != nil {
		// UUID v7 生成失败概率极低，回退到 UUID v4
		id = uuid.Must(uuid.NewRandom())
	}
	return id.String()
}
