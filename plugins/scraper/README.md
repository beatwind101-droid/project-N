# Scraper 插件

Scraper 是一个功能强大的网页爬虫插件，支持批量爬取、站点地图生成、XPath 解析等功能。

## 插件信息

- **名称**：scraper
- **版本**：2.0.0
- **作者**：Toolkit Team
- **类别**：数据采集
- **描述**：网页爬虫插件，支持智能元素定位、反检测绕过、代理轮换

## 开发语言

- **主要语言**：Go 语言
- **依赖库**：
  - github.com/PuerkitoBio/goquery
  - golang.org/x/net
  - google.golang.org/grpc

## 功能特性

### 1. 基础功能
- **fetch**：基础 HTTP 请求，支持 GET、POST 等方法
- **parse**：CSS 选择器解析，提取页面元素
- **xpath**：XPath 表达式解析，更灵活的元素定位
- **links**：提取页面链接
- **text**：提取页面文本内容

### 2. 高级功能
- **crawl**：批量爬取，支持并发控制
- **sitemap**：生成站点地图
- **adaptive_parse**：自适应解析，智能定位元素
- **batch**：批量执行任务
- **export**：结果导出，支持多种格式

## 接口说明

### 1. 执行接口

插件通过 Toolkit 核心系统的 API 执行，请求格式：

```json
{
  "plugin": "scraper",
  "action": "fetch",
  "params": {
    "url": "https://example.com"
  }
}
```

### 2. 支持的操作

| 操作 | 功能描述 | 参数 |
|------|----------|------|
| `fetch` | 基础 HTTP 请求 | `url` (必需), `method` (可选), `body` (可选), `headers` (可选) |
| `parse` | CSS 选择器解析 | `url` (必需), `selector` (必需) |
| `xpath` | XPath 表达式解析 | `url` (必需), `xpath` (必需) |
| `links` | 提取页面链接 | `url` (必需) |
| `text` | 提取页面文本 | `url` (必需) |
| `crawl` | 批量爬取 | `urls` (必需), `type` (可选), `limit` (可选) |
| `sitemap` | 生成站点地图 | `url` (必需), `depth` (可选) |
| `adaptive_parse` | 自适应解析 | `url` (必需), `target` (必需) |
| `batch` | 批量执行 | `tasks` (必需) |
| `export` | 结果导出 | `data` (必需), `format` (可选) |

### 3. 配置接口

在 `tools.yaml` 中配置插件：

```yaml
plugins:
    scraper:
        enabled: true
        config:
            concurrent_limit: 10          # 并发限制
            timeout: 30                    # 超时时间（秒）
            user_agent: "Mozilla/5.0..."    # 用户代理
            proxy: "http://proxy:8080"     # 代理设置
```

## 使用示例

### 1. 基础请求

```json
{
  "plugin": "scraper",
  "action": "fetch",
  "params": {
    "url": "https://example.com"
  }
}
```

### 2. CSS 选择器解析

```json
{
  "plugin": "scraper",
  "action": "parse",
  "params": {
    "url": "https://example.com",
    "selector": "h1"
  }
}
```

### 3. 批量爬取

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

### 4. 站点地图生成

```json
{
  "plugin": "scraper",
  "action": "sitemap",
  "params": {
    "url": "https://example.com",
    "depth": 2
  }
}
```

## 响应格式

所有操作返回统一的响应格式：

```json
{
  "success": true,
  "data": {
    // 具体数据
  },
  "error": "",
  "metrics": {
    "response_time": 123,
    "data_size": 456
  }
}
```

## 错误处理

| 错误类型 | 错误信息 |
|----------|----------|
| 参数错误 | "请提供要访问的网址 (url 参数)" |
| 网络错误 | "请求失败: 网络连接超时" |
| 解析错误 | "HTML 解析失败: 无效的 HTML" |
| 选择器错误 | "未找到匹配元素，请检查选择器是否正确" |

## 安装和使用

### 1. 安装插件

1. 确保 Toolkit 核心系统已安装
2. 将插件目录复制到 `plugins` 目录
3. 构建插件：
   ```bash
   cd plugins/scraper
   go build -o scraper.exe main.go
   ```
4. 在 `tools.yaml` 中配置插件

### 2. 启动服务

```bash
./toolkit-gui.exe
```

### 3. 访问 GUI

在浏览器中访问 `http://localhost:8082`，选择 scraper 插件执行操作。

## 开发指南

### 1. 扩展插件

1. 在 `main.go` 中添加新的操作方法
2. 更新 `Execute` 方法，添加新操作的处理逻辑
3. 更新插件配置和文档

### 2. 最佳实践

- **错误处理**：捕获并处理所有可能的错误
- **日志记录**：记录关键操作和错误信息
- **性能优化**：使用并发和缓存提高性能
- **安全性**：验证输入参数，防止注入攻击
- **可扩展性**：设计模块化的代码结构

## 许可证

本插件采用 MIT 许可证。
