import React from 'react';

// Card - базовая визуальная оболочка для повторяющихся панелей интерфейса.
export function Card({ children, className = '' }: { children: React.ReactNode; className?: string }) {
  return <div className={`glass rounded-[2rem] p-5 shadow-glow ${className}`}>{children}</div>;
}

// StatCard показывает один числовой показатель на dashboard.
export function StatCard({ label, value, hint }: { label: string; value: React.ReactNode; hint?: string }) {
  return (
    <Card>
      <div className="muted text-sm">{label}</div>
      <div className="mt-2 text-3xl font-black">{value}</div>
      {hint && <div className="muted mt-2 text-xs">{hint}</div>}
    </Card>
  );
}

// EmptyState используется на страницах, где данных еще нет или выбранный фильтр
// ничего не вернул.
export function EmptyState({ title, text }: { title: string; text: string }) {
  return (
    <div className="rounded-3xl border border-dashed border-white/20 p-8 text-center">
      <div className="text-lg font-bold">{title}</div>
      <div className="muted mt-2 text-sm">{text}</div>
    </div>
  );
}

// ErrorBox выводит серверные и клиентские ошибки в едином формате.
export function ErrorBox({ error }: { error?: string }) {
  if (!error) return null;
  return <div className="rounded-2xl border border-red-500/30 bg-red-500/10 p-4 text-sm text-red-200 whitespace-pre-wrap">{error}</div>;
}

// formatNumber аккуратно обрабатывает пустые значения, чтобы карточки не
// показывали NaN.
export function formatNumber(value: unknown, digits = 2) {
  if (typeof value !== 'number' || Number.isNaN(value)) return '—';
  return value.toFixed(digits);
}

// toInputDateTime переводит Date в формат, который принимает input datetime-local.
export function toInputDateTime(date: Date) {
  const pad = (n: number) => String(n).padStart(2, '0');
  const yyyy = date.getFullYear();
  const mm = pad(date.getMonth() + 1);
  const dd = pad(date.getDate());
  const hh = pad(date.getHours());
  const mi = pad(date.getMinutes());
  return `${yyyy}-${mm}-${dd}T${hh}:${mi}`;
}

// fromInputDateTime превращает локальное значение datetime-local в ISO-строку
// для backend API.
export function fromInputDateTime(value: string) {
  const date = new Date(value);
  if (!value || Number.isNaN(date.getTime())) {
    return new Date().toISOString();
  }
  return date.toISOString();
}


// ErrorBoundary изолирует падение одной страницы: пользователь может перейти в
// другой раздел, а состояние ошибки сбрасывается при смене pageKey.
export class ErrorBoundary extends React.Component<{ children: React.ReactNode; pageKey?: string }, { error?: Error }> {
  constructor(props: { children: React.ReactNode; pageKey?: string }) {
    super(props);
    this.state = {};
  }

  static getDerivedStateFromError(error: Error) {
    return { error };
  }

  componentDidUpdate(prevProps: { pageKey?: string }) {
    if (prevProps.pageKey !== this.props.pageKey && this.state.error) {
      this.setState({ error: undefined });
    }
  }

  render() {
    if (this.state.error) {
      return (
        <div className="rounded-[2rem] border border-red-500/30 bg-red-500/10 p-6 text-red-100">
          <div className="text-xl font-black">Страница временно недоступна</div>
          <div className="mt-2 text-sm opacity-90">{this.state.error.message}</div>
          <div className="mt-4 text-xs opacity-70">Перейди на другую страницу или обнови приложение.</div>
        </div>
      );
    }
    return this.props.children;
  }
}
