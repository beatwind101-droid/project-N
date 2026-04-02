# API 接口标准

本文档定义了 Toolkit 项目的 API 接口标准，包括接口设计原则、请求/响应格式、错误处理等。

## 设计原则

1. **RESTful 设计**：遵循 RESTful API 设计规范
2. **统一响应格式**：所有接口返回统一的 JSON 格式
3. **无状态**：API 接口应该是无状态的，依赖 token 进行认证
4. **版本控制**：通过 URL 路径进行版本控制
5. **幂等性**：PUT、DELETE 等操作应该是幂等的

## 响应格式

所有 API 接口返回的 JSON 格式如下：

```json
{
  "success": true,       // 操作是否成功
  "data": {},            // 响应数据
  "error": "",          // 错误信息（成功时为空）
  "metrics": {}          // 度量指标
}
```

### 成功响应

```json
{
  "success": true,
  "data": {
    // 具体数据
  },
  "error": "",
  "metrics": {
    "response_time": 123,  // 响应时间（毫秒）
    "data_size": 456       // 数据大小（字节）
  }
}
```

### 错误响应

```json
{
  "success": false,
  "data": {},
  "error": "错误信息",
  "metrics": {}
}
```

## 状态码

| 状态码 | 描述 |
|--------|------|
| 200 | 请求成功 |
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 403 | 禁止访问 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

## 接口列表

### 1. 执行插件

- **URL**: `/api/execute`
- **方法**: `POST`
- **内容类型**: `application/json`
- **请求体**:
  ```json
  {
    "plugin": "scraper",    // 插件名称
    "action": "fetch",      // 操作名称
    "params": {             // 操作参数
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

### 2. 列出插件

- **URL**: `/api/plugins`
- **方法**: `GET`
- **响应**:
  ```json
  {
    "success": true,
    "data": {
      "plugins": [
        {
          "name": "scraper",
          "version": "2.0.0",
          "description": "网页爬虫插件",
          "actions": ["fetch", "parse", "xpath", "links", "text", "crawl", "sitemap"]
        }
      ]
    },
    "error": "",
    "metrics": {}
  }
  ```

### 3. 获取插件信息

- **URL**: `/api/plugins/{name}`
- **方法**: `GET`
- **响应**:
  ```json
  {
    "success": true,
    "data": {
      "name": "scraper",
      "version": "2.0.0",
      "description": "网页爬虫插件",
      "actions": [
        {
          "name": "fetch",
          "description": "基础 HTTP 请求",
          "params": {
            "url": "string (必需)",
            "method": "string (可选)"
          }
        }
      ],
      "config": {
        "concurrent_limit": 10
      }
    },
    "error": "",
    "metrics": {}
  }
  ```

### 4. 初始化插件

- **URL**: `/api/plugins/{name}/init`
- **方法**: `POST`
- **内容类型**: `application/json`
- **请求体**:
  ```json
  {
    "config": {
      "concurrent_limit": 10
    }
  }
  ```
- **响应**:
  ```json
  {
    "success": true,
    "data": {
      "message": "插件初始化成功"
    },
    "error": "",
    "metrics": {}
  }
  ```

### 5. 关闭插件

- **URL**: `/api/plugins/{name}/shutdown`
- **方法**: `POST`
- **响应**:
  ```json
  {
    "success": true,
    "data": {
      "message": "插件关闭成功"
    },
    "error": "",
    "metrics": {}
  }
  ```

## 插件接口标准

### 插件元数据

插件需要实现 `Metadata()` 方法，返回插件的元数据：

```go
type ToolMetadata struct {
    Name        string   `json:"name"`
    Version     string   `json:"version"`
    Description string   `json:"description"`
    Author      string   `json:"author"`
    Category    string   `json:"category"`
    Tags        []string `json:"tags"`
    Actions     []Action `json:"actions"`
}

type Action struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Params      map[string]ParamSchema `json:"params"`
}

type ParamSchema struct {
    Type        string `json:"type"`
    Description string `json:"description"`
    Required    bool   `json:"required"`
    Default     string `json:"default,omitempty"`
}
```

### 执行接口

插件需要实现 `Execute()` 方法，处理具体的操作：

```go
func (t *Tool) Execute(ctx context.Context, params map[string]interface{}) (*Result, error) {
    // 解析参数
    action, ok := params["action"].(string)
    if !ok {
        return &Result{
            Success: false,
            Error:   "缺少 action 参数",
        }, nil
    }

    // 根据 action 执行不同的操作
    switch action {
    case "fetch":
        return t.fetch(params)
    case "parse":
        return t.parse(params)
    // 其他操作...
    default:
        return &Result{
            Success: false,
            Error:   fmt.Sprintf("未知操作: %s", action),
        }, nil
    }
}
```

### 结果格式

插件返回的结果格式：

```go
type Result struct {
    Success bool                   `json:"success"`
    Data    map[string]interface{} `json:"data,omitempty"`
    Error   string                 `json:"error,omitempty"`
    Metrics map[string]interface{} `json:"metrics,omitempty"`
}
```

## 错误处理

### 错误类型

| 错误类型 | 错误码 | 描述 |
|----------|--------|------|
| 参数错误 | 400 | 请求参数缺失或无效 |
| 插件错误 | 404 | 插件不存在或未加载 |
| 执行错误 | 500 | 插件执行失败 |
| 权限错误 | 401 | 未授权访问 |

### 错误消息格式

```json
{
  "success": false,
  "data": {},
  "error": "错误类型: 详细错误信息",
  "metrics": {}
}
```

## 最佳实践

1. **参数验证**：在插件执行前验证所有必需参数
2. **错误处理**：捕获并处理所有可能的错误，返回友好的错误信息
3. **日志记录**：记录插件执行过程中的关键信息和错误
4. **性能监控**：返回执行时间、数据大小等度量指标
5. **安全性**：验证输入参数，防止注入攻击和 SSRF 攻击

## 示例

### 调用爬虫插件的 fetch 操作

**请求**：
```bash
curl -X POST http://localhost:8082/api/execute \
  -H "Content-Type: application/json" \
  -d '{"plugin": "scraper", "action": "fetch", "params": {"url": "https://example.com"}}'
```

**响应**：
```json
{
  "success": true,
  "data": {
    "url": "https://example.com",
    "status_code": 200,
    "status": "200 OK",
    "headers": {
      "Content-Type": "text/html; charset=UTF-8"
    },
    "body": "<!doctype html>...",
    "content_type": "text/html; charset=UTF-8"
  },
  "error": "",
  "metrics": {
    "response_size": 1256,
    "response_time": 345
  }
}
```

### 调用爬虫插件的 crawl 操作

**请求**：
```bash
curl -X POST http://localhost:8082/api/execute \
  -H "Content-Type: application/json" \
  -d '{"plugin": "scraper", "action": "crawl", "params": {"urls": ["https://example.com", "https://google.com"], "type": "text", "limit": 2}}'
```

**响应**：
```json
{
  "success": true,
  "data": {
    "results": [
      {
        "url": "https://example.com",
        "success": true,
        "title": "Example Domain",
        "text": "Example Domain...",
        "char_count": 123
      },
      {
        "url": "https://google.com",
        "success": true,
        "title": "Google",
        "text": "Google...",
        "char_count": 456
      }
    ],
    "total": 2,
    "success": 2,
    "failed": 0,
    "crawl_type": "text"
  },
  "error": "",
  "metrics": {
    "crawl_time": 1234,
    "concurrent": 2
  }
}
```
