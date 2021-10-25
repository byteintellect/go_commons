package tracing

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	traceSdk "go.opentelemetry.io/otel/sdk/trace"
	semConv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"os"
)

// tracerProvider returns an OpenTelemetry TracerProvider configured to use
// the Jaeger exporter that will send spans to the provided url. The returned
// TracerProvider will also use a Resource configured with all the information
// about the application.
func tracerProvider(url string) (*traceSdk.TracerProvider, error) {
	// Create the Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(url)))
	if err != nil {
		return nil, err
	}
	tp := traceSdk.NewTracerProvider(
		// Always be sure to batch in production.
		traceSdk.WithBatcher(exp),
		// Record information about this application in an Resource.
		traceSdk.WithResource(resource.NewWithAttributes(
			semConv.SchemaURL,
			semConv.ServiceNameKey.String(os.Getenv("APP_NAME")),
			attribute.String("environment", os.Getenv("APP_ENV")),
			attribute.String("ID", os.Getenv("APP_VERSION")),
		)),
	)
	return tp, nil
}
