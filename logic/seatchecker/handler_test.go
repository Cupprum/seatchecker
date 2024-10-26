package main

import "testing"

func TestGenerateText(t *testing.T) {
	r := EmptySeats{4, 0, 2}.generateText()
	e := "Window: 4, Middle: 0, Aisle: 2"
	if e != r {
		t.Fatalf("wrong output, expected: %v, received: %v", e, r)
	}
}
