import { useEffect, useState } from 'react';
import { Trash2, UploadCloud } from 'lucide-react';
import { api, ImportFile } from '../api';
import { Card, EmptyState, ErrorBox } from '../components';

// ImportPage загружает XLSX-файлы, показывает ошибки валидации backend и список
// уже импортированных источников для последующего анализа.
export default function ImportPage() {
  const [items, setItems] = useState<ImportFile[]>([]);
  const [file, setFile] = useState<File | null>(null);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const load = async () => {
    try { setItems(await api.imports()); } catch (e) { setItems([]); setError((e as Error).message); }
  };

  useEffect(() => { load(); }, []);

  // upload отправляет файл multipart/form-data; backend сам проверяет структуру
  // и не сохраняет частично ошибочные импорты.
  const upload = async () => {
    if (!file) return;
    setLoading(true);
    setError('');
    setMessage('');
    try {
      const res = await api.uploadImport(file);
      setMessage(`Файл импортирован: ${res.rows} строк`);
      setFile(null);
      await load();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-5">
      <ErrorBox error={error} />
      {message && <div className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 p-4 text-sm text-emerald-200">{message}</div>}
      <Card>
        <h2 className="text-xl font-black">Загрузить XLSX</h2>
        <p className="muted mt-1 text-sm">Обязательные столбцы: measured_at, sensor_code, temperature_c, consumption_kwh.</p>
        <div className="mt-5 grid gap-3 md:grid-cols-[1fr_auto]">
          <input className="input" type="file" accept=".xlsx" onChange={(e) => setFile(e.target.files?.[0] || null)} />
          <button className="btn btn-primary" disabled={!file || loading} onClick={upload}><UploadCloud size={18} /> Импортировать</button>
        </div>
      </Card>

      <Card>
        <h2 className="text-xl font-black">Импортированные источники</h2>
        <div className="mt-5 space-y-3">
          {items.length === 0 ? <EmptyState title="Файлы еще не импортировались" text="Загрузи sample_measurements.xlsx или свой XLSX-файл." /> : items.map((item) => (
            <div key={item.id} className="flex flex-wrap items-center justify-between gap-3 rounded-3xl bg-white/5 p-4">
              <div>
                <div className="font-bold">{item.original_filename}</div>
                <div className="muted text-sm">{item.rows_count} строк · {new Date(item.imported_at).toLocaleString()}</div>
              </div>
              <button className="btn btn-secondary" onClick={async () => { if (confirm('Удалить импорт и все его данные?')) { await api.deleteImport(item.id); await load(); } }}><Trash2 size={16} /> Удалить</button>
            </div>
          ))}
        </div>
      </Card>
    </div>
  );
}
