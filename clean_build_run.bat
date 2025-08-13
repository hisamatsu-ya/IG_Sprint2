@echo off
setlocal enableextensions enabledelayedexpansion

REM ============================
REM  Config
REM ============================
set "CLEAN_ALL=1"                 REM 1 = удалить ВСЁ (контейнеры/образы/сети/тома), 0 = чистим только проект
set "COMPOSE_PROJECT_NAME=ig_sprint2"  REM имя проекта compose (можно поменять при желании)
set "COMPOSE_FILE=docker-compose.yml"
set "HEALTH_URL=http://localhost:8000/api/events/health"

echo.
echo ==== CinemaAbyss: CLEAN + BUILD + UP ====
echo Project: %COMPOSE_PROJECT_NAME%
echo Mode   : CLEAN_ALL=%CLEAN_ALL%
echo.

REM ============================
REM  Stop & remove compose stack
REM ============================
echo [1/6] docker compose down -v --remove-orphans
docker compose down -v --remove-orphans

REM ============================
REM  Remove project-named containers (cinemaabyss-*)
REM ============================
echo [2/6] Removing lingering cinemaabyss-* containers (if any)
for /f "tokens=*" %%i in ('docker ps -a --format "{{.Names}}" ^| findstr /i "^cinemaabyss-"') do (
  echo   - rm %%i
  docker rm -f "%%i" >nul 2>&1
)

REM ============================
REM  Global cleanup (optional)
REM ============================
if "%CLEAN_ALL%"=="1" (
  echo [3/6] Removing ALL containers
  for /f "tokens=*" %%i in ('docker ps -aq') do (
    docker rm -f "%%i" >nul 2>&1
  )

  echo [4/6] Removing ALL images
  for /f "tokens=*" %%i in ('docker images -aq') do (
    docker rmi -f "%%i" >nul 2>&1
  )

  echo [5/6] Pruning networks and volumes
  docker network prune -f >nul 2>&1
  docker volume prune -f >nul 2>&1
) else (
  echo [3-5/6] Skipping global prune (CLEAN_ALL=0)
)

REM ============================
REM  Build & Up
REM ============================
echo [6/6] docker compose build --no-cache
docker compose build --no-cache
if errorlevel 1 (
  echo.
  echo Build failed. Aborting.
  exit /b 1
)

echo.
echo === Starting stack (detached) ===
docker compose up -d
if errorlevel 1 (
  echo.
  echo docker compose up failed. Aborting.
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

