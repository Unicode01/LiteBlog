// render variables from server
var Card_max_width = {{global:card_max_width}}; //w
var Card_max_height = {{global:card_max_height}};
var Card_min_width = {{global:card_min_width}}; //w
var Card_min_height = {{global:card_min_height}};
var Context_menu_html = `{{file:context_menu}}`
// end render
var switchThemeListeners = [];
var contextMenuList = [];

function init() {
    ResizeCard();
    AddEventListener();
    LogHistory();
}

function ResizeCard() {
    var avalia_width = window.screen.availWidth;
    var avalia_height = window.screen.availHeight - 120;
    var root = document.documentElement;
    card_width = Card_min_width;
    if (Card_min_width < avalia_width && avalia_width <= Card_max_width) {
        card_width = avalia_width;
    } else if (avalia_width > Card_max_width) {
        card_width = Card_max_width;
    }
    console.log(card_width);
    card_height = Card_min_height;
    if (Card_min_height < avalia_height && avalia_height <= Card_max_height) {
        card_height = avalia_height;
    } else if (avalia_height > Card_max_height) {
        card_height = Card_max_height;
    }
    root.style.setProperty('--card-width', card_width + 'px');
    root.style.setProperty('--card-height', card_height + 'px');
}

function OnContextMenu(event) {
    // render context menu html
    var context_menu_norender = Context_menu_html;
    const prevmenu = document.getElementById('context-menu');
    prevmenu?.remove();

    domParser = new DOMParser();
    var context_menu_doc = domParser.parseFromString(context_menu_norender, "text/html").body.firstChild;
    // console.log(context_menu_doc);
    // -add basic options
    // ---add copy option
    // ---check selection then add copy option
    if (window.getSelection().toString() != '') {
        var menu_copy = document.createElement('div');
        menu_copy.classList.add('menu-item');
        menu_copy.innerHTML = '<a class="link" href="#">Copy</a>';
        menu_copy.addEventListener('click', function (event) {
            event.preventDefault();
            selection = window.getSelection();
            copyText(selection.toString());
        });
        context_menu_doc.appendChild(menu_copy);
        // add item line
        var menu_line = document.createElement('div');
        menu_line.classList.add('menu-item-line');
        context_menu_doc.appendChild(menu_line);
    }
    // -add Reload option
    var menu_Reload = document.createElement('div');
    menu_Reload.classList.add('menu-item');
    menu_Reload.innerHTML = '<a class="link" href="#">Reload</a>';
    menu_Reload.addEventListener('click', function (event) {
        event.preventDefault();
        location.reload();
    });
    context_menu_doc.appendChild(menu_Reload);

    // add menu line
    var menu_line = document.createElement('div');
    menu_line.classList.add('menu-item-line');
    context_menu_doc.appendChild(menu_line);

    // check if cursor is in card
    const cards = document.querySelectorAll('.card-container');
    cards.forEach(card => {
        const cursorX = event.clientX;
        const cursorY = event.clientY;
        const cardX = card.getBoundingClientRect().left;
        const cardY = card.getBoundingClientRect().top;
        const cardWidth = card.getBoundingClientRect().width;
        const cardHeight = card.getBoundingClientRect().height;
        if (cursorX > cardX && cursorX < cardX + cardWidth && cursorY > cardY && cursorY < cardY + cardHeight) {
            // in card, add get link option
            // add copy link option
            var menu_getlink = document.createElement('div');
            menu_getlink.classList.add('menu-item');
            menu_getlink.innerHTML = '<a class="link" href="#">Copy Link</a>';
            menu_getlink.addEventListener('click', function (event) {
                event.preventDefault();
                var linkdom = card.querySelector('.link');
                if (!linkdom) {
                    console.log("no link found");
                    return;
                }
                var link = linkdom.href;
                console.log(link);
                // copy link to clipboard
                copyText(link);
            });
            context_menu_doc.appendChild(menu_getlink);

            // add item line
            var menu_line = document.createElement('div');
            menu_line.classList.add('menu-item-line');
            context_menu_doc.appendChild(menu_line);

            // check if in edit mode
            if (window._editMode) {
                // add delete option
                var menu_delete = document.createElement('div');
                menu_delete.classList.add('menu-item');
                menu_delete.innerHTML = '<a class="link" href="#">Delete Card</a>';
                menu_delete.addEventListener('click', function (event) {
                    event.preventDefault();
                    if (DeleteCard(card.getAttribute("card-id"))) {
                        console.log("card deleted");
                        // reload window
                        // location.reload();
                    } else {
                        console.log("failed to delete card");
                    }
                });
                context_menu_doc.appendChild(menu_delete);

                // add item line
                var menu_line = document.createElement('div');
                menu_line.classList.add('menu-item-line');
                context_menu_doc.appendChild(menu_line);

                // add edit option
                var menu_edit = document.createElement('div');
                menu_edit.classList.add('menu-item');
                menu_edit.innerHTML = '<a class="link" href="#">Edit Card</a>';
                menu_edit.addEventListener('click', function (event) {
                    event.preventDefault();
                    EditCard(card.getAttribute("card-id"));
                });
                context_menu_doc.appendChild(menu_edit);
                // add item line
                var menu_line = document.createElement('div');
                menu_line.classList.add('menu-item-line');
                context_menu_doc.appendChild(menu_line);
            }

        }
    });

    // check if edit mode
    if (window._editMode) {
        // add addcard option
        var menu_addcard = document.createElement('div');
        menu_addcard.classList.add('menu-item');
        menu_addcard.innerHTML = '<a class="link" href="#">Add Card</a>';
        menu_addcard.addEventListener('click', function (event) {
            event.preventDefault();
            if (AddCard()) {
                console.log("card added");
            };
        });
        context_menu_doc.appendChild(menu_addcard);
        // add item line
        var menu_line = document.createElement('div');
        menu_line.classList.add('menu-item-line');
        context_menu_doc.appendChild(menu_line);
        // add add article option
        var menu_addarticle = document.createElement('div');
        menu_addarticle.classList.add('menu-item');
        menu_addarticle.innerHTML = '<a class="link" href="#">Add Article</a>';
        menu_addarticle.addEventListener('click', function (event) {
            event.preventDefault();
            location.href = '/addarticle.html';
        });
        context_menu_doc.appendChild(menu_addarticle);
        // add item line
        var menu_line = document.createElement('div');
        menu_line.classList.add('menu-item-line');
        context_menu_doc.appendChild(menu_line);
        // add edit custom settings option
        var menu_editsettings = document.createElement('div');
        menu_editsettings.classList.add('menu-item');
        menu_editsettings.innerHTML = '<a class="link" href="#">Custom Settings</a>';
        menu_editsettings.addEventListener('click', function (event) {
            event.preventDefault();
            EditCustomSettings();
        });
        context_menu_doc.appendChild(menu_editsettings);
        // add item line
        var menu_line = document.createElement('div');
        menu_line.classList.add('menu-item-line');
        context_menu_doc.appendChild(menu_line);
    }

    // add editmode option
    const editmodeExist = document.getElementById('edit-button');
    if (editmodeExist && location.pathname == '/index.html') {
        var menu_editmode = document.createElement('div');
        menu_editmode.classList.add('menu-item');
        if (window._editMode) {
            menu_editmode.innerHTML = '<a class="link" href="#">Exit Edit Mode</a>';
        } else {
            menu_editmode.innerHTML = '<a class="link" href="#">Edit Mode</a>';
        }
        menu_editmode.addEventListener('click', function (event) {
            event.preventDefault();
            var editmode = document.getElementById('edit-button');
            editmode?.click();
        });
        context_menu_doc.appendChild(menu_editmode);
        // add item line
        var menu_line = document.createElement('div');
        menu_line.classList.add('menu-item-line');
        context_menu_doc.appendChild(menu_line);
    }

    // check if in addarticle.html or editarticle.html
    if (location.pathname == '/addarticle.html' || location.pathname == '/editarticle.html') {
        // add save article option
        var menu_save = document.createElement('div');
        menu_save.classList.add('menu-item');
        menu_save.innerHTML = '<a class="link" href="#">Save Article</a>';
        menu_save.addEventListener('click', function (event) {
            event.preventDefault();
            SaveArticle();
        });
        context_menu_doc.appendChild(menu_save);
        // add item line
        var menu_line = document.createElement('div');
        menu_line.classList.add('menu-item-line');
        context_menu_doc.appendChild(menu_line);
    }

    // check if in /articles/
    if (location.pathname.startsWith('/articles/')) {
        // add edit article option
        var menu_edit = document.createElement('div');
        menu_edit.classList.add('menu-item');
        menu_edit.innerHTML = '<a class="link" href="#">Edit Article</a>';
        menu_edit.addEventListener('click', function (event) {
            event.preventDefault();
            location.href = '/editarticle.html?article_id=' + location.pathname.split('/')[2];
        });
        context_menu_doc.appendChild(menu_edit);
        // add item line
        var menu_line = document.createElement('div');
        menu_line.classList.add('menu-item-line');
        context_menu_doc.appendChild(menu_line);
        // add delete article option
        var menu_delete = document.createElement('div');
        menu_delete.classList.add('menu-item');
        menu_delete.innerHTML = '<a class="link" href="#">Delete Article</a>';
        menu_delete.addEventListener('click', function (event) {
            event.preventDefault();
            DeleteArticleAPI(location.pathname.split('/')[2], function (result) {
                if (result) {
                    console.log("article deleted");
                    location.href = '/';
                } else {
                    console.log("failed to delete article");
                }
            });
        });
        context_menu_doc.appendChild(menu_delete);
        // add item line
        var menu_line = document.createElement('div');
        menu_line.classList.add('menu-item-line');
        context_menu_doc.appendChild(menu_line);
        // add add comment option
        var menu_addcomment = document.createElement('div');
        menu_addcomment.classList.add('menu-item');
        menu_addcomment.innerHTML = '<a class="link" href="#">Add Comment</a>';
        menu_addcomment.addEventListener('click', function (event) {
            event.preventDefault();
            ShowCommentInputBox();
        });
        context_menu_doc.appendChild(menu_addcomment);
        // add item line
        var menu_line = document.createElement('div');
        menu_line.classList.add('menu-item-line');
        context_menu_doc.appendChild(menu_line);
        // check if in comment doms
        const comment_doms = document.querySelectorAll(".article-comment")
        comment_doms.forEach(comment_dom => {
            const cursorX = event.clientX;
            const cursorY = event.clientY;
            const commentX = comment_dom.getBoundingClientRect().left;
            const commentY = comment_dom.getBoundingClientRect().top;
            const commentWidth = comment_dom.getBoundingClientRect().width;
            const commentHeight = comment_dom.getBoundingClientRect().height;
            if (cursorX > commentX && cursorX < commentX + commentWidth && cursorY > commentY && cursorY < commentY + commentHeight) {
                // in comment block, add delete comment option
                var menu_deletecomment = document.createElement('div');
                menu_deletecomment.classList.add('menu-item');
                menu_deletecomment.innerHTML = '<a class="link" href="#">Delete Comment</a>';
                menu_deletecomment.addEventListener('click', function (event) {
                    event.preventDefault();
                    const comment_id = comment_dom.getAttribute("comment-id");
                    DeleteCommentAPI(comment_id, function (result) {
                        if (result) {
                            console.log("comment deleted");
                            comment_dom.remove();
                        } else {
                            console.log("failed to delete comment");
                        }
                    });
                });
                context_menu_doc.appendChild(menu_deletecomment);
                // add item line
                var menu_line = document.createElement('div');
                menu_line.classList.add('menu-item-line');
                context_menu_doc.appendChild(menu_line);
            }
        });
    }

    contextMenuList.forEach(function (item) {
        decisionFunc = item.decisionFunction;
        title = item.title;
        callback = item.callback;
        if (decisionFunc(event)) {
            // add menu item
            var menu_item = document.createElement('div');
            menu_item.classList.add('menu-item');
            menu_item.innerHTML = '<a class="link" href="#">' + title + '</a>';
            menu_item.addEventListener('click', function (event) {
                event.preventDefault();
                callback(event);
            });
            context_menu_doc.appendChild(menu_item);
            // add item line
            var menu_line = document.createElement('div');
            menu_line.classList.add('menu-item-line');
            context_menu_doc.appendChild(menu_line);
        }
    });

    // check if last item is line
    var last_item = context_menu_doc.lastElementChild;
    if (last_item.classList.contains('menu-item-line')) {
        last_item.remove();
    }
    // set position
    var menu_x = event.clientX;
    var menu_y = event.clientY;
    context_menu_doc.style.left = menu_x + 'px';
    context_menu_doc.style.top = menu_y + 'px';
    // append to body
    document.body.appendChild(context_menu_doc);
    // force rerender
    context_menu_doc.offsetHeight;
    // set scale to 1
    context_menu_doc.style.transform = "scale(1)";

}

