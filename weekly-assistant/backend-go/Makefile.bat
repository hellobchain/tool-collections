@echo off
:: 65001 = UTF-8，936 = GBK，按需切换
chcp 65001 >nul
@echo off
setlocal enabledelayedexpansion
set SERVICE_NAME=weekly-assistant
::==================================================
::  路由表
::==================================================
if /i "%~1"==""          goto :build-local
if /i "%~1"=="linux"     goto :build-linux
if /i "%~1"=="docker"    goto :docker-local
if /i "%~1"=="docker-linux" goto :docker-linux
if /i "%~1"=="clean"     goto :clean
echo 用法:
echo   %~nx0              "编译本地平台可执行文件"
echo   %~nx0 linux        "交叉编译 Linux amd64 二进制"
echo   %~nx0 docker       "本地平台二进制 + docker 镜像"
echo   %~nx0 docker-linux "Linux 二进制 + docker 镜像"
echo   %~nx0 clean        "清理 bin 目录"
goto :eof

::==================================================
::  工具函数：编译 Go
::  入参：%1=GOOS  %2=输出文件名（含后缀）
::==================================================
:go-build
cd /d "%~dp0cmd"
go mod tidy || (echo [ERROR] go mod tidy 失败 & exit /b 1)
set CGO_ENABLED=0
set GOOS=%~1
set GOARCH=amd64
go build -trimpath -ldflags="-s -w" -o "%~dp0bin\%~2" || (echo [ERROR] 编译失败 & exit /b 1)
echo [BUILD] 生成 %~2 完成
goto :eof

::==================================================
::  1. 本地平台
::==================================================
:build-local
call :go-build windows %SERVICE_NAME%.exe
goto :eof

::==================================================
::  2. Linux 交叉编译
::==================================================
:build-linux
call :go-build linux %SERVICE_NAME%.bin
goto :eof

::==================================================
::  3. 本地平台 + docker 镜像
::==================================================
:docker-local
call :build-local
call :docker-build windows Dockerfile %SERVICE_NAME%.exe
goto :eof

::==================================================
::  4. Linux 交叉编译 + docker 镜像
::==================================================
:docker-linux
call :build-linux
call :docker-build linux Dockerfile-linux %SERVICE_NAME%.bin
goto :eof

::==================================================
::  工具函数：打镜像
::  入参：%1=平台标记  %2=Dockerfile 名  %3=二进制名
::==================================================
:docker-build
set VERSION=v1.0.0
echo [DOCKER] 构建镜像 %SERVICE_NAME%:%VERSION%
cd /d "%~dp0"
docker build -t %SERVICE_NAME%:%VERSION% ^
             -f .\docker\%~2 . || (echo [ERROR] docker build 失败 & exit /b 1)
echo [DOCKER] 镜像构建完成
goto :eof

::==================================================
::  5. 清理
::==================================================
:clean
if exist "%~dp0bin" (
    rd /s /q "%~dp0bin" 2>nul
    echo [CLEAN] 已清理 bin 目录
) else (
    echo [CLEAN] bin 目录不存在
)
goto :eof