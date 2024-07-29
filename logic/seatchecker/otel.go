package main

import (
	"context"
	"fmt"

	lambdadetector "go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var tr trace.Tracer
var tp *sdktrace.TracerProvider

func setupOtel(ctx context.Context) (func(), error) {
	// Configure a new OTLP exporter using environment variables for sending data to Honeycomb over gRPC
	client := otlptracegrpc.NewClient()
	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		return func() {}, fmt.Errorf("failed to initialize exporter: %e", err)
	}

	detector := lambdadetector.NewResourceDetector()
	res, err := detector.Detect(context.Background())
	if err != nil {
		fmt.Printf("failed to detect lambda resources: %v\n", err)
	}

	// Create a new tracer provider with a batch span processor and the otlp exporter
	tp = sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exp),
	)

	// Register the global Tracer provider
	otel.SetTracerProvider(tp)

	tr = tp.Tracer("seatchecker-lambda")

	// Handle shutdown to ensure all sub processes are closed correctly and telemetry is exported
	return func() {
		_ = exp.Shutdown(ctx)
		_ = tp.Shutdown(ctx)
	}, nil
}
