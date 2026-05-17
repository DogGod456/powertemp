# PowerTemp

PowerTemp — локальное веб-приложение для курсовой работы по варианту 9: анализ зависимости между температурой воздуха и потреблением электроэнергии.

## Что умеет

- импорт XLSX;
- проверка структуры и значений файла перед импортом;
- хранение датчиков, импортов, измерений, анализов и экспортов в PostgreSQL;
- 5 датчиков по умолчанию: D-1 ... D-5;
- включение/выключение датчиков;
- изменение частоты сбора данных;
- реалистичная генерация данных виртуальными датчиками;
- live-графики через Server-Sent Events;
- анализ за выбранный период;
- общий и раздельный анализ по выбранным датчикам/файлам;
- корреляция Пирсона;
- линейная регрессия `consumption = a * temperature + b`;
- прогноз потребления по заданной температуре;
- экспорт XLSX в папку `exports`;
- запуск одной командой через Docker Compose.

## Быстрый запуск

1. Запусти Docker Desktop.
2. В корне проекта выполни:

```bash
docker compose up --build
```

Или на Windows дважды нажми:

```bash
start.bat
```

3. Открой приложение:

```text
http://localhost:3000
```

API доступно через frontend-прокси:

```text
http://localhost:3000/api
```

Backend напрямую:

```text
http://localhost:8080/api
```

## Как выключить приложение

Если проект запущен командой `docker compose up --build` в открытом терминале, нажми `Ctrl+C`, дождись остановки контейнеров и затем выполни:

```bash
docker compose down
```

Если запуск был через `start.bat` или окно терминала уже закрыто, открой терминал в корне проекта и выполни:

```bash
docker compose down
```

Эта команда останавливает backend, frontend, PostgreSQL и sensor-simulator. Данные PostgreSQL при этом сохраняются в Docker volume, поэтому после следующего запуска импортированные файлы и измерения останутся на месте.

Если нужно полностью очистить базу данных и начать с пустого состояния, используй:

```bash
docker compose down -v
```

Для запуска без Docker каждый процесс останавливается в своем терминале через `Ctrl+C`: backend API Gateway, sensor-simulator и frontend dev server.

## Формат импорта

XLSX должен содержать обязательные столбцы:

```text
measured_at | sensor_code | temperature_c | consumption_kwh
2026-05-16 12:00:00 | D-1 | 18.4 | 245.7
2026-05-16 12:00:01 | D-1 | 18.5 | 245.1
2026-05-16 12:00:02 | D-2 | 17.9 | 260.3
```

Поддерживаются даты в форматах:

- `2026-05-16 12:00:00`
- `2026-05-16 12:00`
- `2026-05-16T12:00:00Z`
- `16.05.2026 12:00:00`

Если в файле есть ошибка, файл не импортируется целиком. Пользователь получает список ошибок.

## Структура проекта

```text
powertemp/
  backend/
    cmd/
      api-gateway/
      sensor-simulator/
    internal/
      analysis/
      api/
      collector/
      config/
      db/
      domain/
      exportutil/
      importutil/
      realtime/
      simulator/
  frontend/
  samples/
  exports/
  docker-compose.yml
  start.bat
```

## Разработка без Docker

Backend API Gateway:

```bash
cd backend
go run ./cmd/api-gateway
```

Sensor simulator:

```bash
cd backend
go run ./cmd/sensor-simulator
```

Frontend:

```bash
cd frontend
npm install
npm run dev
```

Для локального запуска без Docker нужен PostgreSQL и переменная `DATABASE_URL`.

## Переменные окружения backend

```text
HTTP_ADDR=:8080
DATABASE_URL=postgres://powertemp:powertemp@postgres:5432/powertemp?sslmode=disable
SIMULATOR_URL=http://sensor-simulator:8090
EXPORT_DIR=/app/exports
```

## Демонстрационный сценарий

1. Открыть Dashboard.
2. Перейти в Sensors.
3. Включить D-1 и D-2.
4. Перейти в Realtime и увидеть живой график.
5. Изменить частоту сбора данных.
6. Перейти в Import и загрузить `samples/sample_measurements.xlsx`.
7. Перейти в Analysis.
8. Выбрать датчики/импорт, период и режим анализа.
9. Получить корреляцию, регрессию, R², прогноз и графики.
10. Экспортировать результат в XLSX.
11. Открыть Exports и скачать файл.



