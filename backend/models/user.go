package models

import "time"

type User struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    GoogleID  string    `gorm:"uniqueIndex;not null" json:"google_id"`
    Email     string    `gorm:"uniqueIndex;not null" json:"email"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
}
