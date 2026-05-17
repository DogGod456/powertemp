@echo off
cd /d %~dp0
echo Starting PowerTemp...
docker compose up -d --build
if errorlevel 1 (
  echo.
  echo Docker failed to start PowerTemp. Check the messages above.
  pause
  exit /b 1
)
echo.
echo Waiting for frontend...
timeout /t 8 /nobreak > nul
start "" http://localhost:3000
echo.
echo PowerTemp is running at http://localhost:3000
echo To stop it, run: docker compose down
echo.
pause
