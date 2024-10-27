package main

import "testing"

func TestGenerateText(t *testing.T) {
	e := "Window: 4, Middle: 0, Aisle: 2"
	r := EmptySeats{4, 0, 2}.generateText()
	if e != r {
		t.Fatalf("wrong output, expected: %v, received: %v", e, r)
	}
}
