package domain

import (
	"time"

	"github.com/google/uuid"
)

// Sensor описывает виртуальный датчик: его настройки сбора, состояние
// включения и последние известные значения, показанные на dashboard.
type Sensor struct {
	ID                  uuid.UUID  `json:"id"`
	Code                string     `json:"code"`
	Name                string     `json:"name"`
	IsActive            bool       `json:"is_active"`
	FrequencySeconds    int        `json:"frequency_seconds"`
	CollectionStartedAt *time.Time `json:"collection_started_at,omitempty"`
	LastCollectedAt     *time.Time `json:"last_collected_at,omitempty"`
	CurrentTemperature  *float64   `json:"current_temperature,omitempty"`
	CurrentConsumption  *float64   `json:"current_consumption,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// Measurement - единая запись измерения. Она может прийти из активного датчика
// или из импортированного XLSX-файла, что различается полем SourceType.
type Measurement struct {
	ID             int64      `json:"id"`
	SourceType     string     `json:"source_type"`
	SensorID       *uuid.UUID `json:"sensor_id,omitempty"`
	ImportFileID   *uuid.UUID `json:"import_file_id,omitempty"`
	SensorCode     string     `json:"sensor_code"`
	MeasuredAt     time.Time  `json:"measured_at"`
	TemperatureC   float64    `json:"temperature_c"`
	ConsumptionKWh float64    `json:"consumption_kwh"`
	CreatedAt      time.Time  `json:"created_at"`
}

// ImportFile хранит метаданные успешно загруженного XLSX-файла; сами строки
// файла лежат в таблице measurements и связаны через ImportFileID.
type ImportFile struct {
	ID               uuid.UUID `json:"id"`
	OriginalFilename string    `json:"original_filename"`
	StoredFilename   *string   `json:"stored_filename,omitempty"`
	RowsCount        int       `json:"rows_count"`
	ImportedAt       time.Time `json:"imported_at"`
	Status           string    `json:"status"`
	ErrorMessage     *string   `json:"error_message,omitempty"`
}

// AnalysisRun фиксирует краткий результат анализа, чтобы история расчетов
// оставалась в базе независимо от скачанных файлов.
type AnalysisRun struct {
	ID          uuid.UUID `json:"id"`
	Mode        string    `json:"mode"`
	PeriodFrom  time.Time `json:"period_from"`
	PeriodTo    time.Time `json:"period_to"`
	PointsCount int       `json:"points_count"`
	Correlation *float64  `json:"correlation,omitempty"`
	RegressionA *float64  `json:"regression_a,omitempty"`
	RegressionB *float64  `json:"regression_b,omitempty"`
	RSquared    *float64  `json:"r_squared,omitempty"`
	Summary     string    `json:"summary"`
	CreatedAt   time.Time `json:"created_at"`
}

// ExportFile описывает физический XLSX-файл, созданный в каталоге exports.
type ExportFile struct {
	ID         uuid.UUID  `json:"id"`
	AnalysisID *uuid.UUID `json:"analysis_id,omitempty"`
	Filename   string     `json:"filename"`
	Format     string     `json:"format"`
	FilePath   string     `json:"file_path"`
	CreatedAt  time.Time  `json:"created_at"`
}

// MeasurementFilter задает условия выборки измерений для таблиц, графиков,
// анализа и экспорта.
type MeasurementFilter struct {
	SensorIDs     []uuid.UUID
	ImportFileIDs []uuid.UUID
	SourceType    string
	From          *time.Time
	To            *time.Time
	Limit         int
	Offset        int
}
