// render variable
var NewCommentInputBoxHTML = `
{{file:new_comment_input_box}}
`
var CF_Site_key = "{{global:cf_site_key}}"
var Goole_reCaptcha_Site_key = "{{global:google_site_key}}"
var comment_check_type = "{{global:comment_check_type}}"
// end of render variable

function AddArticleAPI(title, author, content, contentHTML, callback) {
    const result = GetAccessPathAndToken();
    if (!result) {
        console.log("Access path and token are required.");
        return false;
    }
    const { path, token } = result;
    console.log("Access path: " + path);
    console.log("Access token: " + token);
    const api_dic = window.location.origin + "/" + path;
    const api_add_article = api_dic + "/add_article";
    const data = {
        token: token,
        article: {
            title: title,
            author: author,
            content: content,
            content_html: contentHTML
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
            callback("");
        });
}

function EditArticleAPI(article_id, title, author, content, contentHTML, callback) {
    const result = GetAccessPathAndToken();
    if (!result) {
        console.log("Access path and token are required.");
        return false;
    }
    const { path, token } = result;
    console.log("Access path: " + path);
    console.log("Access token: " + token);
    const api_dic = window.location.origin + "/" + path;
    const api_edit_article = api_dic + "/edit_article";
    const data = {
        token: token,
        article: {
            article_id: article_id,
            title: title,
            author: author,
            content: content,
            content_html: contentHTML
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
    console.log("Access path: " + path);
    console.log("Access token: " + token);
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
            callback("");
        });
}

function AddCommentAPI(article_id, author, email, content, callback) {
    path = "api/v1"
    const api_dic = window.location.origin + "/" + path;
    const api_add_comment = api_dic + "/add_comment";
    const data = {
        verify_token: window.comment_token,
        article_id: article_id,
        author: author,
        email: email,
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
    console.log("Access path: " + path);
    console.log("Access token: " + token);
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
    console.log("Access path: " + path);
    console.log("Access token: " + token);
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
            callback("");
        });
}

function SaveArticle() {
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
    // check if in /addarticle.html
    if (location.pathname === "/addarticle.html") {
        // add article
        AddArticleAPI(editor_title, author_input, markdown_input, rendered_content, function (result) {
            if (result != "") {
                console.log(result);
                alert("Article added successfully!");
                // jump to article page
                article_id = result.article_id;
                console.log(article_id);
                // clear local storage
                localStorage.removeItem("localStoredArticle");
                // jump to article page
                window.location.href = "/articles/" + article_id;
            }
        });
    } else if (location.pathname === "/editarticle.html") {
        article_id = getQueryVariable("article_id");
        // edit article
        EditArticleAPI(article_id, editor_title, author_input, markdown_input, rendered_content, function (result) {
            if (result != "") {
                alert("Article edited successfully!");
                console.log(result);
                // jump to article page
                window.location.href = "/articles/" + article_id;
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
        document.body.appendChild(CommentInputBoxDoc);
        // append turnstile script
        const turnstile_script = document.createElement("script");
        turnstile_script.src = "https://challenges.cloudflare.com/turnstile/v0/api.js?onload=onloadTurnstileCallback";
        document.body.appendChild(turnstile_script);
    } else if (comment_check_type == "google_recaptcha") {
        const CommentBoxPre = document.querySelector(".comment-input-box");
        CommentBoxPre?.remove()
        // append recaptcha script
        const recaptcha_script = document.createElement("script");
        recaptcha_script.src = "https://www.google.com/recaptcha/api.js?render=" + Goole_reCaptcha_Site_key;
        document.body.appendChild(CommentInputBoxDoc);
        CommentInputBoxDoc.appendChild(recaptcha_script);
    } else {
        alert("Comment system has been disabled.")
    }
}

function CancleCommentInputBox() {
    const CommentBoxPre = document.querySelector(".comment-input-box");
    CommentBoxPre?.remove()
}

function OnAddCommentButtonClick() {
    const article_id = getQueryVariable("article_id");
    const author_input = document.querySelector('#add-comment-author').value;
    const email_address = document.querySelector('#add-comment-emailaddress').value;
    const content_input = document.querySelector('#add-comment-text').value;
    if (!isAvailableEmailAddress(email_address)) {
        alert("Invalid email address.");
        return;
    }
    // check if google recaptcha
    if (comment_check_type == "google_recaptcha") {

        grecaptcha.ready(function () {
            grecaptcha.execute(Goole_reCaptcha_Site_key, { action: 'submit' }).then(function (token) {
                // Add your logic to submit to your backend server here.
                if (!article_id || !author_input || !content_input || !token) {
                    console.log("Article id, author, content and token are required.");
                    alert("Please fill in all required fields.");
                    return;
                }
                window.comment_token = token;
                AddCommentAPI(article_id, author_input, email_address, content_input, function (result) {
                    if (result != "") {
                        console.log(result);
                        alert("Comment added successfully!");
                        // remove comment input box
                        CancleCommentInputBox();
                        // reload page
                        location.reload();
                    } else {
                        alert("Failed to add comment.");
                    }
                });
            });
        });
        return;
    } else if (comment_check_type == "cloudflare_turnstile") {
        if (!article_id || !author_input || !content_input || !window.comment_token) {
            console.log("Article id, author, content and token are required.");
            alert("Please fill in all required fields.");
            return;
        }
        AddCommentAPI(article_id, author_input, email_address, content_input, function (result) {
            if (result != "") {
                console.log(result);
                alert("Comment added successfully!");
                // remove comment input box
                CancleCommentInputBox();
                // reload page
                location.reload();
            } else {
                alert("Failed to add comment.");
            }
        });
    }
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

function RenderOutline() {
    const outlineTitle = document.querySelector('.outline-title');
    const outlineList = document.querySelector('.outline-list');
    const articleTitle = document.querySelector('.article-title');
    const articleDom = document.querySelector('.article-content');
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
    // add mousemove event listener to active outline container
    document.body.addEventListener('mousemove', MouseMoveHandler);
}

function MouseMoveHandler(event) {
    const mouseX = event.clientX;
    const mouseY = event.clientY;
    const windowWidth = window.innerWidth;
    // const windowHeight = window.innerHeight;
    const outlineContainer = document.querySelector('.outline-container');
    const OutlineContainerX = outlineContainer.offsetLeft;
    const OutlineContainerY = outlineContainer.offsetTop;
    const OutlineContainerWidth = outlineContainer.offsetWidth;
    const OutlineContainerHeight = outlineContainer.offsetHeight;
    // console.log(mouseX, mouseY, windowWidth, windowHeight)
    if (mouseX > windowWidth - 50) {
        // console.log('right');
        outlineContainer.style.transform = `translateX(0%)`;
    } else if (mouseX > OutlineContainerX && mouseX < OutlineContainerX + OutlineContainerWidth && mouseY > OutlineContainerY && mouseY < OutlineContainerY + OutlineContainerHeight) {
        // console.log(mouseX, mouseY,  OutlineContainerX + OutlineContainerWidth, OutlineContainerY+ OutlineContainerHeight)

    } else {
        outlineContainer.style.transform = `translateX(150%)`;
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
            heading.style.scrollMarginTop = '50px';
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
            highlight_style.href = "/css/light.highlight.css";
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

window.addEventListener('load', function () {
    RenderOutline();
});

window.addEventListener('DOMContentLoaded', function () {
    RenderHighlight();
});

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