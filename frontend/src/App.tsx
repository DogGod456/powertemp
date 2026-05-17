import { Activity, BarChart3, FileDown, Gauge, Home, PlugZap, UploadCloud } from 'lucide-react';
import { useState } from 'react';
import Dashboard from './pages/Dashboard';
import Sensors from './pages/Sensors';
import ImportPage from './pages/ImportPage';
import AnalysisPage from './pages/AnalysisPage';
import RealtimePage from './pages/RealtimePage';
import ExportsPage from './pages/ExportsPage';
import { ErrorBoundary } from './components';

// Приложение небольшое, поэтому вместо полноценного роутера используется
// локальное состояние с именем текущей страницы.
type Page = 'dashboard' | 'sensors' | 'import' | 'analysis' | 'realtime' | 'exports';
type Theme = 'theme-blue' | 'theme-purple' | 'theme-light';

// Навигация задает и порядок пунктов меню, и иконки для desktop/sidebar и
// mobile/select-переключателя.
const nav = [
  { id: 'dashboard', label: 'Dashboard', icon: Home },
  { id: 'sensors', label: 'Датчики', icon: PlugZap },
  { id: 'import', label: 'Импорт', icon: UploadCloud },
  { id: 'analysis', label: 'Анализ', icon: BarChart3 },
  { id: 'realtime', label: 'Реальное время', icon: Activity },
  { id: 'exports', label: 'Выгрузки', icon: FileDown }
] as const;

// App собирает общий layout: боковое меню, мобильный header, верхний заголовок
// и активную страницу внутри ErrorBoundary.
export default function App() {
  const [page, setPage] = useState<Page>('dashboard');
  const [theme, setTheme] = useState<Theme>('theme-blue');

  return (
    <div className={`${theme} app-bg min-h-screen font-sans`}>
      <div className="mx-auto flex min-h-screen max-w-[1500px] gap-6 p-4 lg:p-6">
        <aside className="glass-strong hidden w-72 shrink-0 rounded-[2rem] p-5 lg:block">
          <Brand />
          <nav className="mt-8 space-y-2">
            {nav.map((item) => {
              const Icon = item.icon;
              const active = page === item.id;
              return (
                <button
                  key={item.id}
                  onClick={() => setPage(item.id as Page)}
                  className={`flex w-full items-center gap-3 rounded-2xl px-4 py-3 text-left text-sm font-semibold transition ${active ? 'btn-primary' : 'hover:bg-white/10'}`}
                >
                  <Icon size={18} />
                  {item.label}
                </button>
              );
            })}
          </nav>
          <div className="mt-8 rounded-3xl border border-white/10 p-4">
            <div className="text-xs uppercase tracking-[0.25em] muted">Тема</div>
            <div className="mt-3 grid grid-cols-3 gap-2">
              <button onClick={() => setTheme('theme-blue')} className="h-9 rounded-xl bg-sky-500/70" title="Dark Blue" />
              <button onClick={() => setTheme('theme-purple')} className="h-9 rounded-xl bg-fuchsia-500/70" title="Dark Purple" />
              <button onClick={() => setTheme('theme-light')} className="h-9 rounded-xl bg-slate-100" title="Light" />
            </div>
          </div>
        </aside>

        <main className="flex min-w-0 flex-1 flex-col">
          <header className="glass-strong mb-5 flex flex-wrap items-center justify-between gap-4 rounded-[2rem] p-4 lg:hidden">
            <Brand compact />
            <select className="input w-auto" value={page} onChange={(e) => setPage(e.target.value as Page)}>
              {nav.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}
            </select>
          </header>
          <TopBar page={page} />
          <div className="min-h-0 flex-1"><ErrorBoundary pageKey={page}>{renderPage(page)}</ErrorBoundary></div>
        </main>
      </div>
    </div>
  );
}

// Brand отображает логотип приложения; compact-режим используется в мобильном
// header, где нужен только знак без подписи.
function Brand({ compact = false }: { compact?: boolean }) {
  return (
    <div className="flex items-center gap-3">
      <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-gradient-to-br from-sky-400 to-indigo-500 shadow-glow">
        <Gauge className="text-white" />
      </div>
      {!compact && (
        <div>
          <div className="text-2xl font-black tracking-tight">PowerTemp</div>
          <div className="muted text-xs">energy correlation analytics</div>
        </div>
      )}
    </div>
  );
}

// TopBar показывает заголовок, соответствующий текущему разделу.
function TopBar({ page }: { page: Page }) {
  const titles: Record<Page, string> = {
    dashboard: 'Панель управления',
    sensors: 'Виртуальные датчики',
    import: 'Импорт XLSX',
    analysis: 'Корреляционно-регрессионный анализ',
    realtime: 'Режим реального времени',
    exports: 'История выгрузок'
  };
  return (
    <div className="mb-5">
      <h1 className="text-3xl font-black tracking-tight lg:text-4xl">{titles[page]}</h1>
      <p className="muted mt-1 text-sm">Анализ зависимости температуры воздуха и потребления электроэнергии</p>
    </div>
  );
}

// renderPage сопоставляет ключ страницы с React-компонентом конкретного
// пользовательского сценария.
function renderPage(page: Page) {
  switch (page) {
    case 'dashboard': return <Dashboard />;
    case 'sensors': return <Sensors />;
    case 'import': return <ImportPage />;
    case 'analysis': return <AnalysisPage />;
    case 'realtime': return <RealtimePage />;
    case 'exports': return <ExportsPage />;
  }
}
