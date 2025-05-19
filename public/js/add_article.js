function AddMarkdownEditorListener() {
    const editor_title = document.querySelector('.title-input');
    const author_input = document.querySelector('.author-input');
    const editor_content = document.querySelector('.markdown-textarea');
    editor_title.addEventListener('input', renderMarkdown);
    author_input.addEventListener('input', renderMarkdown);
    editor_content.addEventListener('input', renderMarkdown);
}

function renderMarkdown() {
    const editor_title = document.querySelector('.title-input');
    const author_input = document.querySelector('.author-input');
    const editor_content = document.querySelector('.markdown-textarea');
    const rendered_title = document.querySelector('.article-title');
    const rendered_author = document.querySelector('.article-author');
    const rendered_content = document.querySelector('.article-content');
    const rendered_date = document.querySelector('.article-date');
    const title_value = editor_title.value;
    const author_value = author_input.value;
    const content_value = editor_content.value;
    const date_value = new Date().toLocaleString();
    rendered_title.textContent = title_value;
    rendered_author.textContent = author_value;
    rendered_content.innerHTML = marked.parse(content_value);
    rendered_date.textContent = date_value;
    // save to localstroage
    if (location.pathname === "/addarticle.html") {
        let localStoredArticle = {
            "title": title_value,
            "author": author_value,
            "content": content_value
        };
        localStorage.setItem('localStoredArticle', JSON.stringify(localStoredArticle));
    }

}

function RenderLocalData() {
    const editor_title = document.querySelector('.title-input');
    const author_input = document.querySelector('.author-input');
    const editor_content = document.querySelector('.markdown-textarea');
    if (location.pathname === "/addarticle.html") {
        let localStoredArticle = JSON.parse(localStorage.getItem('localStoredArticle'));
        if (localStoredArticle) {
            storageTitle = localStoredArticle.title;
            storageAuthor = localStoredArticle.author;
            storageContent = localStoredArticle.content;
            if (storageTitle || storageAuthor || storageContent) {
                editor_title.value = storageTitle;
                author_input.value = storageAuthor;
                editor_content.value = storageContent;
                renderMarkdown();
            }
        }
    } else if (location.pathname === "/editarticle.html") {
        article_id = getQueryVariable("article_id");
        console.log(article_id);
        GetArticleAPI(article_id, function(data) {
            if (data || data.article_type === "markdown"){ 
                storageTitle = data.title;
                storageAuthor = data.author;
                storageContent = data.content;
                if (storageTitle || storageAuthor || storageContent) {
                    editor_title.value = storageTitle;
                    author_input.value = storageAuthor;
                    editor_content.value = storageContent;
                    renderMarkdown();
                }
            }
        })
    }
    
}

AddMarkdownEditorListener();
document.addEventListener('DOMContentLoaded', RenderLocalData);