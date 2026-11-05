package httpapi

import "testing"

func TestDiffTextLines(t *testing.T) {
	lines := diffTextLines("Bir\nİki\nÜç", "Bir\nİki değişti\nÜç\nDört")
	var adds, dels, same int
	for _, l := range lines {
		switch l.Kind {
		case "add":
			adds++
		case "del":
			dels++
		case "same":
			same++
		}
	}
	if adds != 2 || dels != 1 || same != 2 {
		t.Fatalf("beklenmeyen diff: add=%d del=%d same=%d %#v", adds, dels, same, lines)
	}
}
