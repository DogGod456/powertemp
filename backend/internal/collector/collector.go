package collector

import (
	"context"
	"log"
	"time"

	"powertemp/backend/internal/db"
	"powertemp/backend/internal/domain"
	"powertemp/backend/internal/realtime"
	"powertemp/backend/internal/simulator"
)

// Collector связывает активные датчики с sensor-simulator: получает новые
// значения, сохраняет их в PostgreSQL и публикует в realtime-hub.
type Collector struct {
	store *db.Store
	sim   *simulator.Client
	hub   *realtime.Hub
}

// New создает сборщик с клиентом внешнего сервиса симуляции.
func New(store *db.Store, simulatorURL string, hub *realtime.Hub) *Collector {
	return &Collector{
		store: store,
		sim:   simulator.NewClient(simulatorURL),
		hub:   hub,
	}
}

// Start запускает фоновый цикл опроса. Сам тик частый, но isDue пропускает
// датчики, у которых еще не наступила их индивидуальная частота сбора.
func (c *Collector) Start(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.tick(ctx)
		}
	}
}

// tick выполняет один проход: выбирает активные датчики, генерирует данные,
// пишет измерение, обновляет карточку датчика и отправляет SSE-событие.
func (c *Collector) tick(ctx context.Context) {
	sensors, err := c.store.GetActiveSensors(ctx)
	if err != nil {
		log.Printf("collector sensors: %v", err)
		return
	}
	now := time.Now().UTC()
	for _, sensor := range sensors {
		if !isDue(sensor, now) {
			continue
		}
		generated, err := c.sim.Generate(ctx, sensor)
		if err != nil {
			log.Printf("sensor %s generation error: %v", sensor.Code, err)
			continue
		}
		m := domain.Measurement{
			SourceType:     "sensor",
			SensorID:       &sensor.ID,
			SensorCode:     sensor.Code,
			MeasuredAt:     generated.MeasuredAt,
			TemperatureC:   generated.TemperatureC,
			ConsumptionKWh: generated.ConsumptionKWh,
		}
		inserted, err := c.store.InsertMeasurement(ctx, m)
		if err != nil {
			log.Printf("insert measurement: %v", err)
			continue
		}
		if err := c.store.UpdateSensorReading(ctx, sensor.ID, generated.TemperatureC, generated.ConsumptionKWh, generated.MeasuredAt); err != nil {
			log.Printf("update sensor reading: %v", err)
		}
		c.hub.Publish("measurement", inserted)
	}
}

// isDue проверяет, пора ли снимать новое измерение с учетом frequency_seconds.
func isDue(sensor domain.Sensor, now time.Time) bool {
	if sensor.LastCollectedAt == nil {
		return true
	}
	frequency := time.Duration(sensor.FrequencySeconds) * time.Second
	return now.Sub(*sensor.LastCollectedAt) >= frequency
}
