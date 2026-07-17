@echo off
:: 65001 = UTF-8, 936 = GBK, switch as needed
chcp 65001 >nul
@echo off
setlocal enabledelayedexpansion
set VERSION=v1.0.0
set SERVICE_NAME=weekly-assistant
::==================================================
::  Route Table
::==================================================
if /i "%~1"==""          goto :help
if /i "%~1"=="build"     goto :build-local
if /i "%~1"=="linux"     goto :build-linux
if /i "%~1"=="docker"    goto :docker-local
if /i "%~1"=="docker-linux" goto :docker-linux
if /i "%~1"=="clean"     goto :clean

:help
echo Usage:
echo   %~nx0              "Build native platform executable"
echo   %~nx0 linux        "Cross-compile Linux amd64 binary"
echo   %~nx0 docker       "Native binary + docker image"
echo   %~nx0 docker-linux "Linux binary + docker image"
echo   %~nx0 clean        "Clean bin directory"
goto :eof

::==================================================
::  Utility: Build Go binary
::  Params: %1=GOOS  %2=output filename (with extension)
::==================================================
:go-build
cd /d "%~dp0cmd"
go mod tidy || (echo [ERROR] go mod tidy failed & exit /b 1)
set CGO_ENABLED=0
set GOOS=%~1
set GOARCH=amd64
go build -trimpath -ldflags="-s -w" -o "%~dp0bin\%~2" || (echo [ERROR] Build failed & exit /b 1)
echo [BUILD] Generated %~2 completed
goto :eof

::==================================================
::  1. Native platform
::==================================================
:build-local
call :go-build windows %SERVICE_NAME%.exe
goto :eof

::==================================================
::  2. Linux cross-compile
::==================================================
:build-linux
call :go-build linux %SERVICE_NAME%.bin
goto :eof

::==================================================
::  3. Native platform + docker image
::==================================================
:docker-local
call :build-local
call :docker-build windows Dockerfile %SERVICE_NAME%.exe
goto :eof

::==================================================
::  4. Linux cross-compile + docker image
::==================================================
:docker-linux
call :build-linux
call :docker-build linux Dockerfile-linux %SERVICE_NAME%.bin
goto :eof

::==================================================
::  Utility: Build docker image
::  Params: %1=platform tag  %2=Dockerfile name  %3=binary name
::==================================================
:docker-build
set VERSION=v1.0.0
echo [DOCKER] Building image %SERVICE_NAME%:%VERSION%
cd /d "%~dp0"
docker build -t %SERVICE_NAME%:%VERSION% ^
             -f .\docker\%~2 . || (echo [ERROR] docker build failed & exit /b 1)
echo [DOCKER] Image build completed
goto :eof

::==================================================
::  5. Clean
::==================================================
:clean
if exist "%~dp0bin" (
    rd /s /q "%~dp0bin" 2>nul
    echo [CLEAN] Cleaned bin directory
) else (
    echo [CLEAN] bin directory does not exist
)
goto :eof