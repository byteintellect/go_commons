package db

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gLogger "gorm.io/gorm/logger"
	"log"
	"os"
	"time"
)

func NewGormDbConn(dsn string) (*gorm.DB, error) {
	dbLogger := gLogger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		gLogger.Config{
			SlowThreshold:             time.Second,  // Slow SQL threshold
			LogLevel:                  gLogger.Info, // Log level
			IgnoreRecordNotFoundError: true,         // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,        // Disable color
		},
	)
	// create new mysql database connection
	return gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: dbLogger})
}
