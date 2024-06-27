package main

import (
	"testing"
)

func TestGenerateText(t *testing.T) {
	o := generateText(InEvent{Window: 4, Middle: 0, Aisle: 2})
	ex := "Window: 4, Middle: 0, Aisle: 2"
	if ex != o {
		t.Fatalf("wrong output, expected: %v, received: %v", ex, o)
	}
}

// func TestSendNotification(t *testing.T) {
// 	// TODO: implement custom http mock server, i think it should generate url
// 	err := sendNotification("https://www.examplelll.com", "test-topic", "test-text")

// 	if err != nil {
// 		t.Fatalf("error: %v", err)
// 	}
// }
