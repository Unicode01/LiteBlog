// render variable
var NewCommentInputBoxHTML = `
{{file:new_comment_input_box}}
`
var CF_Site_key = "{{global:cf_site_key}}"
var Goole_reCaptcha_Site_key = "{{global:google_site_key}}"
var comment_check_type = "{{global:comment_check_type}}"
// end of render variable

function AddArticleAPI(title, author, content, contentHTML, extraFlags, callback) {
    const result = GetAccessPathAndToken();
    if (!result) {
        console.log("Access path and token are required.");
        return false;
    }
    const { path, token } = result;
    // console.log("Access path: " + path);
    // console.log("Access token: " + token);
    const api_dic = window.location.origin + "/" + path;
    const api_add_article = api_dic + "/add_article";
    const data = {
        token: token,
        article: {
            title: title,
            author: author,
            content: content,
            content_html: contentHTML,
            extra_flags: extraFlags
        }
    }
    console.log(data);
    fetch(api_add_article, {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify(data)
    })
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP ERR，Code：${response.status}`);
            }
            return response.json();
        })
        .then(data => {
            console.log(data);
            callback(data);
        })
        .catch(error => {
            console.log(error);
            window.Notify.error(error.message);
            callback("");
        });
}

function EditArticleAPI(article_id, title, author, content, contentHTML, extraFlags, callback) {
    const result = GetAccessPathAndToken();
    if (!result) {
        console.log("Access path and token are required.");
        return false;
    }
    const { path, token } = result;
    // console.log("Access path: " + path);
    // console.log("Access token: " + token);
    const api_dic = window.location.origin + "/" + path;
    const api_edit_article = api_dic + "/edit_article";
    const data = {
        token: token,
        article: {
            article_id: article_id,
            title: title,
            author: author,
            content: content,
            content_html: contentHTML,
            extra_flags: extraFlags
        }
    }
    console.log(data);
    fetch(api_edit_article, {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify(data)
    })
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP ERR，Code：${response.status}`);
            }
            return response.text();
        })
        .then(data => {
            console.log(data);
            callback(data);
        })
        .catch(error => {
            console.log(error);
            window.Notify.error(error.message);
            callback("");
        });
}

function GetArticleAPI(article_id, callback) {
    const result = GetAccessPathAndToken();
    if (!result) {
        console.log("Access path and token are required.");
        return false;
    }
    const { path, token } = result;
    // console.log("Access path: " + path);
    // console.log("Access token: " + token);
    const api_dic = window.location.origin + "/" + path;
    const api_get_article = api_dic + "/get_article";
    const data = {
        token: token,
        article_id: article_id
    }
    console.log(data);
    fetch(api_get_article, {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify(data)
    })
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP ERR，Code：${response.status}`);
            }
            return response.json();
        })
        .then(data => {
            callback(data);
        })
        .catch(error => {
            console.log(error);
            window.Notify.error(error.message);
            callback("");
        });
}

function AddCommentAPI(article_id, reply_to, author, email, content, subscribed, callback) {
    path = "api/v1"
    const api_dic = window.location.origin + "/" + path;
    const api_add_comment = api_dic + "/add_comment";
    const data = {
        verify_token: window.comment_token,
        article_id: article_id,
        author: author,
        email: email,
        reply_to: reply_to,
        subscribed: subscribed,
        content: content
    }
    console.log(data);
    fetch(api_add_comment, {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify(data)
    })
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP ERR，Code：${response.status}`);
            }
            return response.text();
        })
        .then(data => {
            console.log(data);
            callback(data);
        })
        .catch(error => {
            console.log(error);
            window.Notify.error(error.message);
            callback("");
        });
}

function DeleteCommentAPI(comment_id, callback) {
    const result = GetAccessPathAndToken();
    if (!result) {
        console.log("Access path and token are required.");
        return false;
    }
    const { path, token } = result;
    // console.log("Access path: " + path);
    // console.log("Access token: " + token);
    const api_dic = window.location.origin + "/" + path;
    const api_delete_comment = api_dic + "/delete_comment";
    const data = {
        token: token,
        article_id: getQueryVariable("article_id"),
        comment_id: comment_id
    }
    console.log(data);
    fetch(api_delete_comment, {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify(data)
    })
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP ERR，Code：${response.status}`);
            }
            return response.text();
        })
        .then(data => {
            console.log(data);
            callback(data);
        })
        .catch(error => {
            console.log(error);
            window.Notify.error(error.message);
            callback("");
        });
}

function DeleteArticleAPI(article_id, callback) {
    const result = GetAccessPathAndToken();
    if (!result) {
        console.log("Access path and token are required.");
        return false;
    }
    const { path, token } = result;
    // console.log("Access path: " + path);
    // console.log("Access token: " + token);
    const api_dic = window.location.origin + "/" + path;
    const api_delete_article = api_dic + "/delete_article";
    const data = {
        token: token,
        article_id: article_id
    }
    console.log(data);
    fetch(api_delete_article, {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify(data)
    })
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP ERR，Code：${response.status}`);
            }
            return response.text();
        })
        .then(data => {
            console.log(data);
            callback(data);
        })
        .catch(error => {
            console.log(error);
            window.Notify.error(error.message);
            callback("");
        });
}

function SaveArticle(implicitlySave = false) {
    const editor_title = document.querySelector('.title-input').value;
    const author_input = document.querySelector('.author-input').value;
    var rendered_content = "";
    if (marked) {
        // reder markdown content
        const editor_content = document.querySelector('.markdown-textarea');
        const content_value = editor_content.value;
        rendered_content = marked.parse(content_value);
    } else {
        rendered_content = document.querySelector('.article-content').innerHTML;
    }
    const markdown_input = document.querySelector('#markdown-input').value;

    extra_flags = {};
    // set extra flags
    // get all text
    let renderedContext = document.querySelector('.article-content').textContent;
    // get article language code
    let article_language = detectLanguage(renderedContext);
    extra_flags.language_code = article_language["language"];
    // get article description
    console.log(renderedContext);
    let article_description = renderedContext.slice(0, 100); // get first 100 characters
    // set extra flags
    extra_flags.article_description = article_description;

    // check if in /addarticle.html
    if (location.pathname === "/addarticle.html") {
        // add article
        AddArticleAPI(editor_title, author_input, markdown_input, rendered_content, extra_flags, function (result) {
            if (result != "") {
                console.log(result);
                if (implicitlySave) {
                    window.Notify.add("Article added successfully!", { type: "success" });
                    return;
                }
                window.Notify.add("Article added successfully!", {
                    type: "success",
                    timeout: 3000,
                    onClick: function () {
                        // jump to article page
                        article_id = result.article_id;
                        console.log(article_id);
                        // clear local storage
                        localStorage.removeItem("localStoredArticle");
                        // jump to article page
                        window.location.href = "/articles/" + article_id;
                    },
                    onTimeout: function () {
                        // jump to article page
                        article_id = result.article_id;
                        console.log(article_id);
                        // clear local storage
                        localStorage.removeItem("localStoredArticle");
                        // jump to article page
                        window.location.href = "/articles/" + article_id;
                    },
                    extraStyle: {
                        "cursor": "pointer"
                    }
                });

            } else {
                window.Notify.add("Article added failed!", {
                    type: "error"
                });
            }
        });
    } else if (location.pathname === "/editarticle.html") {
        article_id = getQueryVariable("article_id");
        // edit article
        EditArticleAPI(article_id, editor_title, author_input, markdown_input, rendered_content, extra_flags, function (result) {
            if (implicitlySave) {
                window.Notify.add("Article edited successfully!", { type: "success" });
                return;
            }
            if (result != "") {
                window.Notify.add("Article edited successfully!", {
                    type: "success",
                    timeout: 3000,
                    onClick: () => {
                        console.log(result);
                        // jump to article page
                        window.location.href = "/articles/" + article_id;
                    },
                    onTimeout: () => {
                        console.log(result);
                        // jump to article page
                        window.location.href = "/articles/" + article_id;
                    },
                    extraStyle: {
                        "cursor": "pointer"
                    }
                });
            } else {
                window.Notify.add("Article edited failed!", {
                    type: "error"
                });
            }
        });
    }

}

