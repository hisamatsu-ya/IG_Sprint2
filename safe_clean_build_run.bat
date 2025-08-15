@echo off
setlocal enableextensions enabledelayedexpansion

REM =========================================
REM Config (можно править под себя)
REM =========================================
set "COMPOSE_FILE=docker-compose.yml"
set "COMPOSE_PROJECT_NAME=ig_sprint2"
set "HEALTH_URL=http://localhost:8000/api/events/health"

echo.
echo ==== CinemaAbyss: SAFE CLEAN + BUILD + UP ====
echo Project: %COMPOSE_PROJECT_NAME%
echo Compose: %COMPOSE_FILE%
echo.

REM =========================================
REM 1) Остановить и убрать проект
REM =========================================
echo [1/6] docker compose down -v --remove-orphans
docker compose down -v --remove-orphans
if errorlevel 1 (
  echo.
  echo [ERROR] docker compose down failed.
  pause
  exit /b 1
)

REM =========================================
REM 2) Удалить контейнеры с явными именами (если остались)
REM =========================================
echo [2/6] Removing lingering containers: cinemaabyss-*
for /f "tokens=*" %%i in ('docker ps -a --format "{{.Names}}" ^| findstr /i "^cinemaabyss-"') do (
  echo   - rm %%i
  docker rm -f "%%i" >nul 2>&1
)

REM =========================================
REM 3) Удалить образы текущего проекта (repo начинается с ig_sprint2-)
REM =========================================
echo [3/6] Removing project images: ig_sprint2-*
for /f "tokens=1,2" %%I in ('docker images --format "{{.Repository}} {{.ID}}" ^| findstr /i "^ig_sprint2-"') do (
  echo   - rmi %%I (%%J)
  docker rmi -f "%%J" >nul 2>&1
)

REM =========================================
REM 4) Удалить сети проекта (если остались)
REM =========================================
echo [4/6] Removing project networks (matching *cinemaabyss-network)
for /f "tokens=*" %%N in ('docker network ls --format "{{.Name}}" ^| findstr /i "cinemaabyss-network"') do (
  echo   - network rm %%N
  docker network rm "%%N" >nul 2>&1
)

REM =========================================
REM 5) Удалить тома проекта (префикс %COMPOSE_PROJECT_NAME%_)
REM =========================================
echo [5/6] Removing project volumes (prefix %COMPOSE_PROJECT_NAME%_)
for /f "tokens=*" %%V in ('docker volume ls --format "{{.Name}}" ^| findstr /i "^%COMPOSE_PROJECT_NAME%_"') do (
  echo   - volume rm %%V
  docker volume rm "%%V" >nul 2>&1
)

REM =========================================
REM 6) Сборка и запуск
REM =========================================
echo [6/6] docker compose build --no-cache
docker compose build --no-cache
if errorlevel 1 (
  echo.
  echo [ERROR] Build failed. See output above.
  pause
  exit /b 1
)

echo.
echo === Starting stack (detached) ===
docker compose up -d
if errorlevel 1 (
  echo.
  echo [ERROR] docker compose up failed.
  pause
  exit /b 1
)

echo.
echo === Current services ===
docker compose ps

echo.
echo === Quick health check ===
curl -s "%HEALTH_URL%" || echo (health check skipped or failed)

echo.
echo Done. Press any key to close this window.
pause
endlocal
