# Toolkit 核心项目

Toolkit 是一个功能强大的插件化工具平台，支持动态加载和管理各种插件，提供统一的 API 接口和 GUI 界面。

## 项目结构

```
├── cmd/                  # 命令行工具和 GUI
│   ├── gui/              # GUI 界面
│   ├── plugin-sdk/       # 插件 SDK
│   └── toolkit/          # 工具包主程序
├── docs/                 # 文档
│   ├── api-standard.md   # API 接口标准
│   └── tool-development-standard.md  # 工具开发标准
├── pkg/                  # 核心包
│   ├── config/           # 配置管理
│   ├── core/             # 核心功能
│   ├── di/               # 依赖注入
│   ├── logging/          # 日志系统
│   ├── mcp/              # 消息控制协议
│   ├── plugin/           # 插件系统
│   └── util/             # 工具函数
├── plugins/              # 插件目录（单独管理）
├── .gitignore            # Git 忽略文件
├── go.mod                # Go 模块文件
├── go.sum                # 依赖校验文件
├── tools.yaml            # 工具配置文件
└── README.md             # 项目说明
```

## 开发语言

- **主要语言**：Go 语言
- **GUI 前端**：HTML/CSS/JavaScript
- **API**：RESTful HTTP API
- **插件通信**：gRPC

## 核心功能

### 1. 插件管理
- 动态发现和加载插件
- 插件生命周期管理
- 插件配置管理
- 插件依赖管理

### 2. 任务执行
- 异步任务执行
- 任务状态监控
- 任务结果处理

### 3. API 服务
- RESTful API 接口
- 插件执行接口
- 系统状态接口

### 4. GUI 界面
- 插件管理界面
- 任务执行界面
- 结果展示界面

### 5. 安全机制
- 插件权限控制
- 输入验证
- 安全的插件加载

## 接口说明

### 1. 插件接口

插件需要实现 `tkplugin.Tool` 接口：

```go
type Tool interface {
    Metadata() ToolMetadata
    Init(ctx context.Context, config map[string]interface{}) error
    Execute(ctx context.Context, params map[string]interface{}) (*Result, error)
}
```

### 2. API 接口

#### 执行插件
- **URL**: `/api/execute`
- **方法**: `POST`
- **请求体**:
  ```json
  {
    "plugin": "scraper",
    "action": "fetch",
    "params": {
      "url": "https://example.com"
    }
  }
  ```

#### 列出插件
- **URL**: `/api/plugins`
- **方法**: `GET`

### 3. 配置接口

#### 工具配置 (`tools.yaml`)
```yaml
plugin_dirs:
    - ./plugins
plugins:
    scraper:
        enabled: true
        config:
            concurrent_limit: 10
```

## 安装和使用

### 安装

1. **克隆项目**
   ```bash
   git clone <repository-url>
   cd toolkit-core
   ```

2. **安装依赖**
   ```bash
   go mod download
   ```

3. **构建项目**
   ```bash
   # 构建工具包
   go build -o toolkit.exe ./cmd/toolkit
   
   # 构建 GUI
   go build -o toolkit-gui.exe ./cmd/gui
   
   # 构建插件 SDK
   go build -o plugin-sdk.exe ./cmd/plugin-sdk
   ```

### 使用

1. **启动服务**
   ```bash
   ./toolkit-gui.exe
   ```

2. **访问 GUI**
   在浏览器中访问 `http://localhost:8082`

3. **使用 API**
   ```bash
   curl -X POST http://localhost:8082/api/execute \
     -H "Content-Type: application/json" \
     -d '{"plugin": "scraper", "action": "fetch", "params": {"url": "https://example.com"}}'
   ```

## 插件开发

### 1. 创建插件

1. 在 `plugins` 目录下创建新的插件目录
2. 创建 `main.go` 文件，实现 `tkplugin.Tool` 接口
3. 创建 `plugin.yaml` 文件，配置插件信息
4. 构建插件：`go build -o <plugin-name>.exe main.go`

### 2. 插件配置

```yaml
name: scraper
version: "2.0.0"
description: "网页爬虫插件"
author: "Toolkit Team"
category: 数据采集
tags:
  - 爬虫
  - 网页
  - 数据采集
executable: scraper.exe
plugin_type: exe
enabled: true
```

### 3. 插件 API

插件通过 gRPC 与核心系统通信，实现以下方法：
- `GetMetadata`：获取插件元数据
- `Initialize`：初始化插件
- `Execute`：执行插件操作
- `Validate`：验证参数
- `Shutdown`：关闭插件

## 开发指南

1. **代码风格**：遵循 Go 语言标准代码风格
2. **提交规范**：使用语义化提交消息
3. **测试**：为新功能编写单元测试
4. **文档**：更新相关文档
5. **安全性**：遵循安全最佳实践

## 许可证

本项目采用 MIT 许可证。