function AddEventListener() {
    // add resize event listener to window if /index
    if (location.pathname == '/index.html') {
        window.addEventListener('resize', function () {
            ResizeCard();
        });
    }
    // add contextmenu event listener to body
    document.body.addEventListener('contextmenu', function (event) {
        event.preventDefault();
        OnContextMenu(event);
    });
    // add click event listener to body to hide context menu
    document.addEventListener('click', function () {
        const menu = document.getElementById('context-menu');
        menu?.remove();
    });
    // add click event listener to history button
    const history_button = document.getElementById('history-button');
    history_button?.addEventListener('click', function (e) {
        e.preventDefault();
        OnHistoryButtonClick();
    });
    // add click event listener to theme switch button
    const theme_switch_button = document.getElementById('switch-theme-button');
    theme_switch_button?.addEventListener('click', themeSwitchClick);

}

function OnHistoryButtonClick() {
    var history_menu = document.getElementById('history-menu');
    if (history_menu) {
        history_menu.remove();
    } else {
        historyItems = localStorage.getItem('history');
        if (!historyItems) {
            return;
        }
        // get history from local storage
        const DomParser = new DOMParser();
        let menu_doc = DomParser.parseFromString(Context_menu_html, "text/html").body.firstChild;
        menu_doc.style.minWidth = "200px";
        menu_doc.id = 'history-menu';
        menu_doc.classList.remove("context-menu")
        menu_doc.classList.add("history-menu")
        historyItems = JSON.parse(historyItems);
        historyItems.forEach(function (item) {
            const menu_item = document.createElement('div');
            menu_item.classList.add('menu-item');
            menu_item.innerHTML = '<a class="link" href="' + item.url + '">' + item.title + '</a>';
            menu_item.addEventListener('click', function (event) {
                event.preventDefault();
                window.location.href = item.url;
            });
            menu_doc.appendChild(menu_item);
            var menu_line = document.createElement('div');
            menu_line.classList.add('menu-item-line');
            menu_doc.appendChild(menu_line);
        });

        if (menu_doc.lastElementChild.classList.contains('menu-item-line')) {
            menu_doc.lastElementChild.remove();
        }
        // set position
        const history_button = document.getElementById('history-button');
        const menu_x = history_button.offsetLeft + history_button.offsetWidth - 200;
        const menu_y = history_button.offsetTop + history_button.offsetHeight;
        menu_doc.style.left = menu_x + 'px';
        menu_doc.style.top = menu_y + 'px';
        // append to body
        document.body.appendChild(menu_doc);

    }
}

