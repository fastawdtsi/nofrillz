package snowid

import (
	"testing"

	"nofrillz/internal/config"
)

func TestMonotonicAndDecode(t *testing.T) {
	layout := DefaultLayout()

	config := &config.IDGeneratorConfig{
		Region: 1,
		Node:   42,
	}

	g, err := New(config, layout)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var last uint64
	for i := 0; i < 5000; i++ {
		id, err := g.Next()
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if i > 0 && id <= last {
			t.Fatalf("id not increasing: got %d <= %d", id, last)
		}
		last = id

		d, err := Decode(id, layout)
		if err != nil {
			t.Fatalf("Decode: %v", err)
		}
		if d.Region != 1 || d.Node != 42 {
			t.Fatalf("decode mismatch: region=%d node=%d", d.Region, d.Node)
		}
	}
}

func TestRanges(t *testing.T) {
	layout := DefaultLayout()

	maxRegion := uint64((uint64(1) << layout.RegionBits) - 1)
	maxNode := uint64((uint64(1) << layout.NodeBits) - 1)

	_, err := New(&config.IDGeneratorConfig{Region: maxRegion + 1, Node: 0}, layout)
	if err == nil {
		t.Fatalf("expected region out of range (max=%d)", maxRegion)
	}

	_, err = New(&config.IDGeneratorConfig{Region: 0, Node: maxNode + 1}, layout)
	if err == nil {
		t.Fatalf("expected node out of range (max=%d)", maxNode)
	}

	_, err = New(&config.IDGeneratorConfig{Region: maxRegion, Node: maxNode}, layout)
	if err != nil {
		t.Fatalf("expected max region/node to be valid: %v", err)
	}
}
