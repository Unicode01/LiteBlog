# LiteBlog

A simple, lightweight, and highly customizable blog system written in Go.

## Features

- Simple and lightweight interface
- Markdown & HTML support ([markedJS](https://github.com/markedjs/marked), [highlightJS](https://github.com/highlightjs/highlight.js))
- Custom script and style injection
- Comment system with CAPTCHA support
- Full caching system with async mechanism
- Firewall & XSS protection ([bluemonday](https://github.com/microcosm-cc/bluemonday))
- **Powerful Plugin System** (gRPC & JavaScript) - 21 APIs for articles, comments, cards, routing
- RSS Feed & Sitemap
- Auto backup
- Full static site generation

## Quick Start

### Installation

**Requirements:** Go 1.24+

```bash
# From source
git clone https://github.com/LiteBlog/LiteBlog.git
cd LiteBlog
go build -o LiteBlog
./LiteBlog

# Download binary from releases
# Visit: https://github.com/Unicode01/LiteBlog/releases
```

### Docker

```bash
./build.sh platform linux/amd64
docker build -t liteblog .
docker run -d -p 80:80 -v $(pwd)/configs:/liteblog/configs liteblog
```

---

## Plugin System

**21 Available APIs:**
- **Articles**: GetArticle, AddArticle, EditArticle, DeleteArticle, GetAllArticleIDs
- **Comments**: GetComments, AddComment, DeleteComment
- **Cards**: GetAllCards, GetCard, AddCard, EditCard, DeleteCard
- **Auth & Config**: VerifyToken, GetConfig, Log
- **Routing**: AddHook, DeleteHook, AddRouteListener, DeleteRouteListener
- **Render**: AddRenderMap, GetRenderMap, DeleteRenderMap

📚 **Docs:** `plugins/PLUGIN_API_GUIDE.md`  
💡 **Examples:** `plugins/article-manager.js`, `plugins/comment-card-manager.js`

**Enable plugins in `configs/config.json`:**
```json
{
  "plugins_config": {
    "enabled": true,
    "grpc_config": {
      "enabled": true,
      "listener_address": "127.0.0.1:9090",
      "access_key": "your_key"
    },
    "js_config": {
      "enabled": true,
      "plugin_dir": "plugins"
    }
  }
}
```

---

## Configuration

### Main Config Files

- **`configs/config.json`** - Server settings, cache, backup, plugins
- **`configs/global.json`** - Frontend customization
- **`configs/articles/*.json`** - Article data
- **`configs/cards.json`** - Home page cards
- **`configs/firewall.json`** - Firewall rules

### Key Settings

**access_config**
- `access_token`: Backend access token
- `backend_path`: Backend API path
- `read_only`: Enable read-only mode

**cache_config**
- `use_disk`: Enable disk cache
- `max_cache_size`: Max cache size (bytes)
- `expire_time`: Cache expiration (seconds)

**comment_config**
- `enabled`: Enable comments
- `type`: `cloudflare_turnstile` or `google_recaptcha`
- `min_seconds_between_comments`: Spam prevention

**backup_config**
- `backup_interval`: Backup interval (seconds)
- `max_backups`: Max backup files to keep

---

## Firewall Rules

Add rules to `configs/firewall.json`:

**Block IP:**
```json
{
  "name": "block_ip_example",
  "action": 1,
  "type": "ipaddr",
  "rule": "8.9.10.11",
  "timeout": 99999999999
}
```

**Block IP Range:**
```json
{
  "name": "block_cidr_example",
  "action": 1,
  "type": "ipcidr",
  "rule": "192.168.0.0/24",
  "timeout": 99999999999
}
```

**Rate Limit:**
```json
{
  "name": "rate_limit",
  "action": 1,
  "type": "ratelimit",
  "rule": "200",
  "timeout": 9999999999,
  "args": ["block_time=60", "cycle=10"]
}
```

More types: `useragent`, etc. See `configs/firewall.json`.

---

## Usage

### Edit Mode
- **Index Page**: Click edit button or right-click → Edit Mode
- **Article Page**: Right-click → Edit Article
- **Add Article**: Edit Mode → Add Article button

### Customization
- **Scripts**: `public/js/inject.js`
- **Styles**: `public/css/customizestyle.css`
- **Card Templates**: `templates/your_template.html`

### Static Site Generation
```bash
./LiteBlog -static
```
Generates static files in `static/public/` directory.

---

## Demo

[Unicode LiteBlog](https://un1c0de.com)

## License

MIT
