function AddArticleAPI(title,author,article_type,content,callback) {
    const result = GetAccessPathAndToken();
    if (!result) {
        console.log("Access path and token are required.");
        return false;
    }
    const { path, token } = result;
    console.log("Access path: " + path);
    console.log("Access token: " + token);
    const api_dic = window.location.origin+"/" + path;
    const api_add_article = api_dic + "/add_article";
    const data = {
        token: token,
        article: {
            title: title,
            author: author,
            article_type: article_type,
            content: content
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

function EditArticleAPI(article_id,title,author,article_type,content,callback) {
    const result = GetAccessPathAndToken();
    if (!result) {
        console.log("Access path and token are required.");
        return false;
    }
    const { path, token } = result;
    console.log("Access path: " + path);
    console.log("Access token: " + token);
    const api_dic = window.location.origin+"/" + path;
    const api_edit_article = api_dic + "/edit_article";
    const data = {
        token: token,
        article: {
            article_id: article_id,
            title: title,
            author: author,
            article_type: article_type,
            content: content
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

function GetArticleAPI(article_id,callback) {
    const result = GetAccessPathAndToken();
    if (!result) {
        console.log("Access path and token are required.");
        return false;
    }
    const { path, token } = result;
    console.log("Access path: " + path);
    console.log("Access token: " + token);
    const api_dic = window.location.origin+"/" + path;
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

function DeleteArticleAPI(article_id,callback) {
    const result = GetAccessPathAndToken();
    if (!result) {
        console.log("Access path and token are required.");
        return false;
    }
    const { path, token } = result;
    console.log("Access path: " + path);
    console.log("Access token: " + token);
    const api_dic = window.location.origin+"/" + path;
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

function SaveArticle(type) {
    if (type == "html") {
        const editor_title = document.querySelector('.title-input').value;
        const author_input = document.querySelector('.author-input').value;
        const rendered_content = document.querySelector('.article-content').innerHTML;
        // check if in /addarticle.html
        if (location.pathname==="/addarticle.html") {
            // add article
            AddArticleAPI(editor_title,author_input,"markdown",rendered_content,function(result) {
                if (result!="") {
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
        } else if (location.pathname==="/editarticle.html") {
            article_id = getQueryVariable("article_id");
            // edit article
            EditArticleAPI(article_id,editor_title,author_input,"markdown",rendered_content,function(result) {
                if (result!="") {
                    alert("Article edited successfully!");
                    console.log(result);
                    // jump to article page
                    window.location.href = "/articles/" + article_id;
                }
            });
        }
    } else if (type == "markdown") {
        const editor_title = document.querySelector('.title-input').value;
        const author_input = document.querySelector('.author-input').value;
        const rendered_content = document.querySelector('.markdown-textarea').value;
        // check if in /addarticle.html
        if (location.pathname==="/addarticle.html") {
            // add article
            AddArticleAPI(editor_title,author_input,"markdown",rendered_content,function(result) {
                if (result!="") {
                    console.log(result);
                    alert("Article added successfully!");
                    // jump to article page
                    article_id = result.article_id;
                    console.log(article_id);
                    // clear local storage
                    localStorage.removeItem("localStoredArticle");
                    // jump to article page
                    window.location.href = "/articles/" + article_id;
                } else {
                    alert("Failed to add article.");
                }
            });
        } else if (location.pathname==="/editarticle.html") {
            article_id = getQueryVariable("article_id");
            // edit article
            EditArticleAPI(article_id,editor_title,author_input,"markdown",rendered_content,function(result) {
                if (result!="") {
                    alert("Article edited successfully!");
                    console.log(result);
                    // jump to article page
                    window.location.href = "/articles/" + article_id;
                }
            });
        }
    }
}

function getQueryVariable(variable)
{
    var query = window.location.search.substring(1);
    var vars = query.split("&");
    for (var i=0;i<vars.length;i++) {
        var pair = vars[i].split("=");
        if(pair[0] == variable){return pair[1];}
    }
    return(false);
}