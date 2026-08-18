package model

import "gorm.io/gorm"

type Domain struct {
	gorm.Model
	Name       string `gorm:"uniqueIndex;not null" json:"name" validate:"required,fqdn"`
	DefaultTTL int    `gorm:"not null" json:"default_ttl" validate:"min=20,max=86400"`
	Enabled    bool   `gorm:"not null;default:true" json:"enabled"`
	Wildcard   bool   `gorm:"not null;default:false" json:"wildcard"`
}
