# 工具开发标准

## 目录结构规范

```
plugins/<tool-name>/
├── main.go          # 工具实现
└── plugin.yaml      # 工具清单
```

## 一、清单文件 plugin.yaml

```yaml
name: tool-name              # 唯一标识，小写字母+连字符
version: "1.0.0"             # 语义化版本
description: "功能描述"       # 简短说明
author: "作者名"
category: utility            # 分类: utility/demo/crawler/data
tags:                        # 标签，用于搜索分类
  - tag1
  - tag2
executable: tool-name        # 可执行文件名（Windows 自动补 .exe）
enabled: true                # 是否启用
```

## 二、接口定义

每个工具必须实现 `plugin.Tool` 接口：

```go
type Tool interface {
    Metadata() ToolMetadata                                          // 返回元信息
    Init(ctx context.Context, config map[string]interface{}) error  // 初始化
    Execute(ctx context.Context, params map[string]interface{}) (*Result, error)  // 执行
    Validate(params map[string]interface{}) error                   // 参数校验
    Shutdown(ctx context.Context) error                             // 优雅关闭
}
```

## 三、完整模板

```go
package main

import (
    "context"
    "fmt"
    tkplugin "github.com/yourorg/toolkit/pkg/plugin"
)

// MyTool 工具实现
type MyTool struct {
    // 在此定义内部状态字段
}

// Metadata 返回工具元信息
func (t *MyTool) Metadata() tkplugin.ToolMetadata {
    return tkplugin.ToolMetadata{
        Name:        "my-tool",
        Version:     "1.0.0",
        Description: "工具描述",
        Author:      "作者",
        Category:    "utility",
        Tags:        []string{"tag1", "tag2"},
        ConfigSchema: map[string]tkplugin.Field{
            "key": {
                Type:        "string",
                Description: "参数说明",
                Required:    true,
                Default:     "",
            },
        },
    }
}

// Init 初始化，接收配置
func (t *MyTool) Init(ctx context.Context, config map[string]interface{}) error {
    // 从 config 读取配置
    // 初始化资源
    return nil
}

// Execute 执行核心逻辑
func (t *MyTool) Execute(ctx context.Context, params map[string]interface{}) (*tkplugin.Result, error) {
    // 从 params 读取参数
    key, _ := params["key"].(string)

    // 执行业务逻辑
    result := fmt.Sprintf("处理结果: %s", key)

    return &tkplugin.Result{
        Success: true,
        Data:    map[string]string{"output": result},
    }, nil
}

// Validate 参数校验
func (t *MyTool) Validate(params map[string]interface{}) error {
    if _, ok := params["key"]; !ok {
        return fmt.Errorf("参数 key 必填")
    }
    return nil
}

// Shutdown 释放资源
func (t *MyTool) Shutdown(ctx context.Context) error {
    // 关闭连接、释放内存等
    return nil
}

// main 入口，必须调用 Serve
func main() {
    tkplugin.Serve(&MyTool{})
}
```

## 四、返回值规范

### 成功返回

```go
return &tkplugin.Result{
    Success: true,
    Data:    any,           // 任意类型，会 JSON 序列化
}, nil
```

### 业务失败（非错误）

```go
return &tkplugin.Result{
    Success: false,
    Error:   "失败原因描述",
}, nil
```

### 系统错误

```go
return nil, fmt.Errorf("系统级错误: %w", err)
```

## 五、参数类型定义

```go
ConfigSchema: map[string]tkplugin.Field{
    "name": {
        Type:        "string",   // string / int / bool / float
        Description: "参数说明",
        Required:    true,       // 是否必填
        Default:     "默认值",   // 默认值
    },
}
```

## 六、编译与注册

```bash
# 编译工具
go build -o plugins/<name>/<name>.exe ./plugins/<name>

# 编译后自动被发现，无需手动注册
# 启动 toolkit 或 toolkit-gui 即可
```

## 七、注意事项

1. **main 必须调用 `tkplugin.Serve(&YourTool{})`**
2. **Exec 必须返回 `(*Result, error)`，不能 panic**
3. **Shutdown 中关闭所有打开的资源**
4. **参数通过 `params["key"]` 读取，类型断言为 string**
5. **config 字段默认值必须是 string 类型**
6. **工具名与目录名、可执行文件名保持一致**
