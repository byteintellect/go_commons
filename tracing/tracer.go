package tracing

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	traceSdk "go.opentelemetry.io/otel/sdk/trace"
	semConv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"os"
)

// NewTracer returns an OpenTelemetry TracerProvider configured to use
// the Jaeger exporter that will send spans to the provided url. The returned
// TracerProvider will also use a Resource configured with all the information
// about the application.
func NewTracer(url string) (*traceSdk.TracerProvider, error) {
	// Create the Jaeger exporter
	// Use the agent injected as a sidecar strategy
	exp, err := jaeger.New(jaeger.WithAgentEndpoint(jaeger.WithAgentHost("localhost")))
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
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, nil
}
