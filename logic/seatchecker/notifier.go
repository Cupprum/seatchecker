package main

import (
	"context"
	"fmt"
)

type Notification struct {
	Topic   string   `json:"topic"`
	Message string   `json:"message"`
	Title   string   `json:"title"`
	Tags    []string `json:"tags"`
}

func (c Client) sendNotification(ctx context.Context, topic string, text string) error {
	// _, span := tr.Start(ctx, "send notification")
	// defer span.End()

	b := Notification{
		Topic:   topic,
		Message: text,
		Title:   "Seatchecker",
		Tags:    []string{"airplane"},
	}

	_, err := httpsRequestPost[any](c, "/", b)
	if err != nil {
		return fmt.Errorf("failed to send notification: %v", err)
	}

	return nil
}
