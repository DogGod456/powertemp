package analysis

import (
	"errors"
	"fmt"
	"math"

	"powertemp/backend/internal/domain"
)

// Summary содержит расчетные показатели для одного источника или объединенного
// набора данных: описательную статистику, корреляцию, регрессию и вывод.
type Summary struct {
	SourceLabel          string  `json:"source_label"`
	PointsCount          int     `json:"points_count"`
	MinTemperature       float64 `json:"min_temperature"`
	MaxTemperature       float64 `json:"max_temperature"`
	AvgTemperature       float64 `json:"avg_temperature"`
	MinConsumption       float64 `json:"min_consumption"`
	MaxConsumption       float64 `json:"max_consumption"`
	AvgConsumption       float64 `json:"avg_consumption"`
	Correlation          float64 `json:"correlation"`
	RegressionA          float64 `json:"regression_a"`
	RegressionB          float64 `json:"regression_b"`
	RSquared             float64 `json:"r_squared"`
	RegressionEquation   string  `json:"regression_equation"`
	Interpretation       string  `json:"interpretation"`
	InsufficientData     bool    `json:"insufficient_data"`
	InsufficientDataText string  `json:"insufficient_data_text,omitempty"`
}

// Forecast хранит прогноз потребления для заданной пользователем температуры.
type Forecast struct {
	TemperatureC            float64 `json:"temperature_c"`
	PredictedConsumptionKWh float64 `json:"predicted_consumption_kwh"`
}

// ChartPoint - подготовленная точка для frontend-графиков: фактическое значение
// и значение, рассчитанное линейной моделью.
type ChartPoint struct {
	MeasuredAt              string  `json:"measured_at"`
	SensorCode              string  `json:"sensor_code"`
	TemperatureC            float64 `json:"temperature_c"`
	ConsumptionKWh          float64 `json:"consumption_kwh"`
	PredictedConsumptionKWh float64 `json:"predicted_consumption_kwh"`
}

// Compute рассчитывает минимум/максимум/среднее, коэффициент корреляции
// Пирсона, линейную регрессию y = a*x + b и R2 для набора измерений.
func Compute(label string, measurements []domain.Measurement) Summary {
	if len(measurements) < 2 {
		return Summary{
			SourceLabel:          label,
			PointsCount:          len(measurements),
			InsufficientData:     true,
			InsufficientDataText: "Для анализа нужно минимум 2 измерения.",
		}
	}

	var sumX, sumY float64
	minX, maxX := measurements[0].TemperatureC, measurements[0].TemperatureC
	minY, maxY := measurements[0].ConsumptionKWh, measurements[0].ConsumptionKWh

	for _, m := range measurements {
		x := m.TemperatureC
		y := m.ConsumptionKWh
		sumX += x
		sumY += y
		minX = math.Min(minX, x)
		maxX = math.Max(maxX, x)
		minY = math.Min(minY, y)
		maxY = math.Max(maxY, y)
	}

	n := float64(len(measurements))
	meanX := sumX / n
	meanY := sumY / n

	var sxx, syy, sxy float64
	for _, m := range measurements {
		dx := m.TemperatureC - meanX
		dy := m.ConsumptionKWh - meanY
		sxx += dx * dx
		syy += dy * dy
		sxy += dx * dy
	}

	if sxx == 0 || syy == 0 {
		return Summary{
			SourceLabel:          label,
			PointsCount:          len(measurements),
			MinTemperature:       round4(minX),
			MaxTemperature:       round4(maxX),
			AvgTemperature:       round4(meanX),
			MinConsumption:       round4(minY),
			MaxConsumption:       round4(maxY),
			AvgConsumption:       round4(meanY),
			InsufficientData:     true,
			InsufficientDataText: "Невозможно рассчитать корреляцию и регрессию: все значения одного из признаков одинаковые.",
		}
	}

	correlation := sxy / math.Sqrt(sxx*syy)
	a := sxy / sxx
	b := meanY - a*meanX

	var ssRes, ssTot float64
	for _, m := range measurements {
		pred := a*m.TemperatureC + b
		res := m.ConsumptionKWh - pred
		ssRes += res * res
		tot := m.ConsumptionKWh - meanY
		ssTot += tot * tot
	}
	r2 := 0.0
	if ssTot != 0 {
		r2 = 1 - ssRes/ssTot
	}

	return Summary{
		SourceLabel:        label,
		PointsCount:        len(measurements),
		MinTemperature:     round4(minX),
		MaxTemperature:     round4(maxX),
		AvgTemperature:     round4(meanX),
		MinConsumption:     round4(minY),
		MaxConsumption:     round4(maxY),
		AvgConsumption:     round4(meanY),
		Correlation:        round4(correlation),
		RegressionA:        round4(a),
		RegressionB:        round4(b),
		RSquared:           round4(r2),
		RegressionEquation: fmt.Sprintf("consumption = %.4f * temperature + %.4f", a, b),
		Interpretation:     InterpretCorrelation(correlation, r2),
	}
}

