package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// fixedNow is a stable reference time used across tests (Monday 2024-06-10 14:30 UTC).
var fixedNow = time.Date(2024, 6, 10, 14, 30, 0, 0, time.UTC)

// rfcTime formats a time.Time as RFC3339 for use as sensor state values.
func rfcTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

// ---- buildChartPoints -------------------------------------------------------

func TestBuildChartPoints(t *testing.T) {
	tests := []struct {
		name      string
		values    []float64
		wantLen   int
		wantStep  float64
		wantFirst string
		wantLast  string
	}{
		{
			name:    "empty slice",
			values:  []float64{},
			wantLen: 0, wantStep: 0,
		},
		{
			name:      "single value",
			values:    []float64{42},
			wantLen:   1, wantStep: 0,
			wantFirst: "0.00,25.00",
		},
		{
			name:      "all equal values",
			values:    []float64{5, 5, 5},
			wantLen:   3, wantStep: 50,
			wantFirst: "0.00,25.00", wantLast: "100.00,25.00",
		},
		{
			name:      "min and max (two values)",
			values:    []float64{0, 50},
			wantLen:   2, wantStep: 100,
			// min price → scaled=0 → svgY=50 (bottom); max price → scaled=50 → svgY=0 (top)
			wantFirst: "0.00,50.00", wantLast: "100.00,0.00",
		},
		{
			name:      "ascending three values",
			values:    []float64{0, 25, 50},
			wantLen:   3, wantStep: 50,
			wantFirst: "0.00,50.00", wantLast: "100.00,0.00",
		},
		{
			name:      "descending three values",
			values:    []float64{50, 25, 0},
			wantLen:   3, wantStep: 50,
			wantFirst: "0.00,0.00", wantLast: "100.00,50.00",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pts, step := buildChartPoints(tc.values)
			if len(pts) != tc.wantLen {
				t.Errorf("len(pts) = %d, want %d", len(pts), tc.wantLen)
			}
			if step != tc.wantStep {
				t.Errorf("step = %f, want %f", step, tc.wantStep)
			}
			if tc.wantFirst != "" && (len(pts) == 0 || pts[0] != tc.wantFirst) {
				t.Errorf("pts[0] = %q, want %q", pts[0], tc.wantFirst)
			}
			if tc.wantLast != "" && (len(pts) == 0 || pts[len(pts)-1] != tc.wantLast) {
				t.Errorf("pts[last] = %q, want %q", pts[len(pts)-1], tc.wantLast)
			}
		})
	}
}

func TestBuildChartPointsYInversion(t *testing.T) {
	// Higher price must produce a lower svgY (sits higher on screen).
	pts, _ := buildChartPoints([]float64{10, 100})
	// pts[0] = low price → high svgY; pts[1] = high price → low svgY
	var y0, y1 float64
	fmt.Sscanf(pts[0], "%f,%f", new(float64), &y0)
	fmt.Sscanf(pts[1], "%f,%f", new(float64), &y1)
	if y0 <= y1 {
		t.Errorf("low price should have higher svgY than high price: y0=%f y1=%f", y0, y1)
	}
}

// ---- buildHighlight ---------------------------------------------------------

func makePts(n int) []string {
	pts := make([]string, n)
	for i := range pts {
		pts[i] = fmt.Sprintf("%d.00,25.00", i*10)
	}
	return pts
}

