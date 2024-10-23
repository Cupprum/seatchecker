package main

import "testing"

func TestGenerateText(t *testing.T) {
	o := SeatState{4, 0, 2}.generateText()
	ex := "Window: 4, Middle: 0, Aisle: 2"
	if ex != o {
		t.Fatalf("wrong output, expected: %v, received: %v", ex, o)
	}
}
