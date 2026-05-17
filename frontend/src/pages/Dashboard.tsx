import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { Activity, FileDown, PlugZap, UploadCloud } from 'lucide-react';
import { api, Measurement, Sensor } from '../api';
import { Card, ErrorBox, StatCard, formatNumber } from '../components';

// Dashboard показывает текущее состояние системы: счетчики, активные датчики,
// быстрый сценарий и последнее измерение от live-сенсоров.
export default function Dashboard() {
  const [data, setData] = useState<Record<string, any>>({});
  const [sensors, setSensors] = useState<Sensor[]>([]);
  const [error, setError] = useState('');

  // load обновляет агрегаты и список датчиков параллельно, чтобы dashboard
  // оставался живым без ручного обновления страницы.
  const load = async () => {
    try {
      setError('');
      const [dashboard, sensorList] = await Promise.all([api.dashboard(), api.sensors()]);
      setData(dashboard || {});
      setSensors(Array.isArray(sensorList) ? sensorList : []);
    } catch (e) {
      setError((e as Error).message);
    }
  };

  useEffect(() => {
    load();
    const id = setInterval(load, 1000);
    return () => clearInterval(id);
  }, []);

  const last = data.last_measurement as Measurement | undefined;
  const activeSensors = useMemo(() => sensors.filter((s) => s.is_active), [sensors]);

  return (
    <div className="space-y-5">
      <ErrorBox error={error} />
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <StatCard label="Измерений датчиков" value={String(data.measurements_count ?? 0)} hint="Только данные, собранные датчиками" />
        <StatCard label="Активных датчиков" value={String(data.active_sensors ?? activeSensors.length)} hint="Сейчас собирают данные" />
        <StatCard label="Импортов" value={String(data.imports_count ?? 0)} hint="XLSX источники" />
        <StatCard label="Выгрузок" value={String(data.exports_count ?? 0)} hint="XLSX отчеты" />
      </div>

      <div className="grid gap-5 xl:grid-cols-3">
        <Card className="xl:col-span-2">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-xl font-black">Включенные датчики</h2>
              <p className="muted mt-1 text-sm">Последние значения всех датчиков, которые сейчас собирают данные.</p>
            </div>
            <Activity style={{ color: 'var(--accent)' }} />
          </div>
          {activeSensors.length > 0 ? (
            <div className="mt-6 grid gap-4 md:grid-cols-2 xl:grid-cols-2">
              {activeSensors.map((sensor) => (
                <div key={sensor.id} className="rounded-3xl bg-white/5 p-4">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <div className="text-xl font-black">{sensor.name || sensor.code}</div>
                      <div className="muted text-xs">{sensor.code} · частота {sensor.frequency_seconds} сек.</div>
                    </div>
                    <span className="h-3 w-3 rounded-full bg-emerald-400" />
                  </div>
                  <div className="mt-4 grid grid-cols-3 gap-3">
                    <div>
                      <div className="muted text-xs">Температура</div>
                      <div className="mt-1 text-lg font-black">{formatNumber(sensor.current_temperature)} °C</div>
                    </div>
                    <div>
                      <div className="muted text-xs">Потребление</div>
                      <div className="mt-1 text-lg font-black">{formatNumber(sensor.current_consumption)} кВт·ч</div>
                    </div>
                    <div>
                      <div className="muted text-xs">Последний сбор</div>
                      <div className="mt-1 text-xs font-bold">{sensor.last_collected_at ? new Date(sensor.last_collected_at).toLocaleTimeString() : '—'}</div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="muted mt-8 text-sm">Включенные датчики отсутствуют. Включи один или несколько датчиков на странице «Датчики».</div>
          )}
        </Card>
        <Card>
          <h2 className="text-xl font-black">Быстрый сценарий</h2>
          <div className="mt-5 space-y-4 text-sm">
            <Step icon={<PlugZap size={16} />} text="Включи D-1 и D-2 на странице датчиков" />
            <Step icon={<Activity size={16} />} text="Посмотри live-график в реальном времени" />
            <Step icon={<UploadCloud size={16} />} text="Загрузи XLSX с историей" />
            <Step icon={<FileDown size={16} />} text="Проведи анализ и выгрузи XLSX" />
          </div>
        </Card>
      </div>

      <Card>
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-black">Последнее измерение в системе</h2>
            <p className="muted mt-1 text-sm">Самая новая запись среди данных, собранных датчиками. Импортированные файлы здесь не учитываются.</p>
          </div>
          <Activity style={{ color: 'var(--accent)' }} />
        </div>
        {last ? (
          <div className="mt-6 grid gap-4 md:grid-cols-4">
            <div className="rounded-3xl bg-white/5 p-4"><div className="muted text-xs">Источник</div><div className="mt-1 text-2xl font-black">{last.sensor_code}</div></div>
            <div className="rounded-3xl bg-white/5 p-4"><div className="muted text-xs">Температура</div><div className="mt-1 text-2xl font-black">{formatNumber(last.temperature_c)} °C</div></div>
            <div className="rounded-3xl bg-white/5 p-4"><div className="muted text-xs">Потребление</div><div className="mt-1 text-2xl font-black">{formatNumber(last.consumption_kwh)} кВт·ч</div></div>
            <div className="rounded-3xl bg-white/5 p-4"><div className="muted text-xs">Время</div><div className="mt-1 text-sm font-bold">{new Date(last.measured_at).toLocaleString()}</div></div>
          </div>
        ) : (
          <div className="muted mt-8 text-sm">Пока нет измерений датчиков. Включи один или несколько датчиков.</div>
        )}
      </Card>
    </div>
  );
}

// Step - одна строка демонстрационного сценария на главной странице.
function Step({ icon, text }: { icon: ReactNode; text: string }) {
  return <div className="flex items-center gap-3 rounded-2xl bg-white/5 p-3"><span style={{ color: 'var(--accent)' }}>{icon}</span><span>{text}</span></div>;
}
