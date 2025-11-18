package model

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
)

var _db *gorm.DB

func init() {
	var err error
	// github.com/mattn/go-sqlite3
	_db, err = gorm.Open(sqlite.Open("airplay-tv.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		log.Fatalln("[sqlite.Error]", err.Error())
	}

	autoMigrate()
}

func DB() *gorm.DB {
	return _db
}

func autoMigrate() {
	if err := DB().AutoMigrate(&VideoResolution{}); err != nil {
		log.Println("[AutoMigrate.Error]", err.Error())
	}
	_ = DB().AutoMigrate(&Source{})
	_ = DB().AutoMigrate(&Visitor{})
}