function LogHistory() {
    if (!window.location.pathname.startsWith('/articles/')) {
        return;
    }
    // log to history
    const url = window.location.pathname;
    let historyJson = localStorage.getItem('history');
    if (historyJson) {
        historyJson = JSON.parse(historyJson);
    } else {
        historyJson = [];
    }
    // remove old history
    historyJson = historyJson.filter(function (item) {
        return item.url != url;
    });
    // add new history
    historyJson.unshift({
        url: url,
        title: document.title
    });
    // save to local storage
    localStorage.setItem('history', JSON.stringify(historyJson));
}

// tool functions
async function copyText(text) {
    try {
        await navigator.clipboard.writeText(text);
        console.log("link copied to clipboard with clipboard api:" + text);
    } catch (err) {
        // 现代 API 失败时回退到旧方法
        const success = document.execCommand("copy");
        if (success) {
            console.log("link copied to clipboard:" + text);
        } else {
            console.log("failed to copy link:" + text);
        }
    }
}

function InsertDarkCss(callback) {
    function loadStyleString(css){
        var style = document.createElement("style");
        style.type = "text/css";
        style.id = "dark-theme";
        try{
            style.appendChild(document.createTextNode(css));
        } catch (ex){
            style.textContent = css;
        }
        var head = document.getElementsByTagName("head")[0];
        head.appendChild(style);
    }
    fetch('/css/dark.css')
       .then(response => response.text())
       .then(css => {
            loadStyleString(css);
            // save to local storage
            localStorage.setItem("dark-theme-css", css);
            callback();
        })
       .catch(error => {
            console.log(error);
        });
}

