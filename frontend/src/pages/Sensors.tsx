import { useEffect, useState } from 'react';
import { Pencil, Play, Plus, RefreshCw, Square, Trash2 } from 'lucide-react';
import { api, Sensor } from '../api';
import { Card, ErrorBox, EmptyState, formatNumber } from '../components';

// Sensors управляет виртуальными датчиками: создание, включение/выключение,
// частота сбора, очистка истории и удаление.
export default function Sensors() {
  const [sensors, setSensors] = useState<Sensor[]>([]);
  const [error, setError] = useState('');
  const [name, setName] = useState('');

  // load периодически синхронизирует UI с backend, чтобы карточки показывали
  // новые текущие значения активных датчиков.
  const load = async () => {
    try {
      setError('');
      setSensors(await api.sensors());
    } catch (e) {
      setError((e as Error).message);
    }
  };

  useEffect(() => {
    load();
    const id = setInterval(load, 2000);
    return () => clearInterval(id);
  }, []);

  const create = async () => {
    await api.createSensor({ name, frequency_seconds: 1 });
    setName('');
    await load();
  };

  return (
    <div className="space-y-5">
      <ErrorBox error={error} />
      <Card>
        <div className="grid gap-3 md:grid-cols-[1fr_auto]">
          <input className="input" placeholder="Название нового датчика, например Датчик котельной" value={name} onChange={(e) => setName(e.target.value)} />
          <button className="btn btn-primary" onClick={create}><Plus size={18} /> Создать датчик</button>
        </div>
      </Card>

      {sensors.length === 0 ? <EmptyState title="Датчики не найдены" text="Создай новый датчик или перезапусти backend для начального заполнения D-1...D-5." /> : (
        <div className="grid gap-4 xl:grid-cols-2">
          {sensors.map((sensor) => <SensorCard key={sensor.id} sensor={sensor} onReload={load} />)}
        </div>
      )}
    </div>
  );
}

// SensorCard инкапсулирует все действия над одним датчиком и локальное состояние
// формы редактирования имени/частоты.
function SensorCard({ sensor, onReload }: { sensor: Sensor; onReload: () => Promise<void> }) {
  const [name, setName] = useState(sensor.name);
  const [freq, setFreq] = useState(sensor.frequency_seconds);
  const [busy, setBusy] = useState(false);

  useEffect(() => { setName(sensor.name); setFreq(sensor.frequency_seconds); }, [sensor.name, sensor.frequency_seconds]);

  // act ставит карточку в busy-состояние, выполняет backend-действие и затем
  // перечитывает датчик из API.
  const act = async (fn: () => Promise<unknown>) => {
    setBusy(true);
    try { await fn(); await onReload(); } finally { setBusy(false); }
  };

  return (
    <Card>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <span className={`h-3 w-3 rounded-full ${sensor.is_active ? 'bg-emerald-400' : 'bg-slate-400'}`} />
            <h2 className="text-2xl font-black">{sensor.name || sensor.code}</h2>
          </div>
          <p className="muted mt-1 text-sm">{sensor.code} · {sensor.is_active ? 'работает и генерирует данные' : 'выключен'}</p>
        </div>
        <div className="flex gap-2">
          {sensor.is_active ? (
            <button className="btn btn-secondary" disabled={busy} onClick={() => act(() => api.stopSensor(sensor.id))}><Square size={16} /> Выкл</button>
          ) : (
            <button className="btn btn-primary" disabled={busy} onClick={() => act(() => api.startSensor(sensor.id))}><Play size={16} /> Вкл</button>
          )}
          <button className="btn btn-secondary" onClick={onReload}><RefreshCw size={16} /></button>
        </div>
      </div>

      <div className="mt-5 grid gap-3 md:grid-cols-2">
        <label className="space-y-1 text-sm"><span className="muted">Название</span><input className="input" value={name} onChange={(e) => setName(e.target.value)} /></label>
        <label className="space-y-1 text-sm"><span className="muted">Частота, секунд</span><input className="input" type="number" min={1} max={3600} value={freq} onChange={(e) => setFreq(Number(e.target.value))} /></label>
      </div>

      <div className="mt-4 grid gap-3 md:grid-cols-3">
        <div className="rounded-3xl bg-white/5 p-4"><div className="muted text-xs">Температура</div><div className="mt-1 text-xl font-black">{formatNumber(sensor.current_temperature)} °C</div></div>
        <div className="rounded-3xl bg-white/5 p-4"><div className="muted text-xs">Потребление</div><div className="mt-1 text-xl font-black">{formatNumber(sensor.current_consumption)} кВт·ч</div></div>
        <div className="rounded-3xl bg-white/5 p-4"><div className="muted text-xs">Последний сбор</div><div className="mt-1 text-xs font-bold">{sensor.last_collected_at ? new Date(sensor.last_collected_at).toLocaleString() : '—'}</div></div>
      </div>

      <div className="mt-5 flex flex-wrap gap-2">
        <button className="btn btn-primary" disabled={busy} onClick={() => act(() => api.updateSensor(sensor.id, { name, frequency_seconds: freq }))}><Pencil size={16} /> Сохранить</button>
        <button className="btn btn-secondary" disabled={busy} onClick={() => confirm('Очистить все измерения датчика? Если датчик включен, новые данные начнут появляться снова.') && act(() => api.clearSensorMeasurements(sensor.id))}>Очистить данные</button>
        <button className="btn btn-secondary" disabled={busy} onClick={() => confirm('Удалить датчик навсегда вместе со всеми измерениями?') && act(() => api.deleteSensor(sensor.id))}><Trash2 size={16} /> Удалить</button>
      </div>
    </Card>
  );
}