func TestBuildHighlight(t *testing.T) {
	// stateAt returns an RFC3339 timestamp for a given offset from fixedNow's midnight.
	midnight := time.Date(fixedNow.Year(), fixedNow.Month(), fixedNow.Day(), 0, 0, 0, 0, time.UTC)
	stateAt := func(hoursFromMidnight float64) string {
		d := time.Duration(hoursFromMidnight * float64(time.Hour))
		return rfcTime(midnight.Add(d))
	}

	// 192 pts = 48 hours of 15-min data (today + tomorrow)
	pts192 := makePts(192)

	tests := []struct {
		name         string
		pts          []string
		stateTimeStr string
		wantEmpty    bool
		wantContains string
	}{
		{
			name:      "invalid RFC3339",
			pts:       pts192,
			stateTimeStr: "not-a-timestamp",
			wantEmpty: true,
		},
		{
			name:      "startIdx before midnight (negative)",
			pts:       pts192,
			stateTimeStr: rfcTime(midnight.Add(-1 * time.Hour)),
			wantEmpty: true,
		},
		{
			name:      "startIdx beyond data range",
			pts:       makePts(10),
			stateTimeStr: stateAt(10), // idx = 10*4 = 40, but len=10
			wantEmpty: true,
		},
		{
			name:         "normal 4-hour window",
			pts:          pts192,
			stateTimeStr: stateAt(12), // idx = 48, endIdx = 48+16 = 64
			wantContains: "<polyline",
		},
		{
			name:         "window clamps to end of data",
			pts:          makePts(50),
			stateTimeStr: stateAt(12), // idx=48, endIdx=64 → clamped to 49
			wantContains: "<polyline",
		},
		{
			name: "endIdx equals startIdx after clamp → empty",
			// 1 pt at idx 0, endIdx = 0+16 clamped to 0, not > startIdx
			pts:          makePts(1),
			stateTimeStr: stateAt(0),
			wantEmpty:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildHighlight(tc.pts, tc.stateTimeStr, fixedNow)
			if tc.wantEmpty && got != "" {
				t.Errorf("expected empty string, got %q", got)
			}
			if !tc.wantEmpty && got == "" {
				t.Errorf("expected non-empty string")
			}
			if tc.wantContains != "" && !strings.Contains(got, tc.wantContains) {
				t.Errorf("output does not contain %q: %q", tc.wantContains, got)
			}
		})
	}
}

// ---- buildNowDot ------------------------------------------------------------

