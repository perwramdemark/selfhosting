package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var token string

// pointsPerHour matches the Nordpool sensor's 15-minute interval granularity.
const pointsPerHour = 4

type ApiResponse struct {
	EntityID   string `json:"entity_id"`
	State      string `json:"state"`
	Attributes struct {
		AveragePrice float64 `json:"average_price"`
		DeviceClass  string  `json:"device_class"`
		FriendlyName string  `json:"friendly_name"`
	} `json:"attributes"`
	LastChanged  string `json:"last_changed"`
	LastReported string `json:"last_reported"`
	LastUpdated  string `json:"last_updated"`
	Context      struct {
		ID       string      `json:"id"`
		ParentID interface{} `json:"parent_id"`
		UserID   interface{} `json:"user_id"`
	} `json:"context"`
}

// NordpoolResponse holds the today/tomorrow price arrays from the HA sensor.
type NordpoolResponse struct {
	Attributes struct {
		Today         []float64 `json:"today"`
		Tomorrow      []float64 `json:"tomorrow"`
		TomorrowValid bool      `json:"tomorrow_valid"`
	} `json:"attributes"`
}

// TimeInfo holds the day and time components.
type TimeInfo struct {
	Day  string
	Time string
}

func getTimeInfo(timeStr string) TimeInfo {
	return getTimeInfoAt(timeStr, time.Now())
}

// getTimeInfoAt converts an RFC3339 timestamp to day label and HH:MM, relative to now.
func getTimeInfoAt(timeStr string, now time.Time) TimeInfo {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return TimeInfo{Day: timeStr, Time: ""}
	}
	t = t.Local()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.AddDate(0, 0, 1)
	timeDate := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	timeFormatted := t.Format("15:04")
	if timeDate.Equal(today) {
		return TimeInfo{Day: "Today", Time: timeFormatted}
	} else if timeDate.Equal(tomorrow) {
		return TimeInfo{Day: "Tomorrow", Time: timeFormatted}
	}
	return TimeInfo{Day: t.Format("Monday"), Time: timeFormatted}
}

// buildChartPoints scales price values into SVG coordinate strings.
// X spans 0–100, Y spans 0–50 with high prices at low Y (top of chart).
// Returns the point strings and the X step between points.
func buildChartPoints(values []float64) (pts []string, step float64) {
	if len(values) == 0 {
		return nil, 0
	}
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	if len(values) > 1 {
		step = 100.0 / float64(len(values)-1)
	}
	xs := 0.0
	pts = make([]string, 0, len(values))
	for _, v := range values {
		var scaled float64
		if max == min {
			scaled = 25.0
		} else {
			scaled = ((v - min) / (max - min)) * 50.0
		}
		svgY := 50.0 - scaled
		pts = append(pts, fmt.Sprintf("%.2f,%.2f", xs, svgY))
		xs += step
	}
	return pts, step
}

// buildHighlight returns a <polyline> SVG element highlighting the cheapest
// 4-hour charging window starting at stateTimeStr, or "" if not applicable.
func buildHighlight(pts []string, stateTimeStr string, now time.Time) string {
	stateTime, err := time.Parse(time.RFC3339, stateTimeStr)
	if err != nil {
		return ""
	}
	stateLocal := stateTime.Local()
	todayMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startIdx := int(stateLocal.Sub(todayMidnight).Hours() * float64(pointsPerHour))
	if startIdx < 0 || startIdx >= len(pts) {
		return ""
	}
	endIdx := startIdx + 4*pointsPerHour
	if endIdx > len(pts)-1 {
		endIdx = len(pts) - 1
	}
	if endIdx <= startIdx {
		return ""
	}
	highlightPts := strings.Join(pts[startIdx:endIdx+1], " ")
	return fmt.Sprintf(`<polyline fill="none" stroke="var(--color-primary)" stroke-linejoin="round" stroke-width="1.5px" points="%s" vector-effect="non-scaling-stroke"></polyline>`, highlightPts)
}

// buildNowDot returns a <line> SVG element marking the current time on the
// price chart, or "" if now falls outside the data range.
func buildNowDot(values []float64, pts []string, step float64, now time.Time) string {
	if len(values) == 0 {
		return ""
	}
	todayMid := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	idxFloat := now.Sub(todayMid).Hours() * float64(pointsPerHour)
	if idxFloat < 0 || idxFloat > float64(len(values)-1) {
		return ""
	}
	floorIdx := int(idxFloat)
	frac := idxFloat - float64(floorIdx)
	ceilIdx := floorIdx + 1
	if ceilIdx >= len(values) {
		ceilIdx = floorIdx
	}
	interpolated := values[floorIdx] + (values[ceilIdx]-values[floorIdx])*frac

	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	var scaledNow float64
	if max == min {
		scaledNow = 25.0
	} else {
		scaledNow = ((interpolated - min) / (max - min)) * 50.0
	}
	nowX := idxFloat * step
	nowY := 50.0 - scaledNow
	// A tiny non-zero length ensures stroke-linecap="round" is always rendered.
	return fmt.Sprintf(`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="var(--color-positive)" stroke-width="4" stroke-linecap="round" vector-effect="non-scaling-stroke"></line>`, nowX, nowY, nowX, nowY+0.1)
}

