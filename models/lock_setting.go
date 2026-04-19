package models

type LockSetting struct {
	ID        uint `gorm:"primaryKey;autoIncrement" json:"-"`
	SlotLock  bool `gorm:"not null;default:false" json:"slotLock"`
	RouteLock bool `gorm:"not null;default:false" json:"routeLock"`
}
