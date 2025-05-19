# LiteBlog
LiteBlog is a simple blog system written in Golang,html,css,js. Aim to provide a simple, easy-to-use and lightweight blog system.
## Features
- Simple, Lightweight and easy-to-use interface
- Markdown and HTML support (with [markedJS](https://github.com/markedjs/marked) and [markdown-go](https://github.com/gomarkdown/markdown))
- own script and style injection support
- comment system support (Developing)
- specialized full caching system for blog and asynchronous caching mechanism
- Auto block malicious request with firewall and XSS attack (with [bluemonday](https://github.com/microcosm-cc/bluemonday))
- RSS Feed support
- Auto backup the configs and data
- Easy to deploy and manage
- Full static is under development
## Requirements
- Golang 1.16+
## Installation
### From Source
Clone the repository and run the following command to start the server.
```bash
git clone https://github.com/LiteBlog/LiteBlog.git
cd LiteBlog
go run LiteBlog
```
### From Binary
Download the latest binary from [release page](https://github.com/Unicode01/LiteBlog/releases) and run it.
```bash
git clone https://github.com/LiteBlog/LiteBlog.git
cd LiteBlog
go build -o LiteBlog
./LiteBlog
```
## Configuration
#### configs/config.json
This file contains the server configurations such as server port,TLS settings, cache settings, etc.  
##### access_config
- `backend_path`: This is the path to the backend server.
- `access_token`: This is the access token for the backend server.
##### cache_config
- `use_disk`: This is a boolean value to enable or disable the disk cache.
- `max_cache_size`: This is the maximum cache size in bytes.
- `max_cache_items`: This is the maximum number of cache items.
- `expire_time`: This is the expire time of the cache in seconds.
##### deliver_config
This is the deliver configuration. Impact the asynchronous cache mechanism.
- `buffer`: buffer size of the deliverer.
- `threads`: number of threads of the deliverer.
##### backup_config
- `backup_dir`: This is the directory to store the backup files.
- `backup_interval`: This is the interval of the backup in seconds.
- `max_backups`: This is the maximum number of backup files to keep.
- `max_backups_survival_time`: This is the maximum survival time of the backup files in seconds.
#### configs/global.json
This configs are used to customize the front-end.
#### configs/articles/*.json
This files are the articles data. Include the article title, content, comments, etc.
#### configs/cards.json
This file contains the cards data. It will impact the home page layout, rss feed layout, etc.
#### configs/firewall.json
This file contains the firewall rules. It will block the malicious request.
## Usage
### Index Page Edit Mode
When you in the index page, you can click the edit button or right click on the article title and select `Edit Mode` to enter the edit mode. In the edit mode, you can edit the card order, new cards, delete cards.
### Article Edit Mode
When you in the article page, you can right click the and select `Edit Article` to enter the edit mode. In the edit mode, you can edit the article title, content, author, etc.
### Article Add Mode
When you in the index page, you can enter the edit mode and click the `Add Article` button on context menu to enter the add mode. In the add mode, you can add a new article.
### Article Save
#### Save as Markdown
When you in the article page, you can right click the and select `Save as Markdown` to save the article as markdown file. You can edit it later.
#### Save as HTML
When you in the article page, you can right click the and select `Save as HTML` to save the article as html file. **You can't edit it more!**
### Comment System
The comment system is under development.
### Add your own script and style
You can add your own script and style to the `public/js/inject.js` and `public/css/customizestyle.css` directory. The script and style will be injected to the page automatically.
### Add your own card template
You can add your own card template to the `templates/your_card_template_name.html` file. The card template will be used to display the cards on the home page.
## FAQ
### How does the cache work?
The cache is used to improve the performance of the server. The cache will store the articles data and the rendered pages. When the server receive a request, it will check the cache first. If the cache hit, the server will return the cached data. If the cache miss, the server will render the page and store the data in the cache. The cache will be expired after the expire time.
### How does the asynchronous cache mechanism work?
The asynchronous cache mechanism is used to improve the performance of the server. When the resource can be cached, the server will use asynchronized deliver manager to deliver the content which need be cached. This can improve the performance of the server when need to cache a large amount of data.
### Can I configure the firewall rules?
Yes, but the rules now are not perfect, i will improve it in the future. You can configure the firewall rules in the `configs/firewall.json` file.
## Demo
[Unicode LiteBlog](https://un1c0de.com)
## License
MIT