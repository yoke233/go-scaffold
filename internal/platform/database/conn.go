package database

import (
	"project/internal/conf"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewDB(c *conf.Bootstrap) (*gorm.DB, func(), error) {
	db, err := gorm.Open(postgres.Open(c.Data.Database.DSN), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	}

	return db, cleanup, nil
}
