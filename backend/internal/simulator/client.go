package simulator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"powertemp/backend/internal/domain"
)

// Client вызывает отдельный сервис sensor-simulator, который отвечает за
// генерацию реалистичных показаний виртуальных датчиков.
type Client struct {
	baseURL string
	http    *http.Client
}

type generateRequest struct {
	Code               string   `json:"code"`
	Name               string   `json:"name"`
	CurrentTemperature *float64 `json:"current_temperature"`
	CurrentConsumption *float64 `json:"current_consumption"`
}

type GenerateResponse struct {
	Code           string    `json:"code"`
	MeasuredAt     time.Time `json:"measured_at"`
	TemperatureC   float64   `json:"temperature_c"`
	ConsumptionKWh float64   `json:"consumption_kwh"`
}

// NewClient нормализует адрес симулятора и задает короткий timeout, чтобы
// collector не зависал надолго при проблемах с сервисом генерации.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

// Generate передает симулятору текущие показания датчика и получает следующую
// сгенерированную точку временного ряда.
func (c *Client) Generate(ctx context.Context, sensor domain.Sensor) (GenerateResponse, error) {
	body, err := json.Marshal(generateRequest{
		Code:               sensor.Code,
		Name:               sensor.Name,
		CurrentTemperature: sensor.CurrentTemperature,
		CurrentConsumption: sensor.CurrentConsumption,
	})
	if err != nil {
		return GenerateResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return GenerateResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return GenerateResponse{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return GenerateResponse{}, fmt.Errorf("simulator status: %s", res.Status)
	}
	var out GenerateResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return GenerateResponse{}, err
	}
	return out, nil
}
