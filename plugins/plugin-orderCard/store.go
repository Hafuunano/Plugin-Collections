package pluginordercard

import (
	"sync"
	"time"

	"github.com/Hafuunano/Plugin-Collections/lib/database/sqlite"
	"gorm.io/gorm"
)

// store holds the GORM DB and timezone for pool CRUD (table types in this plugin).
type store struct {
	db *gorm.DB
	mu sync.RWMutex
	tz *time.Location
}

func openStore(dbPath string) (*store, error) {
	db, err := sqlite.Open(dbPath)
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&PoolEntry{}); err != nil {
		return nil, err
	}
	tz, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		tz = time.UTC
	}
	return &store{db: db, tz: tz}, nil
}

func (s *store) today4am(t time.Time) time.Time {
	y, m, d := t.In(s.tz).Date()
	return time.Date(y, m, d, 4, 0, 0, 0, s.tz)
}

func (s *store) shouldReset(now time.Time, lastResetAt time.Time) bool {
	t4 := s.today4am(now)
	return !now.Before(t4) && lastResetAt.Before(t4)
}

func (s *store) Get(groupID string) (instruction string, value int, lastResetAt time.Time, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var row PoolEntry
	result := s.db.Where("group_id = ?", groupID).First(&row)
	if result.Error == gorm.ErrRecordNotFound {
		return "", 0, time.Time{}, nil
	}
	if result.Error != nil {
		return "", 0, time.Time{}, result.Error
	}
	lastResetAt = row.LastResetAt.In(s.tz)
	value = row.Value
	instruction = row.Instruction
	now := time.Now().In(s.tz)
	if s.shouldReset(now, lastResetAt) {
		value = 0
		lastResetAt = s.today4am(now)
	}
	return instruction, value, lastResetAt, nil
}

func (s *store) Create(groupID, instruction string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().In(s.tz)
	entry := PoolEntry{
		GroupID:     groupID,
		Instruction: instruction,
		Value:       0,
		LastResetAt: now,
	}
	return s.db.Save(&entry).Error
}

func (s *store) UpdateValue(groupID string, value int, lastResetAt time.Time) error {
	if value < 0 {
		value = 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Model(&PoolEntry{}).Where("group_id = ?", groupID).Updates(map[string]any{
		"value":         value,
		"last_reset_at": lastResetAt,
	}).Error
}

func (s *store) SetValue(groupID string, value int) error {
	if value < 0 {
		value = 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().In(s.tz)
	return s.db.Model(&PoolEntry{}).Where("group_id = ?", groupID).Updates(map[string]any{
		"value":         value,
		"last_reset_at": now,
	}).Error
}

func (s *store) Delete(groupID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Where("group_id = ?", groupID).Delete(&PoolEntry{}).Error
}
