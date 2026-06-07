package psn

import "testing"

func TestParseDuration(t *testing.T) {
	cases := map[string]int{
		"PT127H51M2S": 127*60 + 51,
		"PT2H":        120,
		"PT45M":       45,
		"PT0S":        0,
		"":            0,
	}
	for in, want := range cases {
		if got := parseDuration(in); got != want {
			t.Errorf("%q: got %d, want %d", in, got, want)
		}
	}
}
