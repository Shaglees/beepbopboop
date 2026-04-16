package weather

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"
)

// Response is the structured weather data returned to clients.
type Response struct {
	Current  Current       `json:"current"`
	Hourly   []HourFcast   `json:"hourly"`
	Daily    []DayFcast    `json:"daily"`
	Location LocationMeta  `json:"location"`
}

type Current struct {
	Temp         float64 `json:"temp_c"`
	FeelsLike    float64 `json:"feels_like_c"`
	Humidity     int     `json:"humidity"`
	WindSpeed    float64 `json:"wind_speed_kmh"`
	UVIndex      float64 `json:"uv_index"`
	IsDay        bool    `json:"is_day"`
	Condition    string  `json:"condition"`
	ConditionCode int    `json:"condition_code"`
}

type HourFcast struct {
	Time          string  `json:"time"`
	Temp          float64 `json:"temp_c"`
	Condition     string  `json:"condition"`
	ConditionCode int     `json:"condition_code"`
	PrecipProb    int     `json:"precip_probability"`
}

type DayFcast struct {
	Date          string  `json:"date"`
	High          float64 `json:"high_c"`
	Low           float64 `json:"low_c"`
	Condition     string  `json:"condition"`
	ConditionCode int     `json:"condition_code"`
	Sunrise       string  `json:"sunrise"`
	Sunset        string  `json:"sunset"`
	PrecipProb    int     `json:"precip_probability"`
}

type LocationMeta struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timezone  string  `json:"timezone"`
}

// Service fetches weather from Open-Meteo with a grid-cell cache.
type Service struct {
	client *http.Client
	mu     sync.RWMutex
	cache  map[string]cacheEntry
}

type cacheEntry struct {
	data      *Response
	expiresAt time.Time
}

const cacheTTL = 15 * time.Minute
const gridSize = 0.1 // ~11 km grid cells

func NewService() *Service {
	return &Service{
		client: &http.Client{Timeout: 10 * time.Second},
		cache:  make(map[string]cacheEntry),
	}
}

func gridKey(lat, lon float64) string {
	glat := math.Round(lat/gridSize) * gridSize
	glon := math.Round(lon/gridSize) * gridSize
	return fmt.Sprintf("%.1f,%.1f", glat, glon)
}

func (s *Service) Fetch(lat, lon float64) (*Response, error) {
	key := gridKey(lat, lon)

	s.mu.RLock()
	if entry, ok := s.cache[key]; ok && time.Now().Before(entry.expiresAt) {
		s.mu.RUnlock()
		return entry.data, nil
	}
	s.mu.RUnlock()

	resp, err := s.fetch(lat, lon)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.cache[key] = cacheEntry{data: resp, expiresAt: time.Now().Add(cacheTTL)}
	s.mu.Unlock()

	return resp, nil
}

func (s *Service) fetch(lat, lon float64) (*Response, error) {
	url := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f"+
			"&current=temperature_2m,relative_humidity_2m,apparent_temperature,weather_code,wind_speed_10m,uv_index,is_day"+
			"&hourly=temperature_2m,weather_code,precipitation_probability"+
			"&daily=temperature_2m_max,temperature_2m_min,weather_code,sunrise,sunset,precipitation_probability_max"+
			"&timezone=auto&forecast_days=5&forecast_hours=12",
		lat, lon,
	)

	resp, err := s.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("open-meteo request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open-meteo returned %d", resp.StatusCode)
	}

	var raw openMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("open-meteo decode: %w", err)
	}

	return transform(raw, lat, lon), nil
}

