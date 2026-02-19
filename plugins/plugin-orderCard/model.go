package pluginordercard

import "time"

// PoolEntry is the GORM model for ordercard_pools table (used by this plugin only).
type PoolEntry struct {
	GroupID     string    `gorm:"column:group_id;primaryKey;type:text"`
	Instruction string    `gorm:"column:instruction;not null;type:text"`
	Value       int       `gorm:"column:value;not null;default:0"`
	LastResetAt time.Time `gorm:"column:last_reset_at;not null"`
}

// TableName returns the table name for GORM.
func (PoolEntry) TableName() string {
	return "ordercard_pools"
}
