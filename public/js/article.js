// render variable
var NewCommentInputBoxHTML = `
{{file:new_comment_input_box}}
`
var CF_Site_key = "{{global:cf_site_key}}"
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

function AddCommentAPI(article_id, author, content, callback) {
    path = "api/v1"
    const api_dic = window.location.origin + "/" + path;
    const api_add_comment = api_dic + "/add_comment";
    const data = {
        verify_token: window.comment_token,
        article_id: article_id,
        author: author,
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
    const rendered_content = document.querySelector('.article-content').innerHTML;
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
    if (comment_check_type === "cloudflare_turnstile") {
        const CommentBoxPre = document.querySelector(".comment-input-box");
        CommentBoxPre?.remove()
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
    }
}

function CancleCommentInputBox() {
    const CommentBoxPre = document.querySelector(".comment-input-box");
    CommentBoxPre?.remove()
}

function OnAddCommentButtonClick() {
    const article_id = getQueryVariable("article_id");
    const author_input = document.querySelector('#add-comment-author').value;
    const content_input = document.querySelector('#add-comment-text').value;
    if (!article_id || !author_input || !content_input || !window.comment_token) {
        console.log("Article id, author, content and token are required.");
        alert("Please fill in all required fields.");
        return;
    }
    AddCommentAPI(article_id, author_input, content_input, function (result) {
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