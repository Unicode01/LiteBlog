// render variables from server
var Card_max_width = parseInt(`{{global:card_max_width}}`); //w
var Card_max_height = parseInt(`{{global:card_max_height}}`);
var Card_min_width = parseInt(`{{global:card_min_width}}`); //w
var Card_min_height = parseInt(`{{global:card_min_height}}`);
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

    contextMenuList.forEach(function (item) {
        if (item.decisionFunction(event)) {
            // add menu item
            var menu_item = document.createElement('div');
            menu_item.classList.add('menu-item');
            menu_item.innerHTML = '<a class="link" href="#">' + item.title + '</a>';
            menu_item.addEventListener('click', function (event2) {
                event2.preventDefault();
                item.callback(event2);
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
    const menu_visual_width = 100;
    var screeen_width = window.screen.availWidth;
    // var screeen_height = window.screen.height;
    // console.log(screeen_width,menu_x);
    if (menu_x + menu_visual_width > screeen_width) {
        context_menu_doc.style.transform = "translateX(-100%)";
    }
    // if (menu_y + context_menu_doc.offsetHeight > screeen_height) {
    //     menu_y = menu_y - context_menu_doc.offsetHeight;
    // }
    context_menu_doc.style.left = menu_x + 'px';
    context_menu_doc.style.top = menu_y + 'px';
    // append to body
    document.body.appendChild(context_menu_doc);
    // force rerender
    context_menu_doc.offsetHeight;
    // get menu height
    var menu_height = context_menu_doc.offsetHeight;
    // force rerender
    context_menu_doc.offsetHeight;
    // set menu height
    context_menu_doc.style.height = '0px';
    // force rerender
    context_menu_doc.offsetHeight;
    // set menu height
    context_menu_doc.style.height = menu_height + 'px';
    // context_menu_doc.style.transform = "scale(1)";

}

function addContextMenuListener() {
    // add copy option
    // check selection then add copy option
    addContextMenuItem(function (event) {
        window._copy_selection = window.getSelection().toString();
        return window._copy_selection != ''
    }, "Copy", function (event) {
        event.preventDefault();
        const string = window._copy_selection;
        copyText(string);
    });
    // add reload option
    addContextMenuItem(function (event) {
        return true;
    }, "Reload", function (event) {
        event.preventDefault();
        location.reload();
    });
    // add card copy link option
    addContextMenuItem(function (event) {
        // check if cursor is in card
        const cards = document.querySelectorAll('.card-container');
        let active = false;
        cards.forEach(card => {
            const cursorX = event.clientX;
            const cursorY = event.clientY;
            const cardX = card.getBoundingClientRect().left;
            const cardY = card.getBoundingClientRect().top;
            const cardWidth = card.getBoundingClientRect().width;
            const cardHeight = card.getBoundingClientRect().height;
            if (cursorX > cardX && cursorX < cardX + cardWidth && cursorY > cardY && cursorY < cardY + cardHeight) {
                // check if link exists
                const linkdom = card.querySelector('.link');
                console.log(linkdom);
                if (linkdom) {
                    window._copy_link = linkdom.href;
                    active = true;
                    return true;
                }
            }
        });
        return active;
    }, "Copy Link", function (event) {
        event.preventDefault();
        const link = window._copy_link;
        console.log(link);
        // copy link to clipboard
        copyText(link);
    });
    // add delete card option
    addContextMenuItem(function (event) {
        // check if cursor is in card
        const cards = document.querySelectorAll('.card-container');
        let active = false;
        cards.forEach(card => {
            const cursorX = event.clientX;
            const cursorY = event.clientY;
            const cardX = card.getBoundingClientRect().left;
            const cardY = card.getBoundingClientRect().top;
            const cardWidth = card.getBoundingClientRect().width;
            const cardHeight = card.getBoundingClientRect().height;
            if (cursorX > cardX && cursorX < cardX + cardWidth && cursorY > cardY && cursorY < cardY + cardHeight) {
                // check if in edit mode
                if (window._editMode) {
                    window._delete_selected_card = card;
                    active = true;
                    return true;
                }
            }
        });
        return active;
    }, "Delete Card", function (event) {
        event.preventDefault();
        // show confirm dialog
        if (!confirm("Are you sure to delete this card?")) {
            return;
        }
        DeleteCard(window._delete_selected_card.getAttribute("card-id"), function(success){
            if (success) {
                console.log("card deleted");
                window._delete_selected_card.remove();
                window.Notify.add("Card deleted", {
                    type: "success",
                    timeout: 3000,
                });
            } else {
                console.log("failed to delete card");
                window.Notify.add("Failed to delete card", {
                    type: "error",
                    timeout: 3000,
                });
            }
        })
    });
    // add edit card option
    addContextMenuItem(function (event) {
        // check if cursor is in card
        const cards = document.querySelectorAll('.card-container');
        let active = false;
        cards.forEach(card => {
            const cursorX = event.clientX;
            const cursorY = event.clientY;
            const cardX = card.getBoundingClientRect().left;
            const cardY = card.getBoundingClientRect().top;
            const cardWidth = card.getBoundingClientRect().width;
            const cardHeight = card.getBoundingClientRect().height;
            if (cursorX > cardX && cursorX < cardX + cardWidth && cursorY > cardY && cursorY < cardY + cardHeight) {
                // check if in edit mode
                if (window._editMode) {
                    window._edit_selected_card = card;
                    active = true;
                    return true;
                }
            }
        });
        return active;
    }, "Edit Card", function (event) {
        event.preventDefault();
        EditCard(window._edit_selected_card.getAttribute("card-id"));
    });
    // add add card option
    addContextMenuItem(function (event) {
        // check if in edit mode
        if (window._editMode) {
            return true;
        }
    }, "Add Card", function (event) {
        event.preventDefault();
        AddCard();
    });
    // add add article option
    addContextMenuItem(function (event) {
        // check if in edit mode
        if (window._editMode) {
            return true;
        }
    }, "Add Article", function (event) {
        event.preventDefault();
        location.href = '/addarticle.html';
    });
    // add edit custom settings option
    addContextMenuItem(function (event) {
        // check if in edit mode
        if (window._editMode) {
            return true;
        }
    }, "Custom Settings", function (event) {
        event.preventDefault();
        EditCustomSettings();
    });
    // add edit mode option
    addContextMenuItem(function (event) {
        // check if in edit mode
        const editmodeExist = document.getElementById('edit-button');
        if (editmodeExist && !window._editMode && (location.pathname == '/index.html' || location.pathname === '/')) {
            return true;
        }
        return false;
    }, "Edit Mode", function (event) {
        event.preventDefault();
        var editmode = document.getElementById('edit-button');
        editmode?.click();
    });
    // add exit edit mode option
    addContextMenuItem(function (event) {
        // check if in edit mode
        const editmodeExist = document.getElementById('edit-button');
        if (editmodeExist && window._editMode && (location.pathname == '/index.html' || location.pathname === '/')) {
            return true;
        }
        return false;
    }, "Exit Edit Mode", function (event) {
        event.preventDefault();
        var editmode = document.getElementById('edit-button');
        editmode?.click();
    });
    // add save article option
    addContextMenuItem(function (event) {
        // check if in addarticle.html or editarticle.html
        if (location.pathname == '/addarticle.html' || location.pathname == '/editarticle.html') {
            return true;
        }
    }, "Save Article", function (event) {
        event.preventDefault();
        SaveArticle();
    });
    // add edit article option
    addContextMenuItem(function (event) {
        // check if in /articles/
        if (location.pathname.startsWith('/articles/') && GetAccessPathAndToken(true) != null) {
            return true;
        }
    }, "Edit Article", function (event) {
        event.preventDefault();
        location.href = '/editarticle.html?article_id=' + location.pathname.split('/')[2];
    });
    // add delete article option
    addContextMenuItem(function (event) {
        // check if in /articles/
        if (location.pathname.startsWith('/articles/') && GetAccessPathAndToken(true) != null) {
            return true;
        }
    }, "Delete Article", function (event) {
        event.preventDefault();
        // show confirm dialog
        if (!confirm("Are you sure to delete this article?")) {
            return;
        }
        DeleteArticleAPI(location.pathname.split('/')[2], function (result) {
            if (result) {
                console.log("article deleted");
                location.href = '/';
            } else {
                console.log("failed to delete article");
            }
        });
    });
    // add add comment option
    addContextMenuItem(function (event) {
        // check if in /articles/
        if (location.pathname.startsWith('/articles/')) {
            return true;
        }
    }, "Add Comment", function (event) {
        event.preventDefault();
        window.CommentReplyTo = "";
        ShowCommentInputBox();
    });
    // add delete comment option
    addContextMenuItem(function (event) {
        // check if cursor is in comment doms
        const comment_doms = document.querySelectorAll(".article-comment")
        active = false;
        comment_doms.forEach(comment_dom => {
            const cursorX = event.clientX;
            const cursorY = event.clientY;
            const commentX = comment_dom.getBoundingClientRect().left;
            const commentY = comment_dom.getBoundingClientRect().top;
            const commentWidth = comment_dom.getBoundingClientRect().width;
            const commentHeight = comment_dom.getBoundingClientRect().height;
            if (cursorX > commentX && cursorX < commentX + commentWidth && cursorY > commentY && cursorY < commentY + commentHeight) {
                // check if in edit mode
                if (GetAccessPathAndToken(true) != null) {
                    window._delete_selected_comment = comment_dom;
                    active = true;
                    return true;
                }
            }
        });
        return active;
    }, "Delete Comment", function (event) { 
        event.preventDefault();
        // show confirm dialog
        if (!confirm("Are you sure to delete this comment?")) {
            return;
        }
        const comment_id = window._delete_selected_comment.getAttribute("comment-id");
        DeleteCommentAPI(comment_id, function (result) {
            if (result) {
                console.log("comment deleted");
                window.Notify.add("Comment deleted", {
                    type: "success",
                    timeout: 3000,
                });
                window._delete_selected_comment.remove();
            } else {
                console.log("failed to delete comment");
                window.Notify.add("Failed to delete comment", {
                    type: "error",
                    timeout: 3000,
                });
            }
        });
    });
    
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
    // add context menu listener
    addContextMenuListener();
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
        max = 7;
        if (historyItems.length > max) {
            historyItems = historyItems.slice(0, max);
            // save
            localStorage.setItem('history', JSON.stringify(historyItems));
        }
        historyItems.forEach(function (item) {
            if (item.url == "" || item.title == "") {
                return;
            }
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
        // force rerender
        menu_doc.offsetHeight;
        // get menu height
        const menu_height = menu_doc.offsetHeight;
        // force rerender
        menu_doc.offsetHeight;
        // set menu height
        menu_doc.style.height = '0px';
        // force rerender
        menu_doc.offsetHeight;
        // set menu height
        menu_doc.style.height = menu_height + 'px';

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
        window.Notify.add("Copied to clipboard!",{
            type: "success",
            timeout: 2000,
        })
    } catch (err) {
        // 现代 API 失败时回退到旧方法
        const success = document.execCommand("copy");
        if (success) {
            console.log("link copied to clipboard:" + text);
            window.Notify.add("Copied to clipboard!",{
                type: "success",
                timeout: 2000,
            })
        } else {
            console.log("failed to copy link:" + text);
        }
    }
}

let abortController = null;
function InsertDarkCss(callback) {
    if (abortController) {
        abortController.abort();
    }
    
    abortController = new AbortController();
    const signal = abortController.signal;

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
    
    fetch('/css/dark.css', { signal })
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
    const link = document.querySelectorAll('style[id="dark-theme"]');
    link.forEach(function (item) {
        item.remove();
    });
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
        window.Notify.add("Dark theme enabled", {
            type: "success",
            timeout: 2000,
        });
    } else {
        console.log("set light theme");
        SetTheme("light");
        window.Notify.add("Light theme enabled", {
            type: "success",
            timeout: 2000,
        });
    }
}

function addThemeSwitchBroadcastListener(callback) {
    switchThemeListeners.push(callback)
}

function addContextMenuItem(decisionFunction = function() {return true;}, title, callback) {
    contextMenuList.push({
        decisionFunction: decisionFunction,
        title: title,
        callback: callback
    });
}

function InitNotifyModule() {
    window.Notify = new Notify({
        position: "top-center",
        notifyMargin: 5,
        notifyIconSize: 25,
        notifyMessageMarginRight: 20,
        maxList: 5,
    }, {
        timeout: 3000,
        onClick: function (notify) {
            window.Notify.remove(notify.id);
        },
    });
}

class Notify {
    constructor(settings, defaultOptions) {
        this.notifyList = {}; // id => notifyNode
        this.domParser = new DOMParser();
        this.current_theme = GetTheme();
        this.notifyCounter = 0;
        this.notiContainer = this.domParser.parseFromString(`
            <div class="notify-container">

            </div>
            `, "text/html").body.firstChild;
        this.basicNotify = this.domParser.parseFromString(`
            <div class="notify" notify-id="">
                <div class="notify-icon">
                    <img src="/img/notify-info.svg" alt="">
                </div>
                <div class="notify-content">
                    <div class="notify-message"></div>
                </div>
                
            </div>
            `, "text/html").body.firstChild;
        this.Settings = Object.assign({}, {
            containerHeight: 400,
            containerWidth: 300,
            notifyWidth: 250,
            notifyHeight: 45,
            notifyMargin: 10,
            notifyFontSize: 14,
            notifyIconSize: 20,
            notifyMessageMarginRight: 10,
            progressMode: "center",  // left,center,right
            progressColor: "#007BFF",
            maxList: 5,
            animationDuration: 300,
            position: "top-left", // top-left, top-center, top-right, bottom-left, bottom-center, bottom-right
            extraStyle: null,
        }, settings);
        this.defaultOptions = Object.assign({},{
            icon: '/img/notify-info.svg',
            type: 'info', // success, warning, error, info
            timeout: 5000, // 0: never close, otherwise in milliseconds
            keepAlive: false, // true: 不会因maxList限制而自动关闭，false: 会自动关闭
            onClick: null,
            onRemove: null,
            onTimeout: null,
            onShow: null,
            onHover: null,
            extraStyle: null,
        }, defaultOptions);
        
        // set properties
        this.notiContainer.classList.add(this.Settings.position);
        this.notiContainer.style.setProperty("--notify-container-width", this.Settings.containerWidth + "px");
        this.notiContainer.style.setProperty("--notify-container-height", this.Settings.containerHeight + "px");
        this.notiContainer.style.setProperty("--notify-width", this.Settings.notifyWidth + "px");
        this.notiContainer.style.setProperty("--notify-height", this.Settings.notifyHeight + "px");
        this.notiContainer.style.setProperty("--notify-margin", this.Settings.notifyMargin + "px");
        this.notiContainer.style.setProperty("--notify-message-margin-right", this.Settings.notifyMessageMarginRight + "px");
        this.notiContainer.style.setProperty("--notify-icon-size", this.Settings.notifyIconSize + "px");
        this.notiContainer.style.setProperty("--notify-font-size", this.Settings.notifyFontSize + "px");
        this.notiContainer.style.setProperty("--notify-animation-duration", this.Settings.animationDuration + "ms");
        this.notiContainer.style.setProperty("--progress-color", this.Settings.progressColor)
        this.notiContainer.style.setProperty("--notify-border-radius", "8px")
        this.notiContainer.style.setProperty("--progress-transform-origin", "center")
        this.notiContainer.style.setProperty("--progress-scale", "1")

        // set extra style
        if (this.Settings.extraStyle) {
            for (let key in this.Settings.extraStyle) {
                this.notiContainer.style[key] = this.Settings.extraStyle[key];
            }
        }

        // set theme
        this.notiContainer.classList.add(this.current_theme + "-theme");

        document.body.appendChild(this.notiContainer);

        // add event listener
        addThemeSwitchBroadcastListener((theme) => {
            this.notiContainer.classList.remove(this.current_theme + "-theme");
            this.notiContainer.classList.add(theme + "-theme");
            this.current_theme = theme;
        });
    }

    add(message, options) {
        let newNotify = {
            message: message,
            options: Object.assign({}, this.defaultOptions, options),
            id: Math.random().toString(36).slice(2, 10), // get random id
            index: this.notifyCounter++,
            status: "showing", // showing, removed
        };
        if (!options?.icon) {
            switch (newNotify.options.type) {
                case "success":
                    newNotify.options.icon = "/img/notify-success.svg";
                    break;
                case "warning":
                    newNotify.options.icon = "/img/notify-warning.svg";
                    break;
                case "error":
                    newNotify.options.icon = "/img/notify-error.svg";
                    break;
                default:
                    newNotify.options.icon = "/img/notify-info.svg";
                    break;
            }
        }
        
        // console.log(newNotify);
        this.notifyList[newNotify.id] = newNotify;
        let childNode = this.basicNotify.cloneNode(true);
        // set properties
        childNode.querySelector(".notify-icon img").src = newNotify.options.icon;
        childNode.classList.add(newNotify.options.type);
        childNode.querySelector(".notify-message").textContent = message;
        childNode.setAttribute("notify-id", newNotify.id);
        // set extra style
        if (newNotify.options.extraStyle) {
            for (let key in newNotify.options.extraStyle) {
                childNode.style[key] = newNotify.options.extraStyle[key];
            }
        }
        // add event listener
        childNode.addEventListener("click", (e) => {
            e.stopPropagation();
            this.onEvent("click", newNotify.id, e);
        });
        childNode.addEventListener("mouseenter", (e) => {
            e.stopPropagation();
            this.onEvent("hover", newNotify.id, e);
        });
        childNode.addEventListener("mouseleave", (e) => {
            e.stopPropagation();
            this.onEvent("hover", newNotify.id, e);
        });
        
        this.notiContainer.insertBefore(childNode, this.notiContainer.firstChild);

        // check timeout
        if (newNotify.options.timeout > 0) {
            setTimeout(() => {
                this.onEvent("timeout", newNotify.id, null);
                this.remove(newNotify.id);
            }, newNotify.options.timeout);
        }

        // check onShow
        if (newNotify.options.onShow) {
            this.onEvent("show", newNotify.id, null);
        }

        // add animation class
        childNode.classList.add("notify-show");

        // add remove animation class timeout
        setTimeout(() => {
            childNode.classList.remove("notify-show");
        }, this.Settings.animationDuration);
        
        // check if need to close
        if (this.Settings.maxList > 0 && Object.keys(this.notifyList).length > this.Settings.maxList) {
            let selected = [];
            let needToRemove = Object.keys(this.notifyList).length - this.Settings.maxList;
        
            console.log("need to remove:", needToRemove);
            for (let id in this.notifyList) {
                if (selected.length >= needToRemove) {
                    break;
                }
                if (this.notifyList[id].status == "showing" && !this.notifyList[id].options.keepAlive) {
                    selected.push(this.notifyList[id]);
                }
            }
            console.log("selected:", selected);
            for (let i = 0; i < selected.length; i++) {
                console.log("remove oldest notify:",selected[i].index);
                this.remove(selected[i].id);
            }
        }

        // set properties
        if (newNotify.options.timeout > 0) {
            childNode.style.setProperty("--notify-duration", newNotify.options.timeout + "ms")
            switch (this.Settings.progressMode) {
                case "center":
                    childNode.offsetHeight // force reflow
                    childNode.style.setProperty("--progress-scale", "0")
                    break;
                case "left":
                    childNode.style.setProperty("--progress-transform-origin", "left")
                    childNode.offsetHeight // force reflow
                    childNode.style.setProperty("--progress-scale", "0")
                    break;
                case "right":
                    childNode.style.setProperty("--progress-transform-origin", "right")
                    childNode.offsetHeight // force reflow
                    childNode.style.setProperty("--progress-scale", "0")
                    break;
            }
            
        } else {
            // remove progress bar
            childNode.style.setProperty("--notify-duration", "0ms")
            childNode.style.setProperty("--progress-scale", "0")
        }
        

        return newNotify.id;
    }

    alert(message) {
        this.add(message, {
            type: "info",
            timeout: 3000,
            keepAlive: true,
            onClick: (notify, event) => {
                this.remove(notify.id);
            }
        });
    }

    error(message) {
        this.add(message, {
            type: "error",
            keepAlive: true,
            onClick: (notify, event) => {
                this.remove(notify.id);
            }
        });
    }

    success(message) {
        this.add(message, {
            type: "success",
            keepAlive: true,
            onClick: (notify, event) => {
                this.remove(notify.id);
            }
        });
    }

    warning(message) {
        this.add(message, {
            type: "warning",
            keepAlive: true,
            onClick: (notify, event) => {
                this.remove(notify.id);
            }
        });
    }

    info(message) {
        this.add(message, {
            type: "info",
            onClick: (notify, event) => {
                this.remove(notify.id);
            }
        });
    }

    remove(id) {
        if (!this.notifyList[id] || this.notifyList[id].status == "removed") { // if not exist, return
            return;
        }
        this.notifyList[id].status = "removed";
        this.onEvent("remove", id, null);
        // query node
        const node = this.notiContainer.querySelector('[notify-id="' + id + '"]');
        if (node) {
            // remove node
            // node.remove();
            // remove from list
            delete this.notifyList[id];
            // show remove animation
            node.classList.add("notify-remove");
            setTimeout(() => {
                node.remove();
            }, this.Settings.animationDuration);
        }
    }

    onEvent(event, id, broswerEvent) {
        let notify = null;
        switch (event) {
            case "click":
                notify = this.notifyList[id];
                if (notify?.options.onClick) {
                    notify.options.onClick(notify, broswerEvent);
                }
                break;
            case "remove":
                notify = this.notifyList[id];
                if (notify?.options.onRemove) {
                    notify.options.onRemove(notify);
                }
                break;
            case "timeout":
                notify = this.notifyList[id];
                if (notify?.options.onTimeout) {
                    notify.options.onTimeout(notify);
                }
                break;
            case "show":
                notify = this.notifyList[id];
                if (notify?.options.onShow) {
                    notify.options.onShow(notify);
                }
                break;
            case "hover":
                notify = this.notifyList[id];
                if (notify?.options.onHover) {
                    notify.options.onHover(notify, broswerEvent);
                }
                break;
            default:
                break;
        }
    }

    destroy() {
        for (let id in this.notifyList) {
            this.remove(id);
        }
        this.notiContainer.remove();
    }
}

SetTheme(GetTheme());
InitNotifyModule();
init();