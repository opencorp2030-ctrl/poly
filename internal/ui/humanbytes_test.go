package ui

import "testing"

func TestHumanBytes(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{0, "0B"},
		{512, "512B"},
		{1023, "1023B"},
		{1024, "1.0KiB"},
		{1740, "1.7KiB"},
		{1024 * 1024, "1.0MiB"},
		{1782579, "1.7MiB"},
		{1024 * 1024 * 1024, "1.0GiB"},
	}
	for _, c := range cases {
		if got := HumanBytes(c.n); got != c.want {
			t.Errorf("HumanBytes(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}
