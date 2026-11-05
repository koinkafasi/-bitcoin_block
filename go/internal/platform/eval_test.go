package platform

import "testing"

func TestCalcParse(t *testing.T) {
	vars := map[string]float64{"ana": 10, "orani": 5, "vade": 12}
	cases := []struct {
		expr string
		want float64
	}{
		{"[ana] * 3 + 10", 40},
		{"([ana] * 3 + 10) / 2", 20},
		{"[orani] / 100 * [ana]", 0.5},
		{"[vade] * [ana] / 100", 1.2},
		{"100 - [orani]", 95},
	}
	for _, c := range cases {
		got, err := CalcParse(c.expr, vars)
		if err != nil {
			t.Fatalf("%q: hata %v", c.expr, err)
		}
		if abs(got-c.want) > 1e-9 {
			t.Errorf("%q = %v; want %v", c.expr, got, c.want)
		}
	}
}

func TestCalcParseDivision(t *testing.T) {
	got, err := CalcParse("[a] / 0", map[string]float64{"a": 5})
	if err == nil {
		t.Fatalf("sıfıra bölme hata vermeli; got %v", got)
	}
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
