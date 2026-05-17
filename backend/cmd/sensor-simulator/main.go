// Sensor simulator - отдельный небольшой HTTP-сервис, который генерирует
// реалистичные значения температуры и потребления для виртуальных датчиков.
// API Gateway вызывает его для каждого активного датчика с нужной частотой.
package main

import (
	"encoding/json"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type generateRequest struct {
	Code               string   `json:"code"`
	Name               string   `json:"name"`
	CurrentTemperature *float64 `json:"current_temperature"`
	CurrentConsumption *float64 `json:"current_consumption"`
}

type generateResponse struct {
	Code           string    `json:"code"`
	MeasuredAt     time.Time `json:"measured_at"`
	TemperatureC   float64   `json:"temperature_c"`
	ConsumptionKWh float64   `json:"consumption_kwh"`
}

func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8090"
	}

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// /api/generate принимает текущее состояние датчика и возвращает следующую
	// точку временного ряда: температуру, потребление и время измерения.
	mux.HandleFunc("/api/generate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req generateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.Code == "" {
			http.Error(w, "code is required", http.StatusBadRequest)
			return
		}

		temperature := nextTemperature(rnd, req.CurrentTemperature)
		consumption := nextConsumption(rnd, req.Code, temperature, req.CurrentConsumption)

		writeJSON(w, http.StatusOK, generateResponse{
			Code:           req.Code,
			MeasuredAt:     time.Now().UTC(),
			TemperatureC:   round2(temperature),
			ConsumptionKWh: round2(consumption),
		})
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("PowerTemp sensor simulator listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// nextTemperature строит плавный температурный ряд: первая точка выбирается
// случайно, а следующие слегка отклоняются от предыдущей с медленным дрейфом.
func nextTemperature(rnd *rand.Rand, current *float64) float64 {
	if current == nil {
		return -10 + rnd.Float64()*35 // от -10 до +25 при первом включении
	}

	// Плавный ход: за один тик изменение примерно от -0.35 до +0.35 °C.
	delta := (rnd.Float64() - 0.5) * 0.7
	seasonDrift := math.Sin(float64(time.Now().Unix())/120.0) * 0.05
	next := *current + delta + seasonDrift
	return clamp(next, -35, 38)
}

// nextConsumption моделирует нагрузку объекта: чем дальше температура от
// комфортной, тем выше потребление; суточный коэффициент и шум делают ряд живее.
func nextConsumption(rnd *rand.Rand, code string, temperature float64, current *float64) float64 {
	profile := profileByCode(code)

	// Реалистичная логика: при отклонении от комфортной температуры потребление растет.
	thermalLoad := math.Abs(temperature-profile.ComfortTemperature) * profile.TemperatureFactor
	dailyLoad := dailyCoefficient() * profile.DailyFactor
	noise := (rnd.Float64() - 0.5) * profile.Noise
	target := profile.BaseLoad + thermalLoad + dailyLoad + noise

	if current == nil {
		return clamp(target, 5, 5000)
	}

	// Сглаживание, чтобы потребление тоже не прыгало резко.
	next := *current*0.72 + target*0.28
	return clamp(next, 5, 5000)
}

type sensorProfile struct {
	BaseLoad           float64
	ComfortTemperature float64
	TemperatureFactor  float64
	DailyFactor        float64
	Noise              float64
}

// profileByCode стабильно назначает датчику один из профилей нагрузки по его
// коду, чтобы один и тот же датчик вел себя похоже между запросами.
func profileByCode(code string) sensorProfile {
	sum := 0
	for _, ch := range code {
		sum += int(ch)
	}
	idx := sum % 5
	profiles := []sensorProfile{
		{BaseLoad: 210, ComfortTemperature: 19, TemperatureFactor: 7.4, DailyFactor: 35, Noise: 18},
		{BaseLoad: 320, ComfortTemperature: 21, TemperatureFactor: 5.8, DailyFactor: 70, Noise: 28},
		{BaseLoad: 160, ComfortTemperature: 20, TemperatureFactor: 6.9, DailyFactor: 24, Noise: 15},
		{BaseLoad: 780, ComfortTemperature: 18, TemperatureFactor: 10.2, DailyFactor: 95, Noise: 45},
		{BaseLoad: 430, ComfortTemperature: 22, TemperatureFactor: 4.8, DailyFactor: 55, Noise: 22},
	}
	return profiles[idx]
}

// dailyCoefficient имитирует суточный график: утром и вечером нагрузка выше,
// ночью ниже, днем остается базовой.
func dailyCoefficient() float64 {
	hour := time.Now().Hour()
	switch {
	case hour >= 7 && hour <= 10:
		return 1.25
	case hour >= 11 && hour <= 18:
		return 1.0
	case hour >= 19 && hour <= 22:
		return 1.15
	default:
		return 0.55
	}
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// round2 округляет значения симулятора до двух знаков, чтобы API отдавал
// человекочитаемые измерения без лишнего шума.
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// writeJSON унифицирует JSON-ответы симулятора.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
