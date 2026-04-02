# Toolkit 项目

一个功能强大的工具包，支持插件化架构，包含网页爬虫、文件管理等功能。

## 项目结构

```
├── cmd/                  # 命令行工具源码
│   ├── gui/              # GUI 界面
│   ├── plugin-sdk/       # 插件 SDK
│   └── toolkit/          # 工具包主程序
├── docs/                 # 文档
│   └── tool-development-standard.md  # 工具开发标准
├── pkg/                  # 核心包
│   ├── config/           # 配置管理
│   ├── core/             # 核心功能
│   ├── di/               # 依赖注入
│   ├── logging/          # 日志系统
│   ├── mcp/              # 消息控制协议
│   ├── plugin/           # 插件系统
│   └── util/             # 工具函数
├── plugins/              # 插件目录
│   └── scraper/          # 网页爬虫插件
├── .gitignore            # Git 忽略文件
├── go.mod                # Go 模块文件
├── go.sum                # 依赖校验文件
├── tools.yaml            # 工具配置文件
└── README.md             # 项目说明
```

## 功能特性

- **插件化架构**：支持动态加载和管理插件
- **网页爬虫**：支持批量爬取、站点地图生成、XPath 解析等功能
- **文件管理**：支持文件操作和管理
- **GUI 界面**：提供直观的用户界面
- **RESTful API**：支持通过 API 调用工具功能

## 安装步骤

1. **克隆项目**
   ```bash
   git clone <repository-url>
   cd <project-directory>
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
   
   # 构建爬虫插件
   cd plugins/scraper
   go build -o scraper.exe main.go
   cd ../..
   ```

4. **配置插件**
   编辑 `tools.yaml` 文件，配置插件路径和参数：
   ```yaml
   plugin_dirs:
       - ./plugins
   plugins:
       scraper:
           enabled: true
           config:
               concurrent_limit: 10
   ```

## 使用方法

### 启动 GUI

```bash
./toolkit-gui.exe
```

然后在浏览器中访问 `http://localhost:8082`。

### 使用命令行

```bash
./toolkit.exe [command] [options]
```

### 使用 API

```bash
# 执行爬虫任务
curl -X POST http://localhost:8082/api/execute \
  -H "Content-Type: application/json" \
  -d '{"plugin": "scraper", "action": "fetch", "params": {"url": "https://example.com"}}'
```

## 插件开发

### 创建插件

1. 在 `plugins` 目录下创建新的插件目录
2. 创建 `main.go` 文件，实现 `tkplugin.Tool` 接口
3. 创建 `plugin.yaml` 文件，配置插件信息
4. 构建插件：`go build -o <plugin-name>.exe main.go`

### 插件接口

插件需要实现以下接口：

```go
type Tool interface {
    Metadata() ToolMetadata
    Init(ctx context.Context, config map[string]interface{}) error
    Execute(ctx context.Context, params map[string]interface{}) (*Result, error)
}
```

## 配置说明

### tools.yaml

```yaml
plugin_dirs:              # 插件目录列表
    - ./plugins
plugins:                  # 插件配置
    scraper:              # 插件名称
        enabled: true     # 是否启用
        config:           # 插件配置
            concurrent_limit: 10  # 并发限制
logging:
    level: info           # 日志级别
    format: text          # 日志格式
general:
    auto_discover: true   # 自动发现插件
    hot_reload: false     # 热重载
```

## 接口标准

### API 接口

#### 执行插件

- **URL**: `/api/execute`
- **方法**: `POST`
- **内容类型**: `application/json`
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
- **响应**:
  ```json
  {
    "success": true,
    "data": {
      "url": "https://example.com",
      "status_code": 200,
      "body": "<html>...</html>"
    },
    "error": "",
    "metrics": {
      "response_size": 1234
    }
  }
  ```

#### 列出插件

- **URL**: `/api/plugins`
- **方法**: `GET`
- **响应**:
  ```json
  {
    "plugins": [
      {
        "name": "scraper",
        "version": "2.0.0",
        "description": "网页爬虫插件"
      }
    ]
  }
  ```

## 爬虫插件功能

### 支持的操作

| 操作 | 功能描述 | 参数 |
|------|----------|------|
| `fetch` | 基础 HTTP 请求 | `url` (必需), `method` (可选) |
| `parse` | CSS 选择器解析 | `url` (必需), `selector` (必需) |
| `xpath` | XPath 表达式解析 | `url` (必需), `xpath` (必需) |
| `links` | 提取页面链接 | `url` (必需) |
| `text` | 提取页面文本 | `url` (必需) |
| `crawl` | 批量爬取 | `urls` (必需), `type` (可选), `limit` (可选) |
| `sitemap` | 生成站点地图 | `url` (必需) |
| `adaptive_parse` | 自适应解析 | `url` (必需), `target` (必需) |
| `batch` | 批量执行 | `tasks` (必需) |
| `export` | 结果导出 | `data` (必需), `format` (可选) |

## 示例

### 基础请求

```json
{
  "plugin": "scraper",
  "action": "fetch",
  "params": {
    "url": "https://example.com"
  }
}
```

### 批量爬取

```json
{
  "plugin": "scraper",
  "action": "crawl",
  "params": {
    "urls": ["https://example.com", "https://google.com"],
    "type": "text",
    "limit": 5
  }
}
```

## 开发指南

1. **代码风格**：遵循 Go 语言标准代码风格
2. **提交规范**：使用语义化提交消息
3. **测试**：为新功能编写单元测试
4. **文档**：更新相关文档

## 许可证

本项目采用 MIT 许可证。
