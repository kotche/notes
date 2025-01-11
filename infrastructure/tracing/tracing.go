package tracing

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"log"
)

const (
	nameTracer = "note-tracer"
)

type TracerWrapper struct {
	Tracer trace.Tracer
}

func InitTracing(endpoint string) (trace.Tracer, func(), error) {
	exporter, err := jaeger.New(
		jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init jaeger exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.Default()),
	)

	otel.SetTracerProvider(tp)

	cleanup := func() {
		if err = tp.Shutdown(context.Background()); err != nil {
			log.Printf("failed to shutdown tracer provider: %v", err)
		}
	}

	return otel.Tracer(nameTracer), cleanup, nil
}

func StartSpan(ctx context.Context, spanName string) (context.Context, trace.Span) {
	return otel.Tracer(nameTracer).Start(ctx, spanName)
}
