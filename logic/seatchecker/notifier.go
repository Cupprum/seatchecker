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

func sendNotification(ctx context.Context, topic string, text string) error {
	_, span := tr.Start(ctx, "send notification")
	defer span.End()

	b := Notification{
		Topic:   topic,
		Message: text,
		Title:   "Seatchecker",
		Tags:    []string{"airplane"},
	}

	c := Client{scheme: "https", fqdn: "ntfy.sh"}
	_, err := httpsRequestPost[any](c, "/", b)
	if err != nil {
		return fmt.Errorf("failed to send notification: %v", err)
	}

	return nil
}
