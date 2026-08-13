package database

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mca-rolando/HermesDDNS/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Open(path string) (*gorm.DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	if err := db.AutoMigrate(
		&model.Domain{},
		&model.Device{},
		&model.DeviceIdentityCredential{},
		&model.DDNSCredential{},
		&model.Host{},
		&model.UpdateLog{},
	); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return db, nil
}

func EnsureDomains(db *gorm.DB, domains []string, defaultTTL int, wildcard bool) error {
	for _, name := range domains {
		var d model.Domain
		err := db.Where("name = ?", name).First(&d).Error
		if err == nil {
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("read domain %s: %w", name, err)
		}
		d = model.Domain{Name: name, DefaultTTL: defaultTTL, Enabled: true, Wildcard: wildcard}
		if err := db.Create(&d).Error; err != nil {
			return fmt.Errorf("seed domain %s: %w", name, err)
		}
	}
	return nil
}
