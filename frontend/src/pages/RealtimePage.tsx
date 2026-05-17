import { useEffect, useMemo, useState } from 'react';
import { Activity, RefreshCw } from 'lucide-react';
import { CartesianGrid, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { api, Measurement, Sensor } from '../api';
import { Card, ErrorBox, EmptyState, formatNumber } from '../components';

const DEFAULT_WINDOW = 250;

// RealtimePage слушает SSE-канал backend и строит live-график последних
// измерений без постоянного polling самих точек.
export default function RealtimePage() {
  const [points, setPoints] = useState<Measurement[]>([]);
  const [sensors, setSensors] = useState<Sensor[]>([]);
  const [selectedCode, setSelectedCode] = useState('all');
  const [error, setError] = useState('');

  const loadSensors = async () => setSensors(await api.sensors());

  useEffect(() => {
    loadSensors().catch((e) => setError((e as Error).message));
    const id = setInterval(() => loadSensors().catch(() => {}), 2000);
    return () => clearInterval(id);
  }, []);

  // EventSource держит открытое соединение с /live/events. Новые измерения
  // добавляются в локальный буфер, из которого график берет последнее окно.
  useEffect(() => {
    const source = new EventSource(api.liveUrl());
    source.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.type === 'measurement') {
        setPoints((prev) => {
          const next = [...prev, data.payload as Measurement];
          return next.slice(-6000);
        });
      }
    };
    source.onerror = () => setError('Нет соединения с live-каналом. Проверь, запущен ли backend.');
    return () => source.close();
  }, []);

  const visible = useMemo(() => selectedCode === 'all' ? points : points.filter((p) => p.sensor_code === selectedCode), [points, selectedCode]);
  const chartData = useMemo(() => visible.slice(-DEFAULT_WINDOW), [visible]);
  const last = visible[visible.length - 1];

  return (
    <div className="space-y-5">
      <ErrorBox error={error} />
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <div className="muted text-sm">Последняя температура</div>
          <div className="mt-2 text-3xl font-black">{formatNumber(last?.temperature_c)} °C</div>
        </Card>
        <Card>
          <div className="muted text-sm">Последнее потребление</div>
          <div className="mt-2 text-3xl font-black">{formatNumber(last?.consumption_kwh)} кВт·ч</div>
        </Card>
        <Card>
          <div className="muted text-sm">Точек в выбранном источнике</div>
          <div className="mt-2 text-3xl font-black">{visible.length}</div>
        </Card>
      </div>

      <Card>
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h2 className="flex items-center gap-2 text-xl font-black"><Activity style={{ color: 'var(--accent)' }} /> Живой график</h2>
            <p className="muted mt-1 text-sm">График автоматически показывает последние {DEFAULT_WINDOW} точек. Пока датчики включены, новые измерения записываются в БД и сразу появляются здесь.</p>
          </div>
          <div className="flex max-w-xl flex-1 flex-wrap justify-end gap-2">
            <select className="input min-w-56 flex-1" value={selectedCode} onChange={(e) => setSelectedCode(e.target.value)}>
              <option value="all">Все датчики</option>
              {sensors.map((s) => <option key={s.id} value={s.code}>{s.code} — {s.name}</option>)}
            </select>
            <button className="btn btn-secondary" onClick={() => setPoints([])}><RefreshCw size={16} /> Очистить экран</button>
          </div>
        </div>

        {visible.length === 0 ? <div className="mt-8"><EmptyState title="Ожидание данных" text="Включи один или несколько датчиков на странице «Датчики»." /></div> : (
          <>
            <div className="muted mt-4 text-sm">Показано последних {chartData.length} из {visible.length} точек.</div>
            <div className="mt-5 h-[560px] rounded-3xl border border-white/5 bg-black/10 p-2">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={chartData} margin={{ top: 10, right: 28, bottom: 20, left: 8 }}>
                  <CartesianGrid strokeDasharray="3 3" opacity={0.25} />
                  <XAxis dataKey="measured_at" minTickGap={34} tick={{ fontSize: 11 }} />
                  <YAxis yAxisId="left" tickFormatter={(v) => `${formatNumber(Number(v), 1)}°`} />
                  <YAxis yAxisId="right" orientation="right" tickFormatter={(v) => `${formatNumber(Number(v), 0)}`} />
                  <Tooltip formatter={(value: any, name: any) => [formatNumber(Number(value), 2), name]} />
                  <Line yAxisId="left" type="monotone" dataKey="temperature_c" name="Температура, °C" dot={false} stroke="var(--accent)" strokeWidth={2} isAnimationActive={false} />
                  <Line yAxisId="right" type="monotone" dataKey="consumption_kwh" name="Потребление, кВт·ч" dot={false} stroke="var(--accent2)" strokeWidth={2} isAnimationActive={false} />
                </LineChart>
              </ResponsiveContainer>
            </div>
          </>
        )}
      </Card>
    </div>
  );
}