function getQueryVariable(variable) {
    if (window.location.pathname === "/editarticle.html" || window.location.pathname === "/addarticles.html") {
        var query = window.location.search.substring(1);
        var vars = query.split("&");
        for (var i = 0; i < vars.length; i++) {
            var pair = vars[i].split("=");
            if (pair[0] == variable) { return pair[1]; }
        }
        return (false);
    } else if (window.location.pathname.startsWith("/articles/")) {
        var article_id = window.location.pathname.split("/")[2];
        return article_id;
    }

}

function ShowCommentInputBox() {
    // Load marked.js if not already loaded
    if (typeof marked === 'undefined') {
        // Check if marked.js script is already being loaded
        const existingScript = document.querySelector('script[src="/js/marked.min.js"]');
        if (!existingScript) {
            const markedScript = document.createElement('script');
            markedScript.src = '/js/marked.min.js';
            markedScript.async = false;
            markedScript.onload = function () {
                console.log('marked.js loaded successfully');
            };
            markedScript.onerror = function () {
                console.error('Failed to load marked.js');
            };
            document.head.appendChild(markedScript);
        }
    }

    const domparse = new DOMParser();
    const CommentInputBoxDoc = domparse.parseFromString(NewCommentInputBoxHTML, "text/html").body.firstChild;
    const article_id = getQueryVariable("article_id");
    if (!CommentInputBoxDoc || !article_id) {
        return;
    }
    if (comment_check_type == "cloudflare_turnstile") {
        const CommentBoxPre = document.querySelector(".comment-input-box");
        CommentBoxPre?.remove()
        // set validator class to `inner-cf-turnstile`
        const validator = CommentInputBoxDoc.querySelector("#comment-validator")
        validator.classList.add("inner-cf-turnstile");
        CommentInputBoxDoc.querySelector(".inner-cf-turnstile").setAttribute("data-sitekey", CF_Site_key);
        window.onloadTurnstileCallback = function () {
            turnstile.render(".inner-cf-turnstile", {
                sitekey: CF_Site_key,
                callback: function (token) {
                    console.log(`Challenge Success ${token}`);
                    window.comment_token = token;
                },
            });
        };
        // append turnstile script
        const turnstile_script = document.createElement("script");
        turnstile_script.src = "https://challenges.cloudflare.com/turnstile/v0/api.js?onload=onloadTurnstileCallback";
        document.body.appendChild(CommentInputBoxDoc);
        CommentInputBoxDoc.appendChild(turnstile_script);
    } else if (comment_check_type == "google_recaptcha") {
        const CommentBoxPre = document.querySelector(".comment-input-box");
        CommentBoxPre?.remove()
        // append recaptcha script
        const recaptcha_script = document.createElement("script");
        recaptcha_script.src = "https://www.google.com/recaptcha/api.js?render=" + Goole_reCaptcha_Site_key;
        document.body.appendChild(CommentInputBoxDoc);
        CommentInputBoxDoc.appendChild(recaptcha_script);
    } else {
        window.Notify.add("Comment has been disabled.", {
            type: "info"
        });
    }
}

function CancelCommentInputBox() {
    const CommentBoxPre = document.querySelector(".comment-input-box");
    CommentBoxPre?.remove()
    window.CommentReplyTo = "";
}

function OnAddCommentButtonClick() {
    const article_id = getQueryVariable("article_id");
    var author_input = document.querySelector('#add-comment-author').value;
    var email_address = document.querySelector('#add-comment-emailaddress').value;
    var content_input = document.querySelector('#add-comment-text').value;
    var subscribed = document.querySelector('.email-subscribe-button').checked;
    if (!isAvailableEmailAddress(email_address)) {
        window.Notify.add("Invalid email address.", {
            type: "error"
        });
        return;
    }
    // parse markdown to HTML
    var content_html = "";
    if (typeof marked !== 'undefined') {
        content_html = marked.parse(content_input);
    } else {
        // fallback: escape string if marked is not available
        content_html = escapeString(content_input);
    }
    // check if google recaptcha
    if (comment_check_type == "google_recaptcha") {

        grecaptcha.ready(function () {
            grecaptcha.execute(Goole_reCaptcha_Site_key, { action: 'submit' }).then(function (token) {
                // Add your logic to submit to your backend server here.
                if (!article_id || !author_input || !content_html || !token) {
                    console.log("Article id, author, content and token are required.");
                    window.Notify.add("Please fill in all fields.", {
                        type: "error"
                    });
                    return;
                }
                window.comment_token = token;
                AddCommentAPI(article_id, window.CommentReplyTo, author_input, email_address, content_html, subscribed, function (result) {
                    if (result != "") {
                        console.log(result);
                        window.CommentReplyTo = "";
                        window.Notify.add("Comment added successfully!", {
                            type: "success",
                            timeout: 3000,
                            onClick: function () {
                                // jump to article page
                                window.location.reload();
                            },
                            onTimeout: function () {
                                // jump to article page
                                window.location.reload();
                            },
                            extraStyle: {
                                "cursor": "pointer"
                            }
                        });
                        // remove comment input box
                        CancelCommentInputBox();
                    } else {
                        window.Notify.add("Failed to add comment.", {
                            type: "error"
                        });
                    }
                });
            });
        });
        return;
    } else if (comment_check_type == "cloudflare_turnstile") {
        if (!article_id || !author_input || !content_html || !window.comment_token) {
            console.log("Article id, author, content and token are required.");
            window.Notify.add("Please fill in all fields.", {
                type: "error"
            });
            return;
        }
        AddCommentAPI(article_id, window.CommentReplyTo, author_input, email_address, content_html, subscribed, function (result) {
            if (result != "") {
                console.log(result);
                window.CommentReplyTo = "";
                window.Notify.add("Comment added successfully!", {
                    type: "success",
                    timeout: 3000,
                    onClick: function () {
                        // jump to article page
                        window.location.reload();
                    },
                    onTimeout: function () {
                        // jump to article page
                        window.location.reload();
                    },
                    extraStyle: {
                        "cursor": "pointer"
                    }
                });
                // remove comment input box
                CancelCommentInputBox();
            } else {
                window.Notify.add("Failed to add comment.", {
                    type: "error"
                });
            }
        });
    }
}

function OnReplyButtonClick(comment_id) {
    console.log(comment_id);
    window.CommentReplyTo = comment_id;
    ShowCommentInputBox();
}