function RemoveDarkCss() {
    const link = document.querySelector('style[id="dark-theme"]');
    if (link) {
        link.remove();
    }
}

function SetTheme(Theme) {
    localStorage.setItem('theme', Theme);
    if (Theme == "dark") {
        // set dark theme
        // InsertDark Css
        InsertDarkCss(function () {
            console.log("dark theme loaded");
            if (IsDarkCSSPreloaded()) {
                // remove preloaded style
                const style = document.querySelector('style[id="preloaded-theme"]');
                if (style) {
                    style.remove();
                }
            };
            // broadcast theme change
            switchThemeListeners.forEach(function (listener) {
                listener(Theme);
            });
        });
        
    } else {
        // set light theme
        // remove dark theme style
        RemoveDarkCss();
        localStorage.removeItem("dark-theme-css")
        if (IsDarkCSSPreloaded()) {
            // remove preloaded style
            const style = document.querySelector('style[id="preloaded-theme"]');
            if (style) {
                style.remove();
            }
        }
        // broadcast theme change
        switchThemeListeners.forEach(function (listener) {
            listener(Theme);
        });

    }
}

function IsDarkCSSPreloaded() {
    const style = document.querySelector('style[id="preloaded-theme"]');
    return style;
}

function GetTheme() {
    return localStorage.getItem('theme') || "light";
}

function themeSwitchClick(e) {
    e.preventDefault();
    const current_theme = GetTheme();
    console.log("current theme:" + current_theme);
    if (current_theme == "light") {
        SetTheme("dark");
    } else {
        console.log("set light theme");
        SetTheme("light");
    }
}

function addThemeSwitchBroadcastListener(callback) {
    switchThemeListeners.push(callback)
}

function addContextMenuItem(decisionFunction, title, callback) {
    contextMenuList.push({
        decisionFunction: decisionFunction,
        title: title,
        callback: callback
    });
}

SetTheme(GetTheme());
init();