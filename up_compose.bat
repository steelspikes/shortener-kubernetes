@echo off
cd /d "%~dp0"
docker compose down --remove-orphans --rmi local
docker compose up -d --build
