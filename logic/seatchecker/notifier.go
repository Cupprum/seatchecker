package main

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Notification struct {
	Topic   string   `json:"topic"`
	Message string   `json:"message"`
	Title   string   `json:"title"`
	Tags    []string `json:"tags"`
}

func (c Client) sendNotification(ctx context.Context, topic string, text string) error {
	ctx, span := tr.Start(ctx, "notifier_send_notification")
	defer span.End()
	span.SetAttributes(attribute.String("topic", topic), attribute.String("text", text)) // NOTE: delete after testing.

	b := Notification{
		Topic:   topic,
		Message: text,
		Title:   "Seatchecker",
		Tags:    []string{"airplane"},
	}

	_, err := httpsRequestPost[any](ctx, c, "/", b)
	if err != nil {
		err = fmt.Errorf("failed to send notification: %v", err)
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.AddEvent("notification sent")

	return nil
}
