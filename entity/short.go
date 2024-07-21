package entity

import "time"

// URL represents the data in the urls table
type URL struct {
	ID          int       `gorm:"primaryKey;autoIncrement" json:"id"`
	OriginalURL string    `gorm:"type:text;not null" json:"original_url"`
	ShortURL    string    `gorm:"type:text;not null;unique" json:"short_url"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}
