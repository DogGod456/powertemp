// API_BASE указывает на backend. В Docker/nginx запросы идут через /api,
// а при разработке можно переопределить адрес через VITE_API_URL.
export const API_BASE = import.meta.env.VITE_API_URL || '/api';

export type Sensor = {
  id: string;
  code: string;
  name: string;
  is_active: boolean;
  frequency_seconds: number;
  collection_started_at?: string;
  last_collected_at?: string;
  current_temperature?: number;
  current_consumption?: number;
  created_at: string;
  updated_at: string;
};

export type ImportFile = {
  id: string;
  original_filename: string;
  rows_count: number;
  imported_at: string;
  status: string;
};

export type ExportFile = {
  id: string;
  filename: string;
  format: string;
  file_path: string;
  created_at: string;
};

export type Measurement = {
  id: number;
  source_type: 'sensor' | 'file';
  sensor_id?: string;
  import_file_id?: string;
  sensor_code: string;
  measured_at: string;
  temperature_c: number;
  consumption_kwh: number;
  created_at: string;
};

export type AnalysisSummary = {
  source_label: string;
  points_count: number;
  min_temperature: number;
  max_temperature: number;
  avg_temperature: number;
  min_consumption: number;
  max_consumption: number;
  avg_consumption: number;
  correlation: number;
  regression_a: number;
  regression_b: number;
  r_squared: number;
  regression_equation: string;
  interpretation: string;
  insufficient_data: boolean;
  insufficient_data_text?: string;
};

export type Forecast = {
  temperature_c: number;
  predicted_consumption_kwh: number;
};

export type ChartPoint = {
  measured_at: string;
  sensor_code: string;
  temperature_c: number;
  consumption_kwh: number;
  predicted_consumption_kwh: number;
};

export type AnalysisResponse = {
  mode: string;
  period_from: string;
  period_to: string;
  results: AnalysisSummary[];
  forecasts: Forecast[];
  points: ChartPoint[];
  warnings: string[];
};

export type AnalysisRequest = {
  mode: string;
  sensor_ids: string[];
  import_file_ids: string[];
  from: string;
  to: string;
  forecast_temperatures: number[];
};

// request - общий wrapper над fetch: выставляет JSON-заголовки, разбирает ответ
// и превращает ошибки API в Error с понятным текстом.
async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: init?.body instanceof FormData ? undefined : { 'Content-Type': 'application/json', ...(init?.headers || {}) },
    ...init
  });
  const text = await res.text();
  let data: any = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = text;
  }
  if (!res.ok) {
    const message = data?.error || data?.message || 'Ошибка запроса';
    const details = data?.errors ? `\n${data.errors.join('\n')}` : '';
    throw new Error(message + details);
  }
  return data as T;
}

// asArray защищает UI от null/undefined/объекта там, где страницы ожидают
// массив данных.
const asArray = <T>(value: unknown): T[] => Array.isArray(value) ? value as T[] : [];

// normalizeAnalysis гарантирует, что блоки результата анализа всегда массивы.
// Это упрощает графики и карточки, особенно при пустых выборках.
function normalizeAnalysis(value: AnalysisResponse): AnalysisResponse {
  return {
    ...value,
    results: asArray<AnalysisSummary>((value as any)?.results),
    forecasts: asArray<Forecast>((value as any)?.forecasts),
    points: asArray<ChartPoint>((value as any)?.points),
    warnings: asArray<string>((value as any)?.warnings)
  };
}

// api группирует все backend endpoints, чтобы страницы не собирали URL и fetch
// параметры вручную.
export const api = {
  dashboard: () => request<Record<string, unknown>>('/dashboard'),
  sensors: async () => asArray<Sensor>(await request<unknown>('/sensors')),
  createSensor: (body: { name?: string; frequency_seconds?: number }) => request<Sensor>('/sensors', { method: 'POST', body: JSON.stringify(body) }),
  updateSensor: (id: string, body: { name?: string; frequency_seconds?: number }) => request<Sensor>(`/sensors/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
  startSensor: (id: string) => request<Sensor>(`/sensors/${id}/start`, { method: 'POST' }),
  stopSensor: (id: string) => request<Sensor>(`/sensors/${id}/stop`, { method: 'POST' }),
  deleteSensor: (id: string) => request<{ status: string }>(`/sensors/${id}`, { method: 'DELETE' }),
  clearSensorMeasurements: (id: string) => request<{ status: string }>(`/sensors/${id}/measurements`, { method: 'DELETE' }),
  imports: async () => asArray<ImportFile>(await request<unknown>('/imports')),
  deleteImport: (id: string) => request<{ status: string }>(`/imports/${id}`, { method: 'DELETE' }),
  uploadImport: async (file: File) => {
    const form = new FormData();
    form.append('file', file);
    return request<{ import: ImportFile; rows: number }>('/imports', { method: 'POST', body: form });
  },
  measurements: async (query = '') => asArray<Measurement>(await request<unknown>(`/measurements${query}`)),
  runAnalysis: async (body: AnalysisRequest) => normalizeAnalysis(await request<AnalysisResponse>('/analysis/run', { method: 'POST', body: JSON.stringify(body) })),
  createExport: async (body: AnalysisRequest) => {
    const res = await request<{ export: ExportFile; analysis: AnalysisResponse }>('/exports', { method: 'POST', body: JSON.stringify({ format: 'xlsx', ...body }) });
    return { ...res, analysis: normalizeAnalysis(res.analysis) };
  },
  exports: async () => asArray<ExportFile>(await request<unknown>('/exports')),
  downloadExportUrl: (id: string) => `${API_BASE}/exports/${id}/download`,
  liveUrl: () => `${API_BASE}/live/events`
};
