package proxy

import "testing"

func TestNew_RejectsUnsupportedTransportMode(t *testing.T) {
	_, err := New(
		"localhost:50051",
		"localhost:50052",
		"localhost:50053",
		"localhost:50054",
		"localhost:50055",
		"localhost:50056",
		"localhost:50057",
		"localhost:50058",
		TransportConfig{Mode: "unsupported"},
	)
	if err == nil {
		t.Fatal("expected error for unsupported transport mode")
	}
}
