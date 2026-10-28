package monitor

import "testing"

func TestClassifyDisk(t *testing.T) {
	const gb = 1 << 30
	cases := []struct {
		name string
		pct  float64
		free uint64
		want DiskLevel
	}{
		{"roomy", 50, 500 * gb, DiskOK},
		{"pct high", 85, 100 * gb, DiskHigh},
		{"pct critical", 92, 100 * gb, DiskCritical},
		{"low free on low pct", 40, 5 * gb, DiskCritical},
		{"free high band", 40, 20 * gb, DiskHigh},
	}
	for _, c := range cases {
		if got := ClassifyDisk(c.pct, c.free); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}
