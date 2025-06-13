@echo off
:: 设置 Go 编译器路径（如果 Go 已添加到系统 PATH 中，可以省略此行）
:: set PATH=C:\Go\bin;%PATH%

:: 清屏
cls

:: 检查是否安装了 Go
where go >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo Go 编译器未安装或未添加到 PATH 中，请先安装 Go。
    pause
    exit /b 1
)

:: 创建输出目录
if not exist "dist" (
    mkdir dist
)

:: 定义目标平台和架构
set "PLATFORMS=windows linux darwin"
set "ARCH=amd64"

:: 编译函数
:build
:: 根据平台设置可执行文件后缀
set "EXT="
if "%1"=="windows" set "EXT=.exe"

echo 正在编译 %1/%2 版本...
set GOOS=%1
set GOARCH=%2
go build -o dist\goping_%1_%2%EXT% main.go
if %ERRORLEVEL% neq 0 (
    echo 编译 %1/%2 版本失败！
    exit /b 1
)
echo 成功生成 dist\goping_%1_%2%EXT%
exit /b 0

:: 开始编译
echo 开始一键编译三个版本：Windows、Linux 和 macOS...
for %%P in (%PLATFORMS%) do (
    call :build %%P %ARCH%
)

:: 提示完成
echo.
echo 编译完成！所有文件已保存到 dist 目录中。
pause