package database

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"reposter/config"
)

type Database struct {
	Conn *gorm.DB
	conf *config.Config
}

func (db *Database) AutoMigrate() error {
	if err := db.Conn.AutoMigrate(&Post{}).Error; err != nil {
		return err
	}

	return nil
}

func NewDatabase(conf *config.Config) (*Database, error) {
	conn, err := gorm.Open("sqlite3", conf.Database)
	if err != nil {
		return nil, err
	}

	return &Database{
		Conn: conn,
		conf: conf,
	}, nil
}
