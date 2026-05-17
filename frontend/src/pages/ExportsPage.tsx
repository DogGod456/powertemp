import { useEffect, useState } from 'react';
import { Download, RefreshCw } from 'lucide-react';
import { api, ExportFile } from '../api';
import { Card, EmptyState, ErrorBox } from '../components';

// ExportsPage показывает историю XLSX-отчетов и дает ссылку на скачивание
// физического файла из каталога exports.
export default function ExportsPage() {
  const [items, setItems] = useState<ExportFile[]>([]);
  const [error, setError] = useState('');

  // load перечитывает список выгрузок после создания новых отчетов на странице
  // анализа или по кнопке обновления.
  const load = async () => {
    try { setError(''); setItems(await api.exports()); } catch (e) { setItems([]); setError((e as Error).message); }
  };

  useEffect(() => { load(); }, []);

  return (
    <div className="space-y-5">
      <ErrorBox error={error} />
      <Card>
        <div className="flex items-center justify-between gap-3">
          <div>
            <h2 className="text-xl font-black">Созданные файлы</h2>
            <p className="muted mt-1 text-sm">Файлы сохраняются в папку exports. Первый лист XLSX совместим с форматом импорта.</p>
          </div>
          <button className="btn btn-secondary" onClick={load}><RefreshCw size={16} /> Обновить</button>
        </div>
        <div className="mt-5 space-y-3">
          {items.length === 0 ? <EmptyState title="Выгрузок пока нет" text="Сначала проведи анализ и нажми «Скачать XLSX»." /> : items.map((item) => (
            <div key={item.id} className="flex flex-wrap items-center justify-between gap-3 rounded-3xl bg-white/5 p-4">
              <div>
                <div className="font-bold">{item.filename}</div>
                <div className="muted text-sm">{item.format.toUpperCase()} · {new Date(item.created_at).toLocaleString()} · {item.file_path}</div>
              </div>
              <a className="btn btn-primary" href={api.downloadExportUrl(item.id)} target="_blank"><Download size={16} /> Скачать</a>
            </div>
          ))}
        </div>
      </Card>
    </div>
  );
}