func snippetHandler(w http.ResponseWriter, r *http.Request) {
	req, err := http.NewRequest("GET", "https://ha.w8k.site/api/states/sensor.ev_cheapest_charging_start_time", nil)
	if err != nil {
		http.Error(w, "Kunde inte hämta data från API", http.StatusInternalServerError)
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Kunde inte hämta data från API", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("API Error - Status: %d, Body: %s\n", resp.StatusCode, string(bodyBytes))
		http.Error(w, fmt.Sprintf("API Error: %d", resp.StatusCode), http.StatusInternalServerError)
		return
	}
	var data ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Printf("JSON Parse Error: %v\n", err)
		http.Error(w, "Fel vid parsning av JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Widget-Title", data.Attributes.FriendlyName+" 🔋")
	w.Header().Set("Widget-Content-Type", "html")

	req2, err := http.NewRequest("GET", "https://ha.w8k.site/api/states/sensor.nordpool_kwh_se4_sek_3_10_025", nil)
	if err == nil {
		req2.Header.Add("Content-Type", "application/json")
		req2.Header.Add("Authorization", "Bearer "+token)
		resp2, err2 := client.Do(req2)
		if err2 == nil {
			defer resp2.Body.Close()
			if resp2.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp2.Body)
				fmt.Printf("Nordpool API Error - Status: %d, Body: %s\n", resp2.StatusCode, string(bodyBytes))
			} else {
				var nord NordpoolResponse
				if err3 := json.NewDecoder(resp2.Body).Decode(&nord); err3 == nil {
					combined := append([]float64{}, nord.Attributes.Today...)
					if nord.Attributes.TomorrowValid {
						combined = append(combined, nord.Attributes.Tomorrow...)
					}
					if len(combined) > 0 {
						pts, step := buildChartPoints(combined)
						pointsStr := strings.Join(pts, " ")
						now := time.Now()
						highlightStr := buildHighlight(pts, data.State, now)
						nowDotStr := buildNowDot(combined, pts, step, now)

						timeInfo := getTimeInfo(data.State)
						htmlContent := fmt.Sprintf(`
  <div class="dynamic-columns list-gap-20 list-with-separator">
	  <div class="flex items-center gap-15">
		  <div class="min-width-0">
			  <a class="color-highlight size-h4 block text-truncate" title="%s">%s</a>
			  <div class="text-truncate">%s</div>
		  </div>
		  <a class="market-chart grow" style="height: 2.5rem">
			  <svg class="market-chart" viewBox="0 0 100 50" width="100%%" height="100%%" preserveAspectRatio="none" overflow="visible">
				  <polyline fill="none" stroke="var(--color-text-subdue)" stroke-linejoin="round" stroke-width="1.0px" points="%s" vector-effect="non-scaling-stroke"></polyline>
				  %s
				  %s
			  </svg>
		  </a>
		  <div class="market-values shrink-0">
			  <div class="size-h3 text-right color-primary">%s</div>
			  <div class="text-right">Avg. Price</div>
		  </div>
	  </div>
  </div>
	`, timeInfo.Day, timeInfo.Day, timeInfo.Time, pointsStr, highlightStr, nowDotStr, fmt.Sprintf("%.2f kr", data.Attributes.AveragePrice))
						w.Write([]byte(htmlContent))
						return
					}
				}
			}
		}
	}

	// Fallback: no Nordpool data
	timeInfo := getTimeInfo(data.State)
	htmlContent := fmt.Sprintf(`
  <div class="dynamic-columns list-gap-20 list-with-separator">
      <div class="flex items-center gap-15">
          <div class="min-width-0">
              <a class="color-highlight size-h4 block text-truncate" title="%s">%s</a>
              <div class="text-truncate">%s</div>
          </div>
          <div class="market-values shrink-0">
              <div class="size-h3 text-right color-positive">%s</div>
              <div class="text-right">Average Price</div>
          </div>
      </div>
  </div>
	`, timeInfo.Day, timeInfo.Day, timeInfo.Time, fmt.Sprintf("%.2f kr", data.Attributes.AveragePrice))
	w.Write([]byte(htmlContent))
}

func main() {
	token = os.Getenv("HA_TOKEN")
	if token == "" {
		panic("HA_TOKEN environment variable is required")
	}
	http.HandleFunc("/ev-start-time-snippet", snippetHandler)
	http.HandleFunc("/health", healthHandler)
	fmt.Println("Servern startar på port 8000...")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		panic(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