// ForecastValues применяет найденную регрессию к списку температур и возвращает
// ожидаемое потребление для каждой из них.
func ForecastValues(summary Summary, temps []float64) ([]Forecast, error) {
	if summary.InsufficientData {
		return nil, errors.New(summary.InsufficientDataText)
	}
	result := make([]Forecast, 0, len(temps))
	for _, t := range temps {
		result = append(result, Forecast{
			TemperatureC:            round4(t),
			PredictedConsumptionKWh: round4(summary.RegressionA*t + summary.RegressionB),
		})
	}
	return result, nil
}

// BuildChartPoints прореживает длинные наборы данных до limit точек и добавляет
// к каждой точке прогнозное значение по линии регрессии.
func BuildChartPoints(measurements []domain.Measurement, summary Summary, limit int) []ChartPoint {
	if len(measurements) == 0 {
		return nil
	}
	if limit <= 0 {
		limit = 2000
	}
	step := 1
	if len(measurements) > limit {
		step = int(math.Ceil(float64(len(measurements)) / float64(limit)))
	}
	points := make([]ChartPoint, 0, min(len(measurements), limit))
	for i, m := range measurements {
		if i%step != 0 {
			continue
		}
		pred := 0.0
		if !summary.InsufficientData {
			pred = summary.RegressionA*m.TemperatureC + summary.RegressionB
		}
		points = append(points, ChartPoint{
			MeasuredAt:              m.MeasuredAt.Format("2006-01-02 15:04:05"),
			SensorCode:              m.SensorCode,
			TemperatureC:            round4(m.TemperatureC),
			ConsumptionKWh:          round4(m.ConsumptionKWh),
			PredictedConsumptionKWh: round4(pred),
		})
	}
	return points
}

// InterpretCorrelation превращает численные r и R2 в короткий текстовый вывод,
// который показывается пользователю в интерфейсе и XLSX-отчете.
func InterpretCorrelation(r, r2 float64) string {
	abs := math.Abs(r)
	direction := "прямая"
	if r < 0 {
		direction = "обратная"
	}
	strength := "слабая"
	switch {
	case abs >= 0.9:
		strength = "очень сильная"
	case abs >= 0.7:
		strength = "сильная"
	case abs >= 0.5:
		strength = "умеренная"
	case abs >= 0.3:
		strength = "заметная, но не сильная"
	}
	return fmt.Sprintf("Обнаружена %s %s линейная зависимость между температурой и потреблением. Модель объясняет примерно %.2f%% вариации потребления.", strength, direction, math.Max(0, r2)*100)
}

// round4 округляет расчетные показатели до четырех знаков для стабильного JSON
// и аккуратного отображения в UI.
func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}

// min возвращает меньшее из двух int и используется при предварительном размере
// слайса точек графика.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