// transform converts Open-Meteo's raw response into our clean API shape.
func transform(raw openMeteoResponse, lat, lon float64) *Response {
	r := &Response{
		Location: LocationMeta{
			Latitude:  lat,
			Longitude: lon,
			Timezone:  raw.Timezone,
		},
	}

	// Current
	r.Current = Current{
		Temp:          raw.Current.Temperature,
		FeelsLike:     raw.Current.ApparentTemp,
		Humidity:      int(raw.Current.Humidity),
		WindSpeed:     raw.Current.WindSpeed,
		UVIndex:       raw.Current.UVIndex,
		IsDay:         raw.Current.IsDay == 1,
		Condition:     wmoCondition(int(raw.Current.WeatherCode)),
		ConditionCode: int(raw.Current.WeatherCode),
	}

	// Hourly (up to 12 hours)
	n := len(raw.Hourly.Time)
	if n > 12 {
		n = 12
	}
	r.Hourly = make([]HourFcast, n)
	for i := 0; i < n; i++ {
		r.Hourly[i] = HourFcast{
			Time:          raw.Hourly.Time[i],
			Temp:          raw.Hourly.Temperature[i],
			Condition:     wmoCondition(int(raw.Hourly.WeatherCode[i])),
			ConditionCode: int(raw.Hourly.WeatherCode[i]),
			PrecipProb:    int(raw.Hourly.PrecipProb[i]),
		}
	}

	// Daily (up to 5 days)
	nd := len(raw.Daily.Time)
	if nd > 5 {
		nd = 5
	}
	r.Daily = make([]DayFcast, nd)
	for i := 0; i < nd; i++ {
		r.Daily[i] = DayFcast{
			Date:          raw.Daily.Time[i],
			High:          raw.Daily.TempMax[i],
			Low:           raw.Daily.TempMin[i],
			Condition:     wmoCondition(int(raw.Daily.WeatherCode[i])),
			ConditionCode: int(raw.Daily.WeatherCode[i]),
			Sunrise:       raw.Daily.Sunrise[i],
			Sunset:        raw.Daily.Sunset[i],
			PrecipProb:    int(raw.Daily.PrecipProbMax[i]),
		}
	}

	return r
}

// Open-Meteo raw response shapes.
type openMeteoResponse struct {
	Timezone string           `json:"timezone"`
	Current  omCurrent        `json:"current"`
	Hourly   omHourly         `json:"hourly"`
	Daily    omDaily          `json:"daily"`
}

type omCurrent struct {
	Temperature  float64 `json:"temperature_2m"`
	ApparentTemp float64 `json:"apparent_temperature"`
	Humidity     float64 `json:"relative_humidity_2m"`
	WeatherCode  float64 `json:"weather_code"`
	WindSpeed    float64 `json:"wind_speed_10m"`
	UVIndex      float64 `json:"uv_index"`
	IsDay        float64 `json:"is_day"`
}

type omHourly struct {
	Time        []string  `json:"time"`
	Temperature []float64 `json:"temperature_2m"`
	WeatherCode []float64 `json:"weather_code"`
	PrecipProb  []float64 `json:"precipitation_probability"`
}

type omDaily struct {
	Time          []string  `json:"time"`
	TempMax       []float64 `json:"temperature_2m_max"`
	TempMin       []float64 `json:"temperature_2m_min"`
	WeatherCode   []float64 `json:"weather_code"`
	Sunrise       []string  `json:"sunrise"`
	Sunset        []string  `json:"sunset"`
	PrecipProbMax []float64 `json:"precipitation_probability_max"`
}

// wmoCondition maps WMO weather codes to human-readable condition strings.
func wmoCondition(code int) string {
	switch code {
	case 0:
		return "Clear"
	case 1:
		return "Mainly Clear"
	case 2:
		return "Partly Cloudy"
	case 3:
		return "Overcast"
	case 45, 48:
		return "Fog"
	case 51:
		return "Light Drizzle"
	case 53:
		return "Drizzle"
	case 55:
		return "Heavy Drizzle"
	case 56, 57:
		return "Freezing Drizzle"
	case 61:
		return "Light Rain"
	case 63:
		return "Rain"
	case 65:
		return "Heavy Rain"
	case 66, 67:
		return "Freezing Rain"
	case 71:
		return "Light Snow"
	case 73:
		return "Snow"
	case 75:
		return "Heavy Snow"
	case 77:
		return "Snow Grains"
	case 80:
		return "Light Showers"
	case 81:
		return "Showers"
	case 82:
		return "Heavy Showers"
	case 85:
		return "Light Snow Showers"
	case 86:
		return "Snow Showers"
	case 95:
		return "Thunderstorm"
	case 96, 99:
		return "Thunderstorm with Hail"
	default:
		return "Partly Cloudy"
	}
}
