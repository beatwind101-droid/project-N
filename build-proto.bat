@echo off

rem 构建 protobuf 代码
rem 确保 protoc 和 protoc-gen-go 已安装

set PROTO_DIR=pkg/plugin/proto
set PROTO_FILE=plugin.proto
set OUT_DIR=pkg/plugin/proto

echo 检查 protoc 是否安装...
protoc --version >nul 2>&1
if %errorlevel% neq 0 (
    echo 错误: protoc 未安装
    echo 请从 https://github.com/protocolbuffers/protobuf/releases 下载并安装 protoc
    echo 然后将 protoc.exe 添加到系统 PATH
    exit /b 1
)

echo 检查 protoc-gen-go 是否安装...
where protoc-gen-go >nul 2>&1
if %errorlevel% neq 0 (
    echo 错误: protoc-gen-go 未安装
    echo 正在尝试安装 protoc-gen-go...
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest >nul 2>&1
    if %errorlevel% neq 0 (
        echo 安装 protoc-gen-go 失败
        exit /b 1
    )
)

echo 检查 protoc-gen-go-grpc 是否安装...
where protoc-gen-go-grpc >nul 2>&1
if %errorlevel% neq 0 (
    echo 错误: protoc-gen-go-grpc 未安装
    echo 正在尝试安装 protoc-gen-go-grpc...
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest >nul 2>&1
    if %errorlevel% neq 0 (
        echo 安装 protoc-gen-go-grpc 失败
        exit /b 1
    )
)

echo 生成 protobuf 代码...
protoc --go_out=%OUT_DIR% --go-grpc_out=%OUT_DIR% %PROTO_DIR%/%PROTO_FILE%
if %errorlevel% neq 0 (
    echo 生成 protobuf 代码失败
    exit /b 1
)

echo 生成成功!
echo 生成的文件: %OUT_DIR%/plugin.pb.go

exit /b 0