function isAvailableEmailAddress(email) {
    // 基础检查：非字符串、空值、无@符号直接返回false
    if (typeof email !== 'string' || !email) return false;
    if (email.indexOf('@') === -1) return false;

    // 分割本地部分和域名部分
    const parts = email.split('@');
    const localPart = parts[0];
    const domainPart = parts[1];

    // 检查分割结果有效性
    if (parts.length !== 2 || !localPart || !domainPart) return false;

    // 1. 本地部分验证
    const localRegex = /^[a-zA-Z0-9!#$%&'*+\-\/=?^_`{|}~]+(\.[a-zA-Z0-9!#$%&'*+\-\/=?^_`{|}~]+)*$/;
    if (
        // 长度检查 (1-64字符)
        localPart.length < 1 || localPart.length > 64 ||
        // 开头/结尾不能是点
        localPart.startsWith('.') || localPart.endsWith('.') ||
        // 连续点检查
        localPart.includes('..') ||
        // 字符有效性
        !localRegex.test(localPart)
    ) {
        return false;
    }

    // 2. 域名部分验证
    if (
        // 长度检查 (1-255字符)
        domainPart.length < 1 || domainPart.length > 255 ||
        // 开头/结尾不能是连字符或点
        domainPart.startsWith('-') || domainPart.endsWith('-') ||
        domainPart.startsWith('.') || domainPart.endsWith('.') ||
        // 连续点检查
        domainPart.includes('..')
    ) {
        return false;
    }

    // 域名标签分割验证
    const domainLabels = domainPart.split('.');
    const labelRegex = /^[a-zA-Z0-9](?:[a-zA-Z0-9\-]*[a-zA-Z0-9])?$/;

    for (const label of domainLabels) {
        if (
            // 标签长度检查 (1-63字符)
            label.length < 1 || label.length > 63 ||
            // 标签格式检查
            !labelRegex.test(label)
        ) {
            return false;
        }
    }

    // 顶级域名检查 (至少2个字母)
    const tld = domainLabels[domainLabels.length - 1];
    if (!/^[a-zA-Z]{2,}$/.test(tld)) {
        return false;
    }

    return true;
}

// 防止重复绑定
var _mouseMoveHandlerBound = false;

function RenderOutline() {
    const outlineTitle = document.querySelector('.outline-title');
    const outlineList = document.querySelector('.outline-list');
    const articleTitle = document.querySelector('.article-title');
    const articleDom = document.querySelector('.article-content');

    // 总是绑定鼠标移动事件（用于工具箱和大纲），但只绑定一次
    if (!_mouseMoveHandlerBound) {
        document.body.addEventListener('mousemove', MouseMoveHandler);
        _mouseMoveHandlerBound = true;
        console.log('MouseMoveHandler bound');
    }

    // console.log(outlineTitle, outlineList, articleTitle, articleDom);
    if (!outlineTitle || !outlineList || !articleTitle || !articleDom) {
        return;
    }
    generateOutline(articleDom, outlineList);
    outlineTitle.textContent = "Outline";
    if (location.pathname.startsWith("/articles/")) {
        // in the article page
        window.addEventListener('scroll', function () {
            const scrollTop = document.documentElement.scrollTop || document.body.scrollTop;
            const scrollReal = scrollTop + 60;
            // console.log(scrollReal);
            // check if scroll in article content
            const headings = articleDom.querySelectorAll('h1, h2, h3')
            for (let i = 0; i < headings.length; i++) {
                const heading = headings[i];
                const headingTop = heading.offsetTop;
                const headingHeight = heading.offsetHeight;
                if (scrollReal >= headingTop && scrollReal <= headingTop + headingHeight) {
                    // highlight outline item
                    heading.HighLightOutline();
                }
            }
        });
    }
}

function MouseMoveHandler(event) {
    const mouseX = event.clientX;
    const mouseY = event.clientY;
    const windowWidth = window.innerWidth;

    // 处理右侧大纲
    const outlineContainer = document.querySelector('.outline-container');
    if (outlineContainer) {
        const rect = outlineContainer.getBoundingClientRect();

        // 判断是否应该显示
        const shouldShow = mouseX > windowWidth - 50 ||
            (mouseX > rect.left && mouseX < rect.right && mouseY > rect.top && mouseY < rect.bottom);

        if (shouldShow) {
            outlineContainer.style.transform = 'translateX(0%)';
            // console.log('Outline: show');
        } else {
            outlineContainer.style.transform = 'translateX(150%)';
        }
    } else {
        console.log('Outline container not found');
    }

    // 处理左侧工具箱 - 编辑页面不处理
    const isEditPage = location.pathname === '/editarticle.html' || location.pathname === '/addarticle.html';
    if (!isEditPage) {
        const toolboxContainer = document.querySelector('.toolbox-container');
        if (toolboxContainer && toolboxContainer.style.display !== 'none') {
            const rect = toolboxContainer.getBoundingClientRect();

            // 判断是否应该显示
            const shouldShow = mouseX < 50 ||
                (mouseX > rect.left && mouseX < rect.right && mouseY > rect.top && mouseY < rect.bottom);

            if (shouldShow) {
                toolboxContainer.style.transform = 'translateY(-50%) translateX(0%)';
                // console.log('Toolbox: show');
            } else {
                toolboxContainer.style.transform = 'translateY(-50%) translateX(-150%)';
            }
        }
    }
}

function generateOutline(articleDom, outlineList) {
    // 获取所有标题元素
    const headings = articleDom.querySelectorAll('h1, h2, h3');

    // 清空现有内容
    outlineList.innerHTML = '';

    // 创建根列表
    const rootList = document.createElement('ul');
    rootList.classList.add('root-list');
    outlineList.appendChild(rootList);

    // 用于存储各级别的当前列表
    const listStack = [rootList];
    const levelStack = [0]; // 记录当前层级

    // 遍历所有标题
    headings.forEach(heading => {
        const level = parseInt(heading.tagName.substring(1));

        // 如果当前级别比栈顶级别小，需要回退
        while (level <= levelStack[levelStack.length - 1]) {
            listStack.pop();
            levelStack.pop();
        }

        // 创建列表项
        const listItem = document.createElement('li');
        listItem.classList.add(`level-${levelStack.length - 1}`);

        const itemDiv = document.createElement('div');
        itemDiv.classList.add('list-item');

        // // 创建切换按钮（如果有子项）
        // const toggleBtn = document.createElement('div');
        // toggleBtn.classList.add('toggle-btn');
        // toggleBtn.innerHTML = '<i class="fas fa-chevron-down"></i>';

        // 创建内容区域
        const contentDiv = document.createElement('div');
        contentDiv.classList.add('item-content');

        // const iconSpan = document.createElement('span');
        // iconSpan.classList.add('item-icon');
        // iconSpan.innerHTML = '<i class="far fa-file-alt"></i>';

        const textSpan = document.createElement('span');
        textSpan.classList.add('item-text');
        textSpan.textContent = heading.textContent;

        // 组装元素
        // contentDiv.appendChild(iconSpan);
        contentDiv.appendChild(textSpan);
        // itemDiv.appendChild(toggleBtn);
        itemDiv.appendChild(contentDiv);
        listItem.appendChild(itemDiv);

        // 添加到当前列表
        const currentList = listStack[listStack.length - 1];
        currentList.appendChild(listItem);

        // 创建子列表（如果有下一级）
        const subList = document.createElement('ul');
        listItem.appendChild(subList);

        // 更新栈
        listStack.push(subList);
        levelStack.push(level);

        // 添加点击事件
        itemDiv.addEventListener('click', function () {
            // heading.style.scrollMarginTop = '50px';
            // 滚动到对应标题
            heading.scrollIntoView({ behavior: 'smooth', block: 'start' });

            // 高亮显示
            // document.querySelectorAll('.list-item').forEach(el => {
            //     // el.style.background = 'none';
            //     el.classList.remove('active');
            // });
            // listItem.querySelector('.list-item').classList.add('active');
            // this.style.background = '#e3f2fd';
        });

        // // 添加展开/折叠事件
        // toggleBtn.addEventListener('click', function(e) {
        //     e.stopPropagation();
        //     listItem.classList.toggle('collapsed');
        // });

        // 添加HighLightOutline函数, 用于高亮显示当前标题的Outline
        heading.HighLightOutline = function () {
            // 高亮显示
            document.querySelectorAll('.list-item').forEach(el => {
                el.classList.remove('active');
            });
            listItem.querySelector('.list-item').classList.add('active');
        }
    });
}

function RenderHighlight() {
    // check if highlight.min.js has been loaded
    if (typeof hljs === 'undefined') {
        // load highlight.min.css
        if (!document.getElementById("article-code-viewer-style")) {
            const highlight_style = document.createElement("link");
            highlight_style.id = "article-code-viewer-style";
            highlight_style.href = "/css/" + GetTheme() + ".highlight.css";
            highlight_style.rel = "stylesheet";
            document.head.appendChild(highlight_style);
        }
        // load highlight.min.js
        const highlight_script = document.createElement("script");
        highlight_script.src = "/js/highlight.min.js";
        document.body.appendChild(highlight_script);
        // add highlight event listener
        highlight_script.addEventListener('load', function () {
            RenderHighlight();
        });
        return;
    }
    // select all code blocks
    document.querySelectorAll('pre code').forEach((el) => {
        hljs.highlightElement(el);
    });
}

function SwitchToRemoveEditDate() {
    // select article-date
    const articleDate = document.querySelector('.article-date');
    // select article-edit-date
    const articleEditDate = document.querySelector('.article-edit-date');
    // get article-edit-date text, remove 'ed. '
    const articleEditDateText = articleEditDate?.textContent.trim().slice(4);
    // get article-date text, remove 'pub. '
    const articleDateText = Array.from(articleDate.childNodes)
        .filter(node => node.nodeType === 3)
        .map(textNode => textNode.textContent.trim())
        .join(' ')
        .replace(/\s+/g, ' ')
        .slice(5)
        .slice(0, -1);
    // compare article-edit-date and article-date
    // console.log(articleEditDateText, articleDateText);
    if (articleEditDateText != articleDateText && location.pathname.startsWith("/articles/")) {
        // remove article-edit-date
        articleEditDate.style.opacity = 1;
    }
}

function AddImgEventListener() {
    articleImgs = document.querySelectorAll(".article-content img");
    articleImgs.forEach(function (img) {
        img.addEventListener("click", function (event) {
            show_big_photo(img.src);
        });
    });
}

function InitToolbox() {
    // 检测当前页面 - 编辑页面不显示工具箱
    const isEditPage = location.pathname === '/editarticle.html' || location.pathname === '/addarticle.html';
    if (isEditPage) {
        console.log('Toolbox disabled on edit pages');
        return;
    }

    // 保存为 HTML
    const saveHtmlBtn = document.querySelector('#save-html-btn');
    if (saveHtmlBtn) {
        saveHtmlBtn.addEventListener('click', function () {
            SaveArticleAsHTML();
        });
    }

    // 笔记功能
    const noteBtn = document.querySelector('#note-btn');
    if (noteBtn) {
        noteBtn.addEventListener('click', function () {
            ToggleNotesPanel();
        });
    }

    // 初始化笔记功能
    InitNotesFeature();
}

function SaveArticleAsHTML() {
    try {
        // 获取文章内容
        const articleTitle = document.querySelector('.article-title')?.textContent || '文章';
        const articleAuthor = document.querySelector('.article-author')?.textContent || '';
        const articleDate = document.querySelector('.article-date')?.textContent || '';
        const articleContent = document.querySelector('.article-content')?.innerHTML || '';

        // 构建完整的 HTML
        const htmlContent = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="author" content="${articleAuthor}">
    <title>${articleTitle}</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            line-height: 1.6;
            color: #333;
        }
        h1 { font-size: 2em; margin-bottom: 0.5em; }
        h2 { font-size: 1.5em; margin-top: 1.5em; }
        h3 { font-size: 1.25em; margin-top: 1.25em; }
        .article-header { margin-bottom: 2em; border-bottom: 1px solid #eee; padding-bottom: 1em; }
        .article-info { color: #666; font-size: 0.9em; margin-top: 0.5em; }
        img { max-width: 100%; height: auto; display: block; margin: 1.5em auto; border-radius: 6px; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; font-family: monospace; }
        pre { background: #f4f4f4; padding: 1em; border-radius: 6px; overflow-x: auto; }
        blockquote { border-left: 4px solid #ddd; margin: 1.5em 0; padding: 0.5em 1em; background: #f9f9f9; }
        table { width: 100%; border-collapse: collapse; margin: 1.5em 0; }
        th, td { padding: 0.75em; border: 1px solid #ddd; text-align: left; }
        th { background: #f4f4f4; font-weight: 600; }
    </style>
</head>
<body>
    <div class="article-header">
        <h1>${articleTitle}</h1>
        <div class="article-info">
            <div>Author: ${articleAuthor}</div>
            <div>${articleDate}</div>
        </div>
    </div>
    <div class="article-content">
        ${articleContent}
    </div>
</body>
</html>`;

        // 创建 Blob
        const blob = new Blob([htmlContent], { type: 'text/html;charset=utf-8' });
        const blobUrl = URL.createObjectURL(blob);

        // 创建下载链接
        const link = document.createElement('a');
        link.href = blobUrl;
        link.download = `${articleTitle}.html`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);

        // 清理
        setTimeout(() => URL.revokeObjectURL(blobUrl), 100);

        // 提示
        window.Notify.add("HTML file saved", {
            type: "success",
            timeout: 2000
        });
    } catch (error) {
        console.error('Save HTML failed:', error);
        window.Notify.add("Save failed: " + error.message, {
            type: "error",
            timeout: 3000
        });
    }
}

addThemeSwitchBroadcastListener(function (theme) {
    const styleDom = document.querySelector('#article-code-viewer-style');
    if (styleDom) {
        styleDom.href = `/css/${theme}.highlight.css`;
    } else {
        const style = document.createElement('link');
        style.id = 'article-code-viewer-style';
        style.rel = 'stylesheet';
        style.href = `/css/${theme}.highlight.css`;
        document.head.appendChild(style);
    }
})

function detectLanguage(text) {
    if (!text || text.trim() === '') return 'en';

    const languageCounts = {
        en: 0, zh: 0, ko: 0, ru: 0, ar: 0, ja: 0
    };

    const languageCodes = {
        en: 'en', zh: 'zh-CN', ko: 'ko', ru: 'ru', ar: 'ar', ja: 'ja'
    }

    var meetSpace = true; // start with space

    for (let i = 0; i < text.length; i++) {
        const char = text[i];
        const code = char.charCodeAt(0);

        // JA
        if ((code >= 0x3040 && code <= 0x309F) ||
            (code >= 0x30A0 && code <= 0x30FF)) {
            languageCounts.ja++;
            continue;
        }

        // KO
        if ((code >= 0xAC00 && code <= 0xD7AF) ||
            (code >= 0x1100 && code <= 0x11FF) ||
            (code >= 0x3130 && code <= 0x318F)) {
            languageCounts.ko++;
            continue;
        }

        // RU
        if (code >= 0x0400 && code <= 0x04FF) {
            languageCounts.ru++;
            continue;
        }

        // AR
        if (code >= 0x0600 && code <= 0x06FF) {
            languageCounts.ar++;
            continue;
        }

        // ZH
        if (code >= 0x4E00 && code <= 0x9FFF) {
            languageCounts.zh++;
            continue;
        }

        // EN, count keywords
        if (code < 128) {
            if (meetSpace) {
                languageCounts.en++;
            }
        }

        if (char === ' ') {
            meetSpace = true;
        } else if (meetSpace) {
            meetSpace = false;
        }
    }

    let maxCount = 0;
    let detectedLang = "en";

    for (const lang in languageCounts) {
        if (languageCounts[lang] > maxCount) {
            maxCount = languageCounts[lang];
            detectedLang = languageCodes[lang];
        }
    }

    return {
        language: detectedLang,
        counts: languageCounts
    };
}

function escapeString(str) {
    return str.replace(/[\n<>]/g, (char) => {
        switch (char) {
            case '\n': return '&#10;';
            case '<': return '&lt;';
            case '>': return '&gt;';
            default: return char;
        }
    });
}

function show_big_photo(photo_url) {
    // remove big photo container if exist
    const big_photo_container_old = document.querySelector(".big-photo-container");
    big_photo_container_old?.remove();
    console.log("show big photo", photo_url);

    // create big photo container
    var big_photo_container = document.createElement("div");
    big_photo_container.classList.add("big-photo-container");

    // create image
    var big_photo_img = document.createElement("img");
    big_photo_img.src = photo_url;
    big_photo_img.classList.add("big-photo-image");

    // rotation state
    var rotation = 0;
    var scale = 1;

    // drag state
    var isDragging = false;
    var dragStart = { x: 0, y: 0 };
    var imgPosition = { x: 0, y: 0 };

    // create control bar
    var control_bar = document.createElement("div");
    control_bar.classList.add("big-photo-control-bar");

    // close button
    var close_btn = document.createElement("div");
    close_btn.classList.add("control-btn", "close-btn");
    close_btn.innerHTML = `
        <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
        </svg>
    `;
    close_btn.title = "Close (Esc)";

    // rotate left button
    var rotate_left_btn = document.createElement("div");
    rotate_left_btn.classList.add("control-btn", "rotate-left-btn");
    rotate_left_btn.innerHTML = `
        <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M7.11 8.53L5.7 7.11C4.8 8.27 4.24 9.61 4.07 11h2.02c.14-.87.49-1.72 1.02-2.47zM6.09 13H4.07c.17 1.39.72 2.73 1.62 3.89l1.41-1.42c-.52-.75-.87-1.59-1.01-2.47zm1.01 5.32c1.16.9 2.51 1.44 3.9 1.61V17.9c-.87-.15-1.71-.49-2.46-1.03L7.1 18.32zM13 4.07V1L8.45 5.55 13 10V6.09c2.84.48 5 2.94 5 5.91s-2.16 5.43-5 5.91v2.02c3.95-.49 7-3.85 7-7.93s-3.05-7.44-7-7.93z"/>
        </svg>
    `;
    rotate_left_btn.title = "Rotate left (←)";

    // rotate right button
    var rotate_right_btn = document.createElement("div");
    rotate_right_btn.classList.add("control-btn", "rotate-right-btn");
    rotate_right_btn.innerHTML = `
        <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M15.55 5.55L11 1v3.07C7.06 4.56 4 7.92 4 12s3.05 7.44 7 7.93v-2.02c-2.84-.48-5-2.94-5-5.91s2.16-5.43 5-5.91V10l4.55-4.45zM19.93 11c-.17-1.39-.72-2.73-1.62-3.89l-1.42 1.42c.54.75.88 1.6 1.02 2.47h2.02zM13 17.9v2.02c1.39-.17 2.74-.71 3.9-1.61l-1.44-1.44c-.75.54-1.59.89-2.46 1.03zm3.89-2.42l1.42 1.41c.9-1.16 1.45-2.5 1.62-3.89h-2.02c-.14.87-.48 1.72-1.02 2.48z"/>
        </svg>
    `;
    rotate_right_btn.title = "Rotate right (→)";

    // zoom out button
    var zoom_out_btn = document.createElement("div");
    zoom_out_btn.classList.add("control-btn", "zoom-out-btn");
    zoom_out_btn.innerHTML = `
        <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M15.5 14h-.79l-.28-.27C15.41 12.59 16 11.11 16 9.5 16 5.91 13.09 3 9.5 3S3 5.91 3 9.5 5.91 16 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14zM7 9h5v1H7z"/>
        </svg>
    `;
    zoom_out_btn.title = "Zoom out (-)";

    // zoom in button
    var zoom_in_btn = document.createElement("div");
    zoom_in_btn.classList.add("control-btn", "zoom-in-btn");
    zoom_in_btn.innerHTML = `
        <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M15.5 14h-.79l-.28-.27C15.41 12.59 16 11.11 16 9.5 16 5.91 13.09 3 9.5 3S3 5.91 3 9.5 5.91 16 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14zm.5-7H9v2H7v1h2v2h1v-2h2V9h-2z"/>
        </svg>
    `;
    zoom_in_btn.title = "Zoom in (+)";

    // reset button
    var reset_btn = document.createElement("div");
    reset_btn.classList.add("control-btn", "reset-btn");
    reset_btn.innerHTML = `
        <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 5V1L7 6l5 5V7c3.31 0 6 2.69 6 6s-2.69 6-6 6-6-2.69-6-6H4c0 4.42 3.58 8 8 8s8-3.58 8-8-3.58-8-8-8z"/>
        </svg>
    `;
    reset_btn.title = "Reset (R)";

    // assemble control bar
    control_bar.appendChild(close_btn);
    control_bar.appendChild(rotate_left_btn);
    control_bar.appendChild(rotate_right_btn);
    control_bar.appendChild(zoom_out_btn);
    control_bar.appendChild(zoom_in_btn);
    control_bar.appendChild(reset_btn);

    // assemble container
    big_photo_container.appendChild(big_photo_img);
    big_photo_container.appendChild(control_bar);
    document.body.appendChild(big_photo_container);

    // update image transform
    function updateTransform() {
        big_photo_img.style.transform = `translate(${imgPosition.x}px, ${imgPosition.y}px) rotate(${rotation}deg) scale(${scale})`;
    }

    // close handler
    function closePhoto() {
        big_photo_container.remove();
    }

    // rotate left
    rotate_left_btn.addEventListener("click", function (event) {
        event.stopPropagation();
        rotation -= 90;
        updateTransform();
    });

    // rotate right
    rotate_right_btn.addEventListener("click", function (event) {
        event.stopPropagation();
        rotation += 90;
        updateTransform();
    });

    // zoom in
    zoom_in_btn.addEventListener("click", function (event) {
        event.stopPropagation();
        scale = Math.min(scale + 0.2, 3);
        updateTransform();
    });

    // zoom out
    zoom_out_btn.addEventListener("click", function (event) {
        event.stopPropagation();
        scale = Math.max(scale - 0.2, 0.5);
        updateTransform();
    });

    // reset
    reset_btn.addEventListener("click", function (event) {
        event.stopPropagation();
        imgPosition.x = 0;
        imgPosition.y = 0;
        rotation = 0;
        scale = 1;
        updateTransform();
    });

    // close button
    close_btn.addEventListener("click", function (event) {
        event.stopPropagation();
        closePhoto();
    });

    // drag image
    big_photo_img.addEventListener("mousedown", function (event) {
        event.preventDefault();
        isDragging = true;
        dragStart.x = event.clientX - imgPosition.x;
        dragStart.y = event.clientY - imgPosition.y;
        big_photo_img.style.cursor = "grabbing";
    });

    document.addEventListener("mousemove", function (event) {
        if (!isDragging) return;
        imgPosition.x = event.clientX - dragStart.x;
        imgPosition.y = event.clientY - dragStart.y;
        updateTransform();
    });

    document.addEventListener("mouseup", function () {
        if (isDragging) {
            isDragging = false;
            big_photo_img.style.cursor = "grab";
        }
    });

    // set initial cursor
    big_photo_img.style.cursor = "grab";

    // click background to close
    big_photo_container.addEventListener("click", function (event) {
        if (event.target === big_photo_container) {
            closePhoto();
        }
    });

    // keyboard shortcuts
    function handleKeyDown(event) {
        switch (event.key) {
            case 'Escape':
                closePhoto();
                break;
            case 'ArrowLeft':
                rotation -= 90;
                updateTransform();
                break;
            case 'ArrowRight':
                rotation += 90;
                updateTransform();
                break;
            case '+':
            case '=':
                scale = Math.min(scale + 0.2, 3);
                updateTransform();
                break;
            case '-':
            case '_':
                scale = Math.max(scale - 0.2, 0.5);
                updateTransform();
                break;
            case 'r':
            case 'R':
                // reset position, rotation and scale
                imgPosition.x = 0;
                imgPosition.y = 0;
                rotation = 0;
                scale = 1;
                updateTransform();
                break;
        }
    }

    document.addEventListener('keydown', handleKeyDown);

    // cleanup on remove
    const observer = new MutationObserver(function (mutations) {
        mutations.forEach(function (mutation) {
            mutation.removedNodes.forEach(function (node) {
                if (node === big_photo_container) {
                    document.removeEventListener('keydown', handleKeyDown);
                    observer.disconnect();
                }
            });
        });
    });
    observer.observe(document.body, { childList: true });

    big_photo_container.addEventListener("contextmenu", function (event) {
        event.stopPropagation();
    });
}

let scroll_struct = {
    direction: 'down',
    offsetTop: 0,
    distance: 0,
    last: {
        direction: 'down',
        offsetTop: 0
    }
}
function AddShrinkTopBarListener() {
    const top_bar = document.querySelector('#top-bar');
    if (!top_bar) return;
    scroll_struct.offsetTop = window.scrollY;
    scroll_struct.last.offsetTop = window.scrollY;

    addEventListener('scroll', function () {
        const currentScrollTop = window.scrollY;

        if (currentScrollTop > scroll_struct.last.offsetTop) { // down
            if (scroll_struct.direction === 'down') {
                scroll_struct.distance += (currentScrollTop - scroll_struct.last.offsetTop);
            } else {
                scroll_struct.direction = 'down';
                scroll_struct.distance = currentScrollTop - scroll_struct.last.offsetTop;
            }
        } else if (currentScrollTop < scroll_struct.last.offsetTop) { // up
            if (scroll_struct.direction === 'up') {
                scroll_struct.distance += (scroll_struct.last.offsetTop - currentScrollTop);
            } else {
                scroll_struct.direction = 'up';
                scroll_struct.distance = scroll_struct.last.offsetTop - currentScrollTop;
            }
        }

        scroll_struct.offsetTop = currentScrollTop;
        scroll_struct.last.direction = scroll_struct.direction;
        scroll_struct.last.offsetTop = currentScrollTop;

        if (scroll_struct.direction === 'down' && scroll_struct.distance > 100) {
            top_bar.classList.add('shrinked');
        } else if (scroll_struct.direction === 'up' || scroll_struct.distance < 100) {
            top_bar.classList.remove('shrinked');
        }
    });
}

// 初始化所有功能
if (typeof window !== 'undefined') {
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', function () {
            InitSidePanels();
            RenderOutline();
            RenderHighlight();
            SwitchToRemoveEditDate();
            AddImgEventListener();
            AddShrinkTopBarListener();
            InitToolbox();
        });
    } else {
        // DOM 已加载完成，直接初始化
        InitSidePanels();
        RenderOutline();
        RenderHighlight();
        SwitchToRemoveEditDate();
        AddImgEventListener();
        AddShrinkTopBarListener();
        InitToolbox();
    }
}

function InitSidePanels() {
    // 检测当前页面
    const isEditPage = location.pathname === '/editarticle.html' || location.pathname === '/addarticle.html';

    // 初始化左侧工具箱 - 编辑页面不显示
    const toolboxContainer = document.querySelector('.toolbox-container');
    if (toolboxContainer) {
        if (isEditPage) {
            // 编辑页面完全隐藏工具箱
            toolboxContainer.style.display = 'none';
            console.log('Toolbox hidden on edit page');
        } else {
            // 文章页面初始化工具箱
            toolboxContainer.style.transform = 'translateY(-50%) translateX(-150%)';
            console.log('Toolbox initialized');
        }
    }

    // 初始化右侧大纲
    const outlineContainer = document.querySelector('.outline-container');
    if (outlineContainer) {
        outlineContainer.style.transform = 'translateX(150%)';
        console.log('Outline initialized');
    }
}

// ==================== 笔记功能 ====================

// 笔记数据存储
let notesData = [];
const NOTES_STORAGE_KEY = 'article_notes_' + getQueryVariable("article_id");

// 用于存储高亮元素的映射
let highlightElementsMap = new Map();

// 初始化笔记功能
function InitNotesFeature() {
    // 检查笔记面板是否存在（仅在文章页面存在）
    const notesPanel = document.querySelector('.notes-panel');
    if (!notesPanel) {
        console.log('Notes panel not found, skipping notes feature initialization');
        return;
    }

    // 加载保存的笔记
    LoadNotesFromStorage();

    // 绑定控制按钮事件
    const addTextNoteBtn = document.querySelector('#add-text-note-btn');
    const saveNotesBtn = document.querySelector('#save-notes-btn');
    const clearNotesBtn = document.querySelector('#clear-notes-btn');
    const closeNotesBtn = document.querySelector('#close-notes-btn');

    if (addTextNoteBtn) {
        addTextNoteBtn.addEventListener('click', () => ShowAddNoteDialog());
    }

    if (saveNotesBtn) {
        saveNotesBtn.addEventListener('click', SaveNotesToFile);
    }

    if (clearNotesBtn) {
        clearNotesBtn.addEventListener('click', ClearAllNotes);
    }

    if (closeNotesBtn) {
        closeNotesBtn.addEventListener('click', () => {
            const notesPanel = document.querySelector('.notes-panel');
            if (notesPanel) {
                notesPanel.classList.remove('active');
            }
        });
    }

    // 绑定文本选择事件（用于添加引用）
    InitTextSelectionFeature();

    // 渲染笔记列表
    RenderNotesList();
}

// 切换笔记面板显示/隐藏
function ToggleNotesPanel() {
    const notesPanel = document.querySelector('.notes-panel');
    if (notesPanel) {
        notesPanel.classList.toggle('active');
    }
}

// 初始化文本选择功能
let isMouseOverPrompt = false;
let selectionCheckTimeout = null;

function InitTextSelectionFeature() {
    // 仅在文章页面启用（非编辑页面）
    const isEditPage = location.pathname === '/editarticle.html' || location.pathname === '/addarticle.html';
    if (isEditPage) {
        console.log('Text selection feature disabled on edit pages');
        return;
    }

    const articleContent = document.querySelector('.article-content');
    if (!articleContent) return;

    // 监听鼠标松开事件
    articleContent.addEventListener('mouseup', () => {
        const selection = window.getSelection();
        const selectedText = selection.toString().trim();

        if (selectedText.length > 0 && selectedText.length < 500) {
            // 显示引用提示
            ShowQuotePrompt(selectedText, selection);
        }
    });

    // 监听选择变化，取消选择时隐藏提示（延迟检测）
    document.addEventListener('selectionchange', () => {
        // 清除之前的检测
        if (selectionCheckTimeout) {
            clearTimeout(selectionCheckTimeout);
        }

        // 延迟200ms检测，避免误判
        selectionCheckTimeout = setTimeout(() => {
            const selection = window.getSelection();
            const selectedText = selection.toString().trim();

            // 只有在没有选择文本且鼠标不在提示框上时才移除
            if (selectedText.length === 0 && !isMouseOverPrompt) {
                const existingPrompt = document.querySelector('.quote-prompt');
                if (existingPrompt) {
                    existingPrompt.remove();
                    isMouseOverPrompt = false;
                }
                if (quotePromptTimeout) {
                    clearTimeout(quotePromptTimeout);
                    quotePromptTimeout = null;
                }
            }
        }, 200);
    });
}

// 显示引用提示
let quotePromptTimeout = null;
function ShowQuotePrompt(text, selection) {
    // 清除之前的提示
    const existingPrompt = document.querySelector('.quote-prompt');
    if (existingPrompt) {
        existingPrompt.remove();
        isMouseOverPrompt = false;
    }

    if (quotePromptTimeout) clearTimeout(quotePromptTimeout);

    // 创建引用提示
    const prompt = document.createElement('div');
    prompt.className = 'quote-prompt';
    prompt.innerHTML = `
        <svg viewBox="0 0 24 24" fill="currentColor" style="width: 18px; height: 18px; display: block;">
            <path d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34c-.39-.39-1.02-.39-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z"/>
        </svg>
    `;
    prompt.title = 'Add Note';
    prompt.style.cssText = `
        position: absolute;
        background: #2d3748;
        color: #fff;
        padding: 8px;
        border-radius: 4px;
        cursor: pointer;
        z-index: 1000;
        box-shadow: 0 2px 12px rgba(0,0,0,0.3);
        transition: all 0.2s;
        display: flex;
        align-items: center;
        justify-content: center;
    `;

    // 获取选中文本的位置（相对于文档）
    const range = selection.getRangeAt(0);
    const rect = range.getBoundingClientRect();
    const scrollTop = window.pageYOffset || document.documentElement.scrollTop;
    const scrollLeft = window.pageXOffset || document.documentElement.scrollLeft;

    prompt.style.left = `${rect.left + scrollLeft}px`;
    prompt.style.top = `${rect.bottom + scrollTop + 10}px`;

    document.body.appendChild(prompt);

    // 点击添加引用笔记
    prompt.addEventListener('click', (e) => {
        e.stopPropagation();
        isMouseOverPrompt = false;
        ShowAddNoteDialog(text);
        prompt.remove();
        if (quotePromptTimeout) {
            clearTimeout(quotePromptTimeout);
            quotePromptTimeout = null;
        }
        // 延迟清除选择，避免触发selectionchange
        setTimeout(() => {
            window.getSelection().removeAllRanges();
        }, 100);
    });

    // hover效果
    prompt.addEventListener('mouseenter', () => {
        isMouseOverPrompt = true;
        prompt.style.background = '#1a202c';
        prompt.style.transform = 'scale(1.05)';
    });

    prompt.addEventListener('mouseleave', () => {
        isMouseOverPrompt = false;
        prompt.style.background = '#2d3748';
        prompt.style.transform = 'scale(1)';
    });

    // 3秒后自动消失
    quotePromptTimeout = setTimeout(() => {
        if (prompt.parentNode) {
            prompt.remove();
            isMouseOverPrompt = false;
        }
    }, 3000);
}

// 显示添加笔记对话框
function ShowAddNoteDialog(quotedText = '') {
    // 创建遮罩层
    const overlay = document.createElement('div');
    overlay.classList.add('note-input-overlay');

    // 创建对话框
    const dialog = document.createElement('div');
    dialog.classList.add('note-input-dialog');

    let quoteHTML = '';
    if (quotedText) {
        quoteHTML = `<div class="note-quote-preview" style="background: var(--base-gray-100); padding: 8px 12px; margin-bottom: 12px; border-left: 3px solid var(--base-gray-700); border-radius: 4px; font-size: 0.9rem; color: var(--base-gray-700);">
            <div style="font-weight: 600; margin-bottom: 4px;">Quote:</div>
            <div style="font-style: italic;">${escapeHTML(quotedText)}</div>
        </div>`;
    }

    dialog.innerHTML = `
        <div class="note-input-header">${quotedText ? 'Add Quote Note' : 'Add Text Note'}</div>
        ${quoteHTML}
        <textarea class="note-input-textarea" placeholder="Enter note content..."></textarea>
        <div class="note-input-actions">
            <button class="note-input-btn cancel">Cancel</button>
            <button class="note-input-btn confirm">Add</button>
        </div>
    `;

    document.body.appendChild(overlay);
    document.body.appendChild(dialog);

    // 聚焦到文本框
    const textarea = dialog.querySelector('.note-input-textarea');
    textarea.focus();

    // 取消按钮
    dialog.querySelector('.cancel').addEventListener('click', () => {
        overlay.remove();
        dialog.remove();
    });

    // 确认按钮
    dialog.querySelector('.confirm').addEventListener('click', () => {
        const content = textarea.value.trim();
        if (content) {
            AddNote(content, quotedText);
            overlay.remove();
            dialog.remove();
            window.Notify.add("Note added successfully", { type: "success", timeout: 2000 });
        } else {
            window.Notify.add("Note content cannot be empty", { type: "error", timeout: 2000 });
        }
    });

    // 点击遮罩关闭
    overlay.addEventListener('click', () => {
        overlay.remove();
        dialog.remove();
    });
}

// 添加笔记
function AddNote(content, quotedText = '') {
    const note = {
        id: Date.now(),
        content: content,
        quotedText: quotedText,
        timestamp: new Date().toLocaleString('en-US')
    };

    // 如果有引用文本，创建高亮标记
    if (quotedText) {
        note.highlightId = 'highlight-' + note.id;
        HighlightTextInArticle(quotedText, note.highlightId);
    }

    notesData.push(note);
    SaveNotesToStorage();
    RenderNotesList();
}

// 在文章中高亮引用的文本
function HighlightTextInArticle(text, highlightId) {
    const articleContent = document.querySelector('.article-content');
    if (!articleContent) return;

    // 查找并高亮第一个匹配的文本
    const walker = document.createTreeWalker(
        articleContent,
        NodeFilter.SHOW_TEXT,
        null
    );

    let node;
    while (node = walker.nextNode()) {
        const index = node.textContent.indexOf(text);
        if (index !== -1) {
            const range = document.createRange();
            range.setStart(node, index);
            range.setEnd(node, index + text.length);

            const span = document.createElement('span');
            span.className = 'text-highlight';
            span.setAttribute('data-highlight-id', highlightId);
            range.surroundContents(span);

            // 保存引用
            highlightElementsMap.set(highlightId, span);
            break;
        }
    }
}

// 滚动到引用文本位置
function ScrollToQuote(highlightId) {
    const highlightElement = highlightElementsMap.get(highlightId);
    if (highlightElement) {
        highlightElement.scrollIntoView({ behavior: 'smooth', block: 'center' });
        // 添加闪烁效果
        highlightElement.classList.add('flash');
        setTimeout(() => {
            highlightElement.classList.remove('flash');
        }, 1000);
    }
}

// 删除笔记
function DeleteNote(noteId) {
    const note = notesData.find(n => n.id === noteId);

    // 删除高亮标记
    if (note && note.highlightId) {
        const highlightElement = highlightElementsMap.get(note.highlightId);
        if (highlightElement) {
            const parent = highlightElement.parentNode;
            while (highlightElement.firstChild) {
                parent.insertBefore(highlightElement.firstChild, highlightElement);
            }
            parent.removeChild(highlightElement);
            highlightElementsMap.delete(note.highlightId);
        }
    }

    notesData = notesData.filter(note => note.id !== noteId);
    SaveNotesToStorage();
    RenderNotesList();
    window.Notify.add("Note deleted", { type: "success", timeout: 2000 });
}

// 编辑笔记
function EditNote(noteId) {
    const note = notesData.find(n => n.id === noteId);
    if (!note) return;

    // 创建遮罩层
    const overlay = document.createElement('div');
    overlay.classList.add('note-input-overlay');

    // 创建对话框
    const dialog = document.createElement('div');
    dialog.classList.add('note-input-dialog');

    dialog.innerHTML = `
        <div class="note-input-header">Edit Note</div>
        <textarea class="note-input-textarea">${note.content}</textarea>
        <div class="note-input-actions">
            <button class="note-input-btn cancel">Cancel</button>
            <button class="note-input-btn confirm">Save</button>
        </div>
    `;

    document.body.appendChild(overlay);
    document.body.appendChild(dialog);

    // 聚焦到文本框
    const textarea = dialog.querySelector('.note-input-textarea');
    textarea.focus();
    textarea.setSelectionRange(textarea.value.length, textarea.value.length);

    // 取消按钮
    dialog.querySelector('.cancel').addEventListener('click', () => {
        overlay.remove();
        dialog.remove();
    });

    // 确认按钮
    dialog.querySelector('.confirm').addEventListener('click', () => {
        const content = textarea.value.trim();
        if (content) {
            note.content = content;
            note.timestamp = new Date().toLocaleString('en-US') + ' (edited)';
            SaveNotesToStorage();
            RenderNotesList();
            overlay.remove();
            dialog.remove();
            window.Notify.add("Note saved successfully", { type: "success", timeout: 2000 });
        } else {
            window.Notify.add("Note content cannot be empty", { type: "error", timeout: 2000 });
        }
    });

    // 点击遮罩关闭
    overlay.addEventListener('click', () => {
        overlay.remove();
        dialog.remove();
    });
}

// 渲染笔记列表
function RenderNotesList() {
    const notesContent = document.querySelector('#notes-content');
    if (!notesContent) return;

    // 如果没有笔记，显示空状态
    if (notesData.length === 0) {
        notesContent.innerHTML = '<div class="notes-empty">No notes yet. Click the button above to add notes.</div>';
        return;
    }

    // 渲染笔记列表（按时间倒序）
    notesContent.innerHTML = '';
    const sortedNotes = [...notesData].reverse();

    sortedNotes.forEach(note => {
        const noteItem = document.createElement('div');
        noteItem.classList.add('note-item');

        // 构建引用块HTML
        let quoteHTML = '';
        if (note.quotedText) {
            quoteHTML = `<div class="note-quote" data-highlight-id="${note.highlightId || ''}">${escapeHTML(note.quotedText)}</div>`;
        }

        noteItem.innerHTML = `
            <div class="note-item-header">
                <div class="note-item-time">${note.timestamp}</div>
                <div class="note-item-actions">
                    <button class="note-item-btn comment-btn" title="Submit as Comment">
                        <svg viewBox="0 0 24 24" fill="currentColor">
                            <path d="M20 2H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h14l4 4V4c0-1.1-.9-2-2-2zm-2 12H6v-2h12v2zm0-3H6V9h12v2zm0-3H6V6h12v2z"/>
                        </svg>
                    </button>
                    <button class="note-item-btn edit-btn" title="Edit">
                        <svg viewBox="0 0 24 24" fill="currentColor">
                            <path d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34c-.39-.39-1.02-.39-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z"/>
                        </svg>
                    </button>
                    <button class="note-item-btn delete-btn" title="Delete">
                        <svg viewBox="0 0 24 24" fill="currentColor">
                            <path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
                        </svg>
                    </button>
                </div>
            </div>
            ${quoteHTML}
            <div class="note-item-content">${escapeHTML(note.content)}</div>
        `;

        // 如果有引用，点击引用块滚动到原文
        if (note.quotedText && note.highlightId) {
            const quoteBlock = noteItem.querySelector('.note-quote');
            if (quoteBlock) {
                quoteBlock.addEventListener('click', () => {
                    ScrollToQuote(note.highlightId);
                });
            }
        }

        // 提交为评论按钮
        noteItem.querySelector('.comment-btn').addEventListener('click', () => {
            SubmitNoteAsComment(note);
        });

        // 编辑按钮
        noteItem.querySelector('.edit-btn').addEventListener('click', () => {
            EditNote(note.id);
        });

        // 删除按钮
        noteItem.querySelector('.delete-btn').addEventListener('click', () => {
            if (confirm('Are you sure you want to delete this note?')) {
                DeleteNote(note.id);
            }
        });

        notesContent.appendChild(noteItem);
    });
}

// 提交笔记为评论
function SubmitNoteAsComment(note) {
    // 构建评论内容
    let commentContent = note.content;
    if (note.quotedText) {
        commentContent = `> ${note.quotedText}\n\n${note.content}`;
    }

    // 显示评论输入框
    ShowCommentInputBox();

    // 等待评论框加载
    setTimeout(() => {
        const commentTextarea = document.querySelector('#add-comment-text');
        if (commentTextarea) {
            commentTextarea.value = commentContent;
            commentTextarea.focus();
            // 滚动到评论框
            commentTextarea.scrollIntoView({ behavior: 'smooth', block: 'center' });
            window.Notify.add("Note content has been filled into comment box", { type: "success", timeout: 2000 });
        } else {
            window.Notify.add("Comment function is not enabled", { type: "error", timeout: 2000 });
        }
    }, 300);
}

// 保存笔记到 localStorage
function SaveNotesToStorage() {
    try {
        localStorage.setItem(NOTES_STORAGE_KEY, JSON.stringify(notesData));
    } catch (e) {
        console.error('Failed to save notes:', e);
        window.Notify.add("Failed to save notes, storage limit may be exceeded", { type: "error", timeout: 3000 });
    }
}

// 从 localStorage 加载笔记
function LoadNotesFromStorage() {
    try {
        const stored = localStorage.getItem(NOTES_STORAGE_KEY);
        if (stored) {
            notesData = JSON.parse(stored);
            // 恢复高亮
            notesData.forEach(note => {
                if (note.quotedText && note.highlightId) {
                    HighlightTextInArticle(note.quotedText, note.highlightId);
                }
            });
        }
    } catch (e) {
        console.error('Failed to load notes:', e);
        notesData = [];
    }
}

// 保存笔记到本地文件
function SaveNotesToFile() {
    if (notesData.length === 0) {
        window.Notify.add("No notes to save", { type: "info", timeout: 2000 });
        return;
    }

    try {
        const articleTitle = document.querySelector('.article-title')?.textContent || 'Article';
        const articleId = getQueryVariable("article_id") || 'unknown';

        // 构建JSON数据
        const exportData = {
            article_id: articleId,
            article_title: articleTitle,
            export_time: new Date().toLocaleString('en-US'),
            notes_count: notesData.length,
            notes: notesData
        };

        // 创建 Blob
        const blob = new Blob([JSON.stringify(exportData, null, 2)], {
            type: 'application/json;charset=utf-8'
        });
        const blobUrl = URL.createObjectURL(blob);

        // 创建下载链接
        const link = document.createElement('a');
        link.href = blobUrl;
        link.download = `notes-${articleTitle}-${new Date().getTime()}.json`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);

        // 清理
        setTimeout(() => URL.revokeObjectURL(blobUrl), 100);

        window.Notify.add("Notes saved to local file", { type: "success", timeout: 2000 });
    } catch (error) {
        console.error('Failed to save notes:', error);
        window.Notify.add("Save failed: " + error.message, { type: "error", timeout: 3000 });
    }
}

// 清空所有笔记
function ClearAllNotes() {
    if (notesData.length === 0) {
        window.Notify.add("No notes to clear", { type: "info", timeout: 2000 });
        return;
    }

    if (confirm(`Are you sure you want to clear all ${notesData.length} notes? This action cannot be undone.`)) {
        // 清除所有高亮
        notesData.forEach(note => {
            if (note.highlightId) {
                const highlightElement = highlightElementsMap.get(note.highlightId);
                if (highlightElement) {
                    const parent = highlightElement.parentNode;
                    while (highlightElement.firstChild) {
                        parent.insertBefore(highlightElement.firstChild, highlightElement);
                    }
                    parent.removeChild(highlightElement);
                    highlightElementsMap.delete(note.highlightId);
                }
            }
        });

        notesData = [];
        SaveNotesToStorage();
        RenderNotesList();
        window.Notify.add("All notes cleared", { type: "success", timeout: 2000 });
    }
}

// HTML转义函数
function escapeHTML(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML.replace(/\n/g, '<br>');
}