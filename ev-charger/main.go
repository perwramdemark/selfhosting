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

type ApiResponse struct {
	EntityID    string `json:"entity_id"`
	State       string `json:"state"`
	Attributes  struct {
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
// Struktur för Nordpool-sensorens response (vi behöver today & tomorrow arrayerna)
type NordpoolResponse struct {
	Attributes struct {
		Today         []float64 `json:"today"`
		Tomorrow      []float64 `json:"tomorrow"`
		TomorrowValid bool      `json:"tomorrow_valid"`
	} `json:"attributes"`
}
// TimeInfo holds the day and time components
type TimeInfo struct {
	Day  string
	Time string
}

// getTimeInfo converts ISO 8601 timestamp to day and time components
func getTimeInfo(timeStr string) TimeInfo {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return TimeInfo{Day: timeStr, Time: ""}
	}
	// Convert to local OS timezone
	t = t.Local()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.AddDate(0, 0, 1)
	timeDate := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	timeFormatted := t.Format("15:04")
	if timeDate.Equal(today) {
		return TimeInfo{Day: "Today", Time: timeFormatted}
	} else if timeDate.Equal(tomorrow) {
		return TimeInfo{Day: "Tomorrow", Time: timeFormatted}
	} else {
		return TimeInfo{Day: t.Format("Monday"), Time: timeFormatted}
	}
}
func snippetHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Hämta data från det externa API:et
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
	defer resp.Body.Close() // Stäng strömmen när vi är klara
	// 2. Verifiera HTTP status code innan JSON-parsning
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("API Error - Status: %d, Body: %s\n", resp.StatusCode, string(bodyBytes))
		http.Error(w, fmt.Sprintf("API Error: %d", resp.StatusCode), http.StatusInternalServerError)
		return
	}
	// 3. Dekoda JSON-responsen
	var data ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Printf("JSON Parse Error: %v\n", err)
		http.Error(w, "Fel vid parsning av JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// 4. Sätt rätt Content-Type så webbläsaren fattar att det är HTML
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Widget-Title", data.Attributes.FriendlyName + " 🔋")
	w.Header().Set("Widget-Content-Type", "html")
	// --- Hämta Nordpool-data (today + tomorrow) ---
	req2, err := http.NewRequest("GET", "https://ha.w8k.site/api/states/sensor.nordpool_kwh_se4_sek_3_10_025", nil)
	if err == nil {
		req2.Header.Add("Content-Type", "application/json")
		req2.Header.Add("Authorization", "Bearer "+token)
		resp2, err2 := client.Do(req2)
		if err2 == nil {
			defer resp2.Body.Close()
			// Verifiera HTTP status code innan JSON-parsning
			if resp2.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp2.Body)
				fmt.Printf("Nordpool API Error - Status: %d, Body: %s\n", resp2.StatusCode, string(bodyBytes))
				// Fallback till originaldata utan Nordpool
			} else {
				var nord NordpoolResponse
				if err3 := json.NewDecoder(resp2.Body).Decode(&nord); err3 == nil {
					// Kombinera today + tomorrow, men bara om tomorrow_valid är true
					combined := append([]float64{}, nord.Attributes.Today...)
					if nord.Attributes.TomorrowValid {
						combined = append(combined, nord.Attributes.Tomorrow...)
					}
					// Om vi har data, skala om till 0-50 och bygg points-strängen
					pointsStr := ""
					highlightStr := ""
					nowDotStr := ""
					if len(combined) > 0 {
						// hitta min och max
						min := combined[0]
						max := combined[0]
						for _, v := range combined {
							if v < min {
								min = v
							}
							if v > max {
								max = v
							}
						}
						// bygg punkter, X skalas så att alla punkter passar inom 0-100
						var step float64
						if len(combined) > 1 {
							step = 100.0 / float64(len(combined)-1)
						} else {
							step = 0.0
						}
						xs := 0.0
						pts := make([]string, 0, len(combined))
						for _, v := range combined {
							var scaled float64
							if max == min {
								// om konstant, placera mitten
								scaled = 25.0
							} else {
								scaled = ((v - min) / (max - min)) * 50.0
							}
							// SVG y går nedåt, så invertera så högre pris hamnar högre på grafen
							svgY := 50.0 - scaled
							pts = append(pts, fmt.Sprintf("%.2f,%.2f", xs, svgY))
							xs += step
						}
						pointsStr = strings.Join(pts, " ")
						// Beräkna highlight-fönster: 5 timmar från data.State
						if stateTime, err4 := time.Parse(time.RFC3339, data.State); err4 == nil {
							stateLocal := stateTime.Local()
							now := time.Now()
							todayMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
							startIdx := int(stateLocal.Sub(todayMidnight).Hours())
							if startIdx >= 0 && startIdx < len(pts) {
								endIdx := startIdx + 5
								if endIdx > len(pts)-1 {
									endIdx = len(pts) - 1
								}
								if endIdx > startIdx {
									highlightPts := strings.Join(pts[startIdx:endIdx+1], " ")
									highlightStr = fmt.Sprintf(`<polyline fill="none" stroke="var(--color-primary)" stroke-linejoin="round" stroke-width="1.5px" points="%s" vector-effect="non-scaling-stroke"></polyline>`, highlightPts)
								}
							}
						}
						// Markera nuvarande tid (lokal/Stockholm) som en punkt
						nowTime := time.Now()
						todayMid := time.Date(nowTime.Year(), nowTime.Month(), nowTime.Day(), 0, 0, 0, 0, nowTime.Location())
						hoursFloat := nowTime.Sub(todayMid).Hours()
						if hoursFloat >= 0 && hoursFloat <= float64(len(combined)-1) {
							floorIdx := int(hoursFloat)
							frac := hoursFloat - float64(floorIdx)
							ceilIdx := floorIdx + 1
							if ceilIdx >= len(combined) {
								ceilIdx = floorIdx
							}
							interpolated := combined[floorIdx] + (combined[ceilIdx]-combined[floorIdx])*frac
							var scaledNow float64
							if max == min {
								scaledNow = 25.0
							} else {
								scaledNow = ((interpolated - min) / (max - min)) * 50.0
							}
							nowX := hoursFloat * step
							nowY := 50.0 - scaledNow
							nowDotStr = fmt.Sprintf(`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="var(--color-highlight)" stroke-width="6" stroke-linecap="round" vector-effect="non-scaling-stroke"></line>`, nowX, nowY, nowX, nowY)
						}
					}
					// ersätt den hårdkodade polyline-points med vår genererade om vi lyckades
					if pointsStr != "" {
						timeInfo := getTimeInfo(data.State)
						htmlContent := fmt.Sprintf(`
  <div class="dynamic-columns list-gap-20 list-with-separator">
	  <div class="flex items-center gap-15">
		  <div class="min-width-0">
			  <a class="color-highlight size-h4 block text-truncate" title="%s">%s</a>
			  <div class="text-truncate">%s</div>
		  </div>
		  <a class="market-chart grow" style="height: 2.5rem">
			  <svg class="market-chart" viewBox="0 0 100 50" width="100%%" height="100%%" preserveAspectRatio="none">
				  <polyline fill="none" stroke="var(--color-text-subdue)" stroke-linejoin="round" stroke-width="1.0px" points="%s" vector-effect="non-scaling-stroke"></polyline>
				  %s
				  %s
			  </svg>
		  </a>
		  <div class="market-values shrink-0">
			  <div class="size-h3 text-right color-positive">%s</div>
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
	// Fallback: Bygg ditt HTML-snippet utan Nordpool-data
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
	// Sätt upp routen
	http.HandleFunc("/ev-start-time-snippet", snippetHandler)
	// healthcheck
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

