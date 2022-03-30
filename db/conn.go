package db

import (
	"github.com/byteintellect/gorm-opentelemetry"
	traceSdk "go.opentelemetry.io/otel/sdk/trace"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gLogger "gorm.io/gorm/logger"
	gProm "gorm.io/plugin/prometheus"
	"log"
	"os"
	"time"
)

func NewGormDbConn(httpServerPort uint32, dbName, dsn string, traceProvider *traceSdk.TracerProvider) (*gorm.DB, error) {
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

	promPlugin := gProm.New(gProm.Config{
		DBName:         dbName,
		HTTPServerPort: httpServerPort,
		StartServer:    false,
		MetricsCollector: []gProm.MetricsCollector{
			&gProm.MySQL{
				VariableNames: []string{"threads_running"},
			},
		},
	})
	// create new mysql database connection
	if db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: dbLogger}); err == nil {
		db.Use(plugin)
		db.Use(promPlugin)
		return db, nil
	} else {
		return nil, err
	}
}