func TestBuildNowDot(t *testing.T) {
	midnight := time.Date(fixedNow.Year(), fixedNow.Month(), fixedNow.Day(), 0, 0, 0, 0, time.UTC)
	// 48 hourly values spanning today+tomorrow; step for 48 pts = 100/47
	values48 := make([]float64, 48)
	for i := range values48 {
		values48[i] = float64(i)
	}
	pts48, step48 := buildChartPoints(values48)

	tests := []struct {
		name         string
		values       []float64
		pts          []string
		step         float64
		now          time.Time
		wantEmpty    bool
		wantContains string
	}{
		{
			name:      "empty values",
			values:    []float64{},
			pts:       []string{},
			step:      0,
			now:       fixedNow,
			wantEmpty: true,
		},
		{
			// now is the previous day at 23:00 → idxFloat = 23h * 4 = 92 > 47
			name:      "now before midnight (previous day)",
			values:    values48,
			pts:       pts48,
			step:      step48,
			now:       midnight.Add(-1 * time.Hour),
			wantEmpty: true,
		},
		{
			// idxFloat = 12h * 4 = 48 > 47 (last valid index for 48 values)
			name:      "now after last data point",
			values:    values48,
			pts:       pts48,
			step:      step48,
			now:       midnight.Add(12 * time.Hour),
			wantEmpty: true,
		},
		{
			name:         "now at midnight (index 0)",
			values:       values48,
			pts:          pts48,
			step:         step48,
			now:          midnight,
			wantContains: `x1="0.00"`,
		},
		{
			// idxFloat = 11.75h * 4 = 47.0 — exactly the last valid index
			name:         "now at last index (ceilIdx clamped)",
			values:       values48,
			pts:          pts48,
			step:         step48,
			now:          midnight.Add(11*time.Hour + 45*time.Minute),
			wantContains: "<line",
		},
		{
			// idxFloat = 2.125h * 4 = 8.5 — fractional, not integer
			name:         "now between two points gives fractional X",
			values:       values48,
			pts:          pts48,
			step:         step48,
			now:          midnight.Add(time.Duration(2.125 * float64(time.Hour))),
			wantContains: "<line",
		},
		{
			// idxFloat = 0.75h * 4 = 3.0 — within [0, 3] for 4 values
			name:         "all-equal values → y=25",
			values:       []float64{7, 7, 7, 7},
			pts:          func() []string { p, _ := buildChartPoints([]float64{7, 7, 7, 7}); return p }(),
			step:         func() float64 { _, s := buildChartPoints([]float64{7, 7, 7, 7}); return s }(),
			now:          midnight.Add(45 * time.Minute),
			wantContains: `y1="25.00"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildNowDot(tc.values, tc.pts, tc.step, tc.now)
			if tc.wantEmpty && got != "" {
				t.Errorf("expected empty string, got %q", got)
			}
			if !tc.wantEmpty && got == "" {
				t.Errorf("expected non-empty string")
			}
			if tc.wantContains != "" && !strings.Contains(got, tc.wantContains) {
				t.Errorf("output does not contain %q: %q", tc.wantContains, got)
			}
		})
	}
}

func TestBuildNowDotFractionalX(t *testing.T) {
	// Dot at 14.5 hours must have a fractional X, not the integer-truncated value.
	midnight := time.Date(fixedNow.Year(), fixedNow.Month(), fixedNow.Day(), 0, 0, 0, 0, time.UTC)
	values := make([]float64, 96) // one day of 15-min data
	for i := range values {
		values[i] = float64(i)
	}
	pts, step := buildChartPoints(values)
	now := midnight.Add(time.Duration(14.5 * float64(time.Hour)))
	got := buildNowDot(values, pts, step, now)

	// idxFloat = 14.5 * 4 = 58.0  → x = 58 * step. Not 57 or 59.
	expectedX := 58.0 * step
	wantX1 := fmt.Sprintf(`x1="%.2f"`, expectedX)
	if !strings.Contains(got, wantX1) {
		t.Errorf("expected %q in output %q", wantX1, got)
	}
}

// ---- getTimeInfoAt ----------------------------------------------------------

func TestGetTimeInfoAt(t *testing.T) {
	// fixedNow = 2024-06-10 14:30 UTC (Monday)
	midnight := time.Date(fixedNow.Year(), fixedNow.Month(), fixedNow.Day(), 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		input    string
		wantDay  string
		wantTime string
	}{
		{
			name:     "invalid string",
			input:    "not-a-timestamp",
			wantDay:  "not-a-timestamp",
			wantTime: "",
		},
		{
			name:     "same day morning",
			input:    rfcTime(midnight.Add(9 * time.Hour)),
			wantDay:  "Today",
			wantTime: "09:00",
		},
		{
			name:     "same day at midnight",
			input:    rfcTime(midnight),
			wantDay:  "Today",
			wantTime: "00:00",
		},
		{
			name:     "same day just before midnight",
			input:    rfcTime(midnight.Add(23*time.Hour + 59*time.Minute)),
			wantDay:  "Today",
			wantTime: "23:59",
		},
		{
			name:     "next day",
			input:    rfcTime(midnight.Add(25 * time.Hour)),
			wantDay:  "Tomorrow",
			wantTime: "01:00",
		},
		{
			name:     "two days out",
			input:    rfcTime(midnight.Add(49 * time.Hour)), // Wednesday
			wantDay:  "Wednesday",
			wantTime: "01:00",
		},
		{
			name:     "yesterday",
			input:    rfcTime(midnight.Add(-1 * time.Hour)), // Sunday 23:00
			wantDay:  "Sunday",
			wantTime: "23:00",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := getTimeInfoAt(tc.input, fixedNow)
			if got.Day != tc.wantDay {
				t.Errorf("Day = %q, want %q", got.Day, tc.wantDay)
			}
			if got.Time != tc.wantTime {
				t.Errorf("Time = %q, want %q", got.Time, tc.wantTime)
			}
		})
	}
}
