import React, { useEffect, useMemo, useState } from 'react';
import { ArrowLeft, ArrowRight, Download, FileSpreadsheet, Minus, PlayCircle, Plus, RotateCcw, Waves } from 'lucide-react';
import { CartesianGrid, ComposedChart, Legend, Line, LineChart, ResponsiveContainer, Scatter, Tooltip, XAxis, YAxis } from 'recharts';
import { api, AnalysisRequest, AnalysisResponse, ChartPoint, ImportFile, Sensor } from '../api';
import { Card, EmptyState, ErrorBox, formatNumber, fromInputDateTime, toInputDateTime } from '../components';

const chartStrokes = ['var(--accent)', 'var(--accent2)', '#34d399', '#f472b6', '#fbbf24', '#fb7185', '#a78bfa', '#22d3ee'];

// AnalysisPage собирает параметры расчета, выбранные источники и прогнозные
// температуры, затем отображает статистику, графики и кнопку XLSX-экспорта.
export default function AnalysisPage() {
  const [sensors, setSensors] = useState<Sensor[]>([]);
  const [imports, setImports] = useState<ImportFile[]>([]);
  const [selectedSensors, setSelectedSensors] = useState<string[]>([]);
  const [selectedImports, setSelectedImports] = useState<string[]>([]);
  const [mode, setMode] = useState('combined');
  const [from, setFrom] = useState(() => toInputDateTime(new Date(Date.now() - 60 * 60 * 1000)));
  const [to, setTo] = useState(() => toInputDateTime(new Date()));
  const [forecastInput, setForecastInput] = useState('-10, 0, 10, 20');
  const [result, setResult] = useState<AnalysisResponse | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  // loadSources загружает оба типа источников, которые пользователь может
  // включить в общий, раздельный или сравнительный анализ.
  const loadSources = async () => {
    const [sensorList, importList] = await Promise.all([api.sensors(), api.imports()]);
    setSensors(sensorList);
    setImports(importList);
  };

  useEffect(() => { loadSources().catch((e) => setError((e as Error).message)); }, []);

  // requestBody является единственным объектом параметров для анализа и
  // экспорта, поэтому оба действия работают по одной и той же выборке.
  const requestBody = useMemo<AnalysisRequest>(() => ({
    mode,
    sensor_ids: selectedSensors,
    import_file_ids: selectedImports,
    from: fromInputDateTime(from),
    to: fromInputDateTime(to),
    forecast_temperatures: forecastInput.split(',').map((x) => Number(x.trim())).filter((x) => !Number.isNaN(x))
  }), [mode, selectedSensors, selectedImports, from, to, forecastInput]);

  // run отправляет расчет на backend и сохраняет нормализованный результат для
  // карточек, таблиц и графиков.
  const run = async () => {
    setLoading(true);
    setError('');
    try {
      setResult(await api.runAnalysis(requestBody));
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  };

  // exportFile просит backend заново посчитать анализ по текущим параметрам,
  // записать XLSX и открыть URL скачивания.
  const exportFile = async () => {
    setLoading(true);
    setError('');
    try {
      const res = await api.createExport(requestBody);
      window.open(api.downloadExportUrl(res.export.id), '_blank');
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  };

  const selectedSourcesCount = selectedSensors.length + selectedImports.length;

  return (
    <div className="space-y-5">
      <ErrorBox error={error} />
      <Card>
        <div className="grid gap-5 xl:grid-cols-4">
          <div>
            <label className="muted text-sm">Режим анализа</label>
            <select className="input mt-1" value={mode} onChange={(e) => setMode(e.target.value)}>
              <option value="combined">Общий анализ</option>
              <option value="separate">Раздельный анализ</option>
              <option value="comparison">Сравнительный анализ</option>
            </select>
          </div>
          <div>
            <label className="muted text-sm">С даты</label>
            <input className="input mt-1" type="datetime-local" value={from} onChange={(e) => setFrom(e.target.value)} />
          </div>
          <div>
            <label className="muted text-sm">По дату</label>
            <input className="input mt-1" type="datetime-local" value={to} onChange={(e) => setTo(e.target.value)} />
          </div>
          <div>
            <label className="muted text-sm">Прогнозные температуры</label>
            <input className="input mt-1" value={forecastInput} onChange={(e) => setForecastInput(e.target.value)} />
          </div>
        </div>

        <div className="mt-5 grid gap-5 xl:grid-cols-2">
          <SourceList
            title="Датчики"
            subtitle="Живые источники, которые пишут измерения в БД"
            items={sensors.map((s) => ({
              id: s.id,
              title: s.name,
              meta: s.code,
              detail: s.is_active ? `работает, сбор каждые ${s.frequency_seconds} сек.` : 'выключен',
              badge: s.is_active ? 'Активен' : 'Выкл',
              active: s.is_active
            }))}
            selected={selectedSensors}
            setSelected={setSelectedSensors}
          />
          <SourceList
            title="Импортированные файлы"
            subtitle="Загруженные XLSX-файлы можно анализировать как отдельные источники"
            items={imports.map((i) => ({
              id: i.id,
              title: i.original_filename,
              meta: `${i.rows_count} строк`,
              detail: `импорт: ${new Date(i.imported_at).toLocaleString('ru-RU')}`,
              badge: i.status || 'success',
              active: true
            }))}
            selected={selectedImports}
            setSelected={setSelectedImports}
          />
        </div>

        <div className="mt-5 flex flex-wrap items-center gap-3">
          <button className="btn btn-primary" disabled={loading} onClick={run}><PlayCircle size={18} /> Рассчитать</button>
          <button className="btn btn-secondary" disabled={loading || !result} onClick={exportFile}><Download size={18} /> Скачать XLSX</button>
          <span className="muted text-sm">Выбрано источников: {selectedSourcesCount}</span>
        </div>
      </Card>

      {result ? <AnalysisResult result={result} /> : <EmptyState title="Анализ еще не выполнен" text="Выбери источники, период и нажми «Рассчитать»." />}
    </div>
  );
}

type SourceItem = { id: string; title: string; meta: string; detail: string; badge: string; active: boolean };

// SourceList отображает выбираемые источники данных. Один компонент используется
// и для live-датчиков, и для импортированных XLSX-файлов.
function SourceList({ title, subtitle, items, selected, setSelected }: { title: string; subtitle: string; items: SourceItem[]; selected: string[]; setSelected: (v: string[]) => void }) {
  const toggle = (id: string) => setSelected(selected.includes(id) ? selected.filter((x) => x !== id) : [...selected, id]);
  const allSelected = items.length > 0 && selected.length === items.length;
  return (
    <div className="rounded-[1.75rem] border border-white/10 bg-white/[0.03] p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-2 text-lg font-black"><FileSpreadsheet size={18} style={{ color: 'var(--accent)' }} /> {title}</div>
          <p className="muted mt-1 text-xs">{subtitle}</p>
        </div>
        <div className="flex gap-2 text-xs">
          <button className="rounded-xl border border-white/10 px-3 py-1 hover:bg-white/10" onClick={() => setSelected(allSelected ? [] : items.map((x) => x.id))}>{allSelected ? 'снять все' : 'выбрать все'}</button>
          <span className="rounded-xl bg-white/10 px-3 py-1">{selected.length}/{items.length}</span>
        </div>
      </div>
      <div className="mt-4 max-h-60 space-y-2 overflow-auto pr-1">
        {items.length === 0 ? <div className="muted rounded-2xl border border-dashed border-white/15 p-4 text-sm">Нет источников</div> : items.map((item) => {
          const checked = selected.includes(item.id);
          return (
            <button key={item.id} type="button" onClick={() => toggle(item.id)} className={`w-full rounded-2xl border p-3 text-left transition ${checked ? 'border-[color:var(--accent)] bg-[color-mix(in_srgb,var(--accent)_16%,transparent)]' : 'border-white/10 bg-white/5 hover:bg-white/10'}`}>
              <div className="flex items-center justify-between gap-3">
                <div className="min-w-0">
                  <div className="flex items-center gap-2 font-bold"><span className={`h-2.5 w-2.5 rounded-full ${item.active ? 'bg-emerald-400' : 'bg-slate-400'}`} /> <span className="truncate">{item.title}</span></div>
                  <div className="muted mt-1 text-xs">{item.meta} · {item.detail}</div>
                </div>
                <div className="flex shrink-0 items-center gap-2">
                  <span className="rounded-xl bg-white/10 px-2 py-1 text-xs">{item.badge}</span>
                  <input type="checkbox" readOnly checked={checked} />
                </div>
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}

// AnalysisResult собирает все блоки результата: метрики, источники, графики,
// прогноз и итоговый вывод.
function AnalysisResult({ result }: { result: AnalysisResponse }) {
  const results = Array.isArray(result.results) ? result.results : [];
  const forecasts = Array.isArray(result.forecasts) ? result.forecasts : [];
  const points = Array.isArray(result.points) ? result.points : [];
  const warnings = Array.isArray(result.warnings) ? result.warnings : [];
  const primary = results[0];
  return (
    <div className="space-y-5">
      {warnings.length > 0 && <ErrorBox error={warnings.join('\n')} />}

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <Metric label="Записей" value={primary?.points_count ?? 0} />
        <Metric label="Корреляция Пирсона" value={formatNumber(primary?.correlation, 4)} />
        <Metric label="R²" value={formatNumber(primary?.r_squared, 4)} />
        <Metric label="Уравнение" value={primary?.regression_equation || '—'} small />
      </div>

      <SourcesResultCard result={result} />

      <ScatterRegressionChart points={points} />
      <TimeModelChart points={points} />

      <Card>
        <h2 className="text-xl font-black">Прогноз</h2>
        <div className="mt-4 grid gap-3 md:grid-cols-4">
          {forecasts.length === 0 ? <div className="muted text-sm">Для прогноза недостаточно данных.</div> : forecasts.map((f) => (
            <div key={f.temperature_c} className="rounded-3xl bg-white/5 p-4">
              <div className="muted text-xs">{f.temperature_c} °C</div>
              <div className="mt-1 text-2xl font-black">{formatNumber(f.predicted_consumption_kwh)} кВт·ч</div>
            </div>
          ))}
        </div>
      </Card>

      <ConclusionCard result={result} />
    </div>
  );
}

// SourcesResultCard показывает summary по каждому источнику. В раздельном и
// сравнительном режимах каждая строка рассчитана backend отдельно.
function SourcesResultCard({ result }: { result: AnalysisResponse }) {
  const results = Array.isArray(result.results) ? result.results : [];
  return (
    <Card>
      <h2 className="text-xl font-black">Источники данных и результаты</h2>
      <p className="muted mt-1 text-sm">Блок показывает, какие наборы данных реально участвовали в анализе. В режиме «Раздельный» и «Сравнительный» каждая строка считается отдельно.</p>
      <div className="mt-4 grid gap-3">
        {results.length === 0 ? (
          <div className="muted rounded-3xl border border-dashed border-white/15 p-6 text-sm">Нет результатов. Выбери источник с измерениями и нажми «Рассчитать».</div>
        ) : results.map((r) => (
          <div key={r.source_label} className="rounded-3xl border border-white/10 bg-white/[0.04] p-4">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <div className="text-lg font-black">{r.source_label}</div>
                <div className="muted mt-1 text-xs">{r.points_count} записей · средняя температура {formatNumber(r.avg_temperature)} °C · среднее потребление {formatNumber(r.avg_consumption)} кВт·ч</div>
              </div>
              <div className="flex flex-wrap gap-2 text-sm">
                <span className="rounded-2xl bg-white/10 px-3 py-2">r: {r.insufficient_data ? '—' : formatNumber(r.correlation, 4)}</span>
                <span className="rounded-2xl bg-white/10 px-3 py-2">R²: {r.insufficient_data ? '—' : formatNumber(r.r_squared, 4)}</span>
              </div>
            </div>
            <div className="mt-3 grid gap-3 lg:grid-cols-[1fr_2fr]">
              <div className="rounded-2xl bg-black/10 p-3 text-sm">
                <div className="muted text-xs">Регрессия</div>
                <div className="mt-1 font-semibold">{r.insufficient_data ? r.insufficient_data_text : r.regression_equation}</div>
              </div>
              <div className="rounded-2xl bg-black/10 p-3 text-sm">
                <div className="muted text-xs">Интерпретация</div>
                <div className="mt-1">{r.interpretation || r.insufficient_data_text || '—'}</div>
              </div>
            </div>
          </div>
        ))}
      </div>
    </Card>
  );
}

// ScatterRegressionChart строит диаграмму рассеяния "температура-потребление"
// и линию линейной регрессии, рассчитанную backend.
function ScatterRegressionChart({ points }: { points: ChartPoint[] }) {
  const sorted = useMemo(() => [...points].sort((a, b) => a.temperature_c - b.temperature_c), [points]);
  return (
    <ZoomableCard title="Диаграмма рассеяния и линия регрессии" hint="Ось X — температура, ось Y — потребление. Точки — фактические измерения, линия — расчетная линейная модель." data={sorted} defaultWindow={600}>
      {(data, range, setRange) => (
        <ResponsiveContainer width="100%" height="100%">
          <ComposedChart data={data} margin={{ top: 10, right: 28, bottom: 20, left: 8 }}>
            <CartesianGrid strokeDasharray="3 3" opacity={0.25} />
            <XAxis type="number" dataKey="temperature_c" name="Температура" domain={['dataMin', 'dataMax']} tickFormatter={(v) => `${formatNumber(Number(v), 1)}°C`} />
            <YAxis type="number" dataKey="consumption_kwh" name="Потребление" tickFormatter={(v) => `${formatNumber(Number(v), 0)}`} />
            <Tooltip formatter={(value: any, name: any) => [formatNumber(Number(value), 2), name]} labelFormatter={(value) => `Температура: ${formatNumber(Number(value), 2)} °C`} />
            <Legend />
            <Scatter dataKey="consumption_kwh" name="Факт, кВт·ч" fill="var(--accent)" isAnimationActive={false} />
            <Line type="monotone" dataKey="predicted_consumption_kwh" name="Линия регрессии" dot={false} stroke="var(--accent2)" strokeWidth={3} isAnimationActive={false} />
          </ComposedChart>
        </ResponsiveContainer>
      )}
    </ZoomableCard>
  );
}

// TimeModelChart показывает фактическое потребление во времени по источникам и
// прогнозную линию модели на тех же точках.
function TimeModelChart({ points }: { points: ChartPoint[] }) {
  const { rows, series } = useMemo(() => buildTimeSeries(points), [points]);
  return (
    <ZoomableCard title="Динамика фактического и прогнозного потребления" hint="Для нескольких датчиков фактические линии разделены по источникам, чтобы график не соединял разные датчики одной ломаной." data={rows} defaultWindow={600}>
      {(data, range, setRange) => (
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={data} margin={{ top: 10, right: 28, bottom: 20, left: 8 }}>
            <CartesianGrid strokeDasharray="3 3" opacity={0.25} />
            <XAxis dataKey="measured_at" minTickGap={32} tick={{ fontSize: 11 }} />
            <YAxis tickFormatter={(v) => `${formatNumber(Number(v), 0)}`} />
            <Tooltip formatter={(value: any, name: any) => [formatNumber(Number(value), 2), name]} />
            <Legend />
            {series.map((s, idx) => (
              <Line key={s.key} type="monotone" dataKey={s.key} name={`Факт ${s.label}`} dot={false} connectNulls stroke={chartStrokes[idx % chartStrokes.length]} strokeWidth={2} isAnimationActive={false} />
            ))}
            <Line type="monotone" dataKey="predicted_consumption_kwh" name="Прогноз модели" dot={false} connectNulls stroke="var(--accent2)" strokeWidth={3} isAnimationActive={false} />
          </LineChart>
        </ResponsiveContainer>
      )}
    </ZoomableCard>
  );
}

// ZoomableCard добавляет к графикам общую механику масштабирования, сдвига и
// выбора видимого окна через range-slider.
function ZoomableCard<T>({ title, hint, data, defaultWindow, children }: { title: string; hint: string; data: T[]; defaultWindow: number; children: (visibleData: T[], range: Range, setRange: (r: Range) => void) => React.ReactNode }) {
  const [range, setRange] = useState<Range>(() => initialRange(data.length, defaultWindow));

  useEffect(() => {
    setRange(initialRange(data.length, defaultWindow));
  }, [data.length, defaultWindow]);

  const safeRange = clampRange(range, data.length);
  const visibleData = data.slice(safeRange.start, safeRange.end + 1);

  const zoom = (factor: number) => setRange((r) => zoomRange(r, data.length, factor));
  const pan = (direction: -1 | 1) => setRange((r) => panRange(r, data.length, direction));

  return (
    <Card>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-xl font-black">{title}</h2>
          <p className="muted mt-1 text-sm">{hint}</p>
        </div>
        <ChartControls disabled={data.length < 2} onZoomIn={() => zoom(0.6)} onZoomOut={() => zoom(1.6)} onLeft={() => pan(-1)} onRight={() => pan(1)} onReset={() => setRange(initialRange(data.length, defaultWindow))} />
      </div>
      {data.length === 0 ? <div className="mt-6"><EmptyState title="Нет точек для графика" text="Выбери период, где есть измерения." /></div> : (
        <>
          <div className="mt-4 h-[520px]">{children(visibleData, safeRange, setRange)}</div>
          <RangeSlider range={safeRange} setRange={setRange} total={data.length} />
        </>
      )}
    </Card>
  );
}

// ChartControls - кнопки управления видимым окном графика.
function ChartControls({ disabled, onZoomIn, onZoomOut, onLeft, onRight, onReset }: { disabled: boolean; onZoomIn: () => void; onZoomOut: () => void; onLeft: () => void; onRight: () => void; onReset: () => void }) {
  return (
    <div className="flex flex-wrap gap-2">
      <button className="btn btn-secondary" disabled={disabled} onClick={onZoomIn}><Plus size={16} /> Приблизить</button>
      <button className="btn btn-secondary" disabled={disabled} onClick={onZoomOut}><Minus size={16} /> Отдалить</button>
      <button className="btn btn-secondary" disabled={disabled} onClick={onLeft}><ArrowLeft size={16} /></button>
      <button className="btn btn-secondary" disabled={disabled} onClick={onRight}><ArrowRight size={16} /></button>
      <button className="btn btn-secondary" disabled={disabled} onClick={onReset}><RotateCcw size={16} /> Сброс</button>
    </div>
  );
}

// RangeSlider двигает текущее окно по полному набору точек без изменения его
// размера.
function RangeSlider({ range, setRange, total }: { range: Range; setRange: (r: Range) => void; total: number }) {
  if (total < 2) return null;
  const windowSize = Math.max(1, range.end - range.start + 1);
  const maxStart = Math.max(0, total - windowSize);
  return (
    <div className="mt-3 flex items-center gap-3">
      <span className="muted text-xs">Окно</span>
      <input className="w-full accent-sky-400" type="range" min={0} max={maxStart} value={Math.min(range.start, maxStart)} onChange={(e) => {
        const start = Number(e.target.value);
        setRange({ start, end: Math.min(total - 1, start + windowSize - 1) });
      }} />
      <span className="muted whitespace-nowrap text-xs">{range.start + 1}–{range.end + 1} из {total}</span>
    </div>
  );
}

// ConclusionCard формирует короткий человекочитаемый вывод из первого summary
// и подсказывает, когда объединение разных источников ослабляет корреляцию.
function ConclusionCard({ result }: { result: AnalysisResponse }) {
  const results = Array.isArray(result.results) ? result.results : [];
  const points = Array.isArray(result.points) ? result.points : [];
  const primary = results[0];
  const sourceCount = new Set(points.map((p) => p.sensor_code)).size;
  const hasManySources = sourceCount > 1 || results.length > 1;
  const absR = Math.abs(primary?.correlation ?? 0);
  const conclusion = primary?.insufficient_data
    ? 'Для выбранных источников недостаточно измерений. Увеличь период анализа или выбери больше источников данных.'
    : [
      `За выбранный период обработано ${primary?.points_count ?? 0} измерений.`,
      primary?.interpretation,
      hasManySources && absR < 0.3 ? 'Связь в общем наборе слабая: это нормально, если объединены разные датчики с разной базовой нагрузкой. Для более корректного сравнения открой режим «Раздельный анализ» или «Сравнительный анализ».' : '',
      'Линейная модель подходит для учебной корреляционно-регрессионной оценки; при реальной эксплуатации ее стоит проверять отдельно для каждого типа объекта и периода.'
    ].filter(Boolean).join(' ');

  return (
    <Card>
      <h2 className="flex items-center gap-2 text-xl font-black"><Waves style={{ color: 'var(--accent)' }} /> Краткий вывод</h2>
      <p className="mt-3 leading-7">{conclusion}</p>
    </Card>
  );
}

// Metric - компактная карточка одного итогового показателя анализа.
function Metric({ label, value, small = false }: { label: string; value: any; small?: boolean }) {
  return <Card><div className="muted text-sm">{label}</div><div className={`mt-2 font-black ${small ? 'text-lg' : 'text-3xl'}`}>{value}</div></Card>;
}

type Range = { start: number; end: number };

// initialRange по умолчанию показывает последние defaultWindow точек.
function initialRange(length: number, defaultWindow: number): Range {
  if (length <= 0) return { start: 0, end: 0 };
  const size = Math.min(length, defaultWindow);
  return { start: Math.max(0, length - size), end: length - 1 };
}

// clampRange удерживает окно графика внутри доступного массива данных.
function clampRange(range: Range, length: number): Range {
  if (length <= 0) return { start: 0, end: 0 };
  const start = Math.max(0, Math.min(range.start, length - 1));
  const end = Math.max(start, Math.min(range.end, length - 1));
  return { start, end };
}

// zoomRange пересчитывает размер окна относительно его центра.
function zoomRange(range: Range, length: number, factor: number): Range {
  if (length <= 1) return range;
  const safe = clampRange(range, length);
  const current = safe.end - safe.start + 1;
  const nextSize = Math.max(10, Math.min(length, Math.round(current * factor)));
  const center = Math.round((safe.start + safe.end) / 2);
  let start = center - Math.floor(nextSize / 2);
  start = Math.max(0, Math.min(start, length - nextSize));
  return { start, end: start + nextSize - 1 };
}

// panRange сдвигает окно на четверть его текущей ширины.
function panRange(range: Range, length: number, direction: -1 | 1): Range {
  if (length <= 1) return range;
  const safe = clampRange(range, length);
  const size = safe.end - safe.start + 1;
  const shift = Math.max(1, Math.round(size * 0.25)) * direction;
  let start = safe.start + shift;
  start = Math.max(0, Math.min(start, length - size));
  return { start, end: start + size - 1 };
}

// safeSeriesKey превращает код датчика в безопасный ключ поля для Recharts.
function safeSeriesKey(code: string) {
  return `actual_${code.replace(/[^a-zA-Z0-9_]/g, '_')}`;
}

// buildTimeSeries раскладывает фактические значения по отдельным сериям, чтобы
// график не соединял разные датчики одной непрерывной линией.
function buildTimeSeries(points: ChartPoint[]) {
  const codes = Array.from(new Set(points.map((p) => p.sensor_code))).slice(0, 8);
  const series = codes.map((code) => ({ code, label: code, key: safeSeriesKey(code) }));
  const rows = points.map((p) => {
    const row: Record<string, any> = {
      measured_at: p.measured_at,
      sensor_code: p.sensor_code,
      predicted_consumption_kwh: p.predicted_consumption_kwh
    };
    row[safeSeriesKey(p.sensor_code)] = p.consumption_kwh;
    return row;
  });
  return { rows, series };
}
