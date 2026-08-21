package version

import "testing"

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"15.1.0", "15.1.0", 0},
		{"15.1.0", "15.1.1", -1},
		{"15.1.1", "15.1.0", 1},
		{"9.9.9", "10.0.0", -1},
		{"10.0.0", "9.9.9", 1},
		{"1.0.0", "1.0.0-rc1", 1},
		{"1.0.0-rc1", "1.0.0", -1},
		{"v1.2.3", "1.2.3", 0},
		{"1.2.3", "1.2", 1},
		{"1.2", "1.2.0", 0},
		{"", "1.0.0", -1},
		{"dev", "0.1.0", -1},
		{"0.1.0", "dev", 1},
		{"1.0", "1", 0},
		{"2.31.0", "2.31.0", 0},
	}
	for _, c := range cases {
		if got := Compare(c.a, c.b); got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestNewer(t *testing.T) {
	if !Newer("2.0.0", "1.9.9") {
		t.Error("Newer(2.0.0, 1.9.9) = false, want true")
	}
	if Newer("1.0.0", "1.0.0") {
		t.Error("Newer(1.0.0, 1.0.0) = true, want false")
	}
	if Newer("1.0.0-beta", "1.0.0") {
		t.Error("Newer(1.0.0-beta, 1.0.0) = true, want false")
	}
}
