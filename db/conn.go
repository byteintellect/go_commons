package db

import (
	"github.com/byteintellect/gorm-opentelemetry"
	traceSdk "go.opentelemetry.io/otel/sdk/trace"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gLogger "gorm.io/gorm/logger"
	"log"
	"os"
	"time"
)

func NewGormDbConn(dsn string, traceProvider *traceSdk.TracerProvider) (*gorm.DB, error) {
	dbLogger := gLogger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		gLogger.Config{
			SlowThreshold:             time.Second,  // Slow SQL threshold
			LogLevel:                  gLogger.Info, // Log level
			IgnoreRecordNotFoundError: true,         // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,        // Disable color
		},
	)
	// Initialize otel plugin with options
	plugin := otelgorm.NewPlugin(
		// include any options here
		otelgorm.WithTracerProvider(traceProvider),
	)
	// create new mysql database connection
	if db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: dbLogger}); err == nil {
		db.Use(plugin)
		return db, nil
	} else {
		return nil, err
	}
}
