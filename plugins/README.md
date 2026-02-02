# LiteBlog 插件示例

## 快速开始

### 1. 启用插件系统

编辑 `configs/config.json`:

```json
{
  "plugins_config": {
    "enabled": true,
    "js_config": {
      "enabled": true,
      "plugin_dir": "plugins"
    }
  }
}
```

### 2. 测试插件

```bash
# 启动 LiteBlog
./LiteBlog

# 文章 API
curl http://localhost:8080/api/plugin/articles
curl http://localhost:8080/api/stats/articles

# 评论 API
curl http://localhost:8080/api/plugin/articles/firstArticle/comments

# 卡片 API
curl http://localhost:8080/api/plugin/cards

# gRPC 插件示例
curl http://localhost:8080/api/demo/stats
curl http://localhost:8080/api/demo/comments/firstArticle

# 创建操作（需要token）
curl -X POST http://localhost:8080/api/plugin/articles \
  -H "Authorization: your-token" \
  -H "Content-Type: application/json" \
  -d '{"title":"测试","content":"内容","content_html":"<p>内容</p>","author":"测试"}'
```

## 示例文件

### JavaScript 插件
- **`article-manager.js`** - 文章管理 REST API
- **`comment-card-manager.js`** - 评论和卡片管理 REST API
- **`test.js`** - 功能演示

### gRPC 插件
- **`plugin-example.go`** - 完整示例（路由、文章、评论、卡片）

编译并运行:
```bash
cd plugins
go build -o plugin-example plugin-example.go
./plugin-example 127.0.0.1:9090 your-access-key
```

## API 文档

详见 **`PLUGIN_API_GUIDE.md`**

## 调试

查看日志文件 `liteblog.log` 中带有 `[Plugin]` 前缀的日志。

