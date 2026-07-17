@echo off
:: 65001 = UTF-8，936 = GBK，按需切换
chcp 65001 >nul
@echo off
setlocal enabledelayedexpansion
set VERSION=v1.0.0
set SERVICE_NAME=weekly-assistant-web

if "%1"=="" goto help
if "%1"=="build" goto build
if "%1"=="run" goto run
if "%1"=="docker-build" goto docker-build
if "%1"=="clean" goto clean
goto help

:build
echo Building project...
call npm run build -- --mode production
goto :eof

:run
echo Running development server...
call npm run serve
goto :eof

:docker-build
echo Building Docker image...
docker build -t %SERVICE_NAME%:%VERSION% -f ./docker/Dockerfile .
goto :eof

:clean
echo Cleaning dist directory...
if exist dist (
    rmdir /s /q dist
)
goto :eof

:help
echo Usage:
echo   build        - Build the project
echo   run          - Run development server
echo   docker-build - Build Docker image
echo   clean        - Remove dist directory
goto :eof

