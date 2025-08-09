function AddMarkdownEditorListener() {
    const editor_title = document.querySelector('.title-input');
    const author_input = document.querySelector('.author-input');
    const editor_content = document.querySelector('.markdown-textarea');
    editor_title.addEventListener('input', renderMarkdown);
    author_input.addEventListener('input', renderMarkdown);
    editor_content.addEventListener('input', function () {
        renderMarkdown();
        RenderHighlight();
        const articleDom = document.querySelector('.article-content');
        const outlineList = document.querySelector('.outline-list');
        generateOutline(articleDom, outlineList)
    });
    // set editor content key event listener
    editor_content.addEventListener('keydown', function (event) {
        if (event.ctrlKey) {
            switch (event.key) {
                case "s": // ctrl + s
                    event.preventDefault();
                    if (location.pathname === "/addarticle.html") {
                        window.Notify.add("Article saved to local.", { type: "success" })
                    } else if (location.pathname === "/editarticle.html") {
                        SaveArticle(true);
                    }
                    break;
            }
        }
        if (event.key === "Tab") { // turn tab into 4 spaces
            event.preventDefault();
            selectionStart = editor_content.selectionStart;
            selectionEnd = editor_content.selectionEnd;
            const tab = "    ";
            editor_content.value =
                editor_content.value.substring(0, selectionStart) +
                tab +
                editor_content.value.substring(selectionEnd);
            editor_content.selectionStart = selectionStart + tab.length;
            editor_content.selectionEnd = selectionStart + tab.length;
            // rerender markdown
            renderMarkdown();
            const articleDom = document.querySelector('.article-content');
            const outlineList = document.querySelector('.outline-list');
            generateOutline(articleDom, outlineList)
        }
    });
    // set editor content input event listener
    editor_content.addEventListener('input', function () {
        // check if input text is the last character of the content
        if (editor_content.value.length === editor_content.selectionEnd) {
            // stroll to the end of the content
            preview_content = document.querySelector('.preview-content');
            preview_content.scrollTop = preview_content.scrollHeight;
        }
    });
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
                const articleDom = document.querySelector('.article-content');
                const outlineList = document.querySelector('.outline-list');
                generateOutline(articleDom, outlineList)
            }
        }
    } else if (location.pathname === "/editarticle.html") {
        article_id = getQueryVariable("article_id");
        console.log(article_id);
        GetArticleAPI(article_id, function (data) {
            if (data) {
                storageTitle = data.title;
                storageAuthor = data.author;
                storageContent = data.content;
                if (storageTitle || storageAuthor || storageContent) {
                    editor_title.value = storageTitle;
                    author_input.value = storageAuthor;
                    editor_content.value = storageContent;
                    renderMarkdown();
                    const articleDom = document.querySelector('.article-content');
                    const outlineList = document.querySelector('.outline-list');
                    generateOutline(articleDom, outlineList)
                }
            } else {
                window.Notify.add("Article not found or not logged in.", { type: "error" })
            }
        })
    }

}

function onToolbarButtonClick(buttontype) {
    const editor_content = document.querySelector('.markdown-textarea');
    if (!editor_content) return;
    const selectionStart = editor_content.selectionStart;
    const selectionEnd = editor_content.selectionEnd;
    // if (selectionStart === selectionEnd) {
    //     return;
    // }
    var selectedText = editor_content.value.substring(selectionStart, selectionEnd);

    function insertText(text) {
        editor_content.focus();
        const beforeLength = selectedText.length;
        const newLength = text.length;

        // set selection text
        editor_content.value =
            editor_content.value.substring(0, selectionStart) +
            text +
            editor_content.value.substring(selectionEnd);
        // set cursor position
        editor_content.selectionStart = selectionStart;
        editor_content.selectionEnd = selectionStart + newLength;
    }

    switch (buttontype) {
        case "bold":
            // check if selection is already bold
            if (selectedText === "") {
                selectedText = "bold text"
            }
            if (selectedText.startsWith('**') && selectedText.endsWith('**')) {
                // remove bold
                insertText(selectedText.substring(2, selectedText.length - 2));
            } else {
                // add bold
                insertText('**' + selectedText + '**');
            }
            break;
        case "italic":
            if (selectedText === "") {
                selectedText = "italic text"
            }
            // check if selection is already italic
            if (selectedText.startsWith('*') && selectedText.endsWith('*')) {
                // remove italic
                insertText(selectedText.substring(1, selectedText.length - 1));
            } else {
                // add italic
                insertText('*' + selectedText + '*');
            }
            break;
        case "link":
            if (selectedText === "") {
                selectedText = "link text"
            }
            regex = /(\[.*?\]\(.*?\))/ // []() pattern for links
            // check if selection is already a link
            if (regex.test(selectedText)) {
                // remove link
                const match = selectedText.match(/\[(.*?)\]\((.*?)\)/);
                insertText(match[1]);
            } else {
                insertText('[' + selectedText + '](https://)');
            }
            break;
        case "image":
            if (selectedText === "") {
                selectedText = "image text"
            }
            regex = /(!\[(.*?)\]\(.*?\))/ // ![alt text]() pattern for images
            // check if selection is already an image
            if (regex.test(selectedText)) {
                // remove image
                const match = selectedText.match(/!\[(.*?)\]\((.*?)\)/);
                insertText(match[1]);
            } else {
                insertText('![' + selectedText + '](https://)');
            }
            break;
        case "title":
            if (selectedText === "") {
                selectedText = "title text"
            }
            // check if selection is already a title
            if (selectedText.startsWith('# ')) {
                // remove title
                insertText(selectedText.substring(2));
            } else {
                // add title
                insertText('# ' + selectedText);
            }
            break;
        case "code":
            if (selectedText === "") {
                selectedText = "code text"
            }
            // check if selection is already code
            if (selectedText.startsWith('```') && selectedText.endsWith('```')) {
                // remove code
                insertText(selectedText.substring(3, selectedText.length - 3));
            } else {
                // add code
                insertText('```\n' + selectedText + '\n```');
            }
            break;
        case "list":
            if (selectedText === "") {
                selectedText = "list text"
            }
            // check if selection is already a list
            if (selectedText.startsWith('- ') || selectedText.startsWith('* ')) {
                // remove list
                insertText(selectedText.substring(2));
            } else {
                // add list
                insertText('- ' + selectedText);
            }
            break;
        case "quote":
            if (selectedText === "") {
                selectedText = "quote text"
            }
            // check if selection is already a quote
            if (selectedText.startsWith('> ')) {
                // remove quote
                insertText(selectedText.substring(2));
            } else {
                // add quote
                insertText('> ' + selectedText);
            }
            break;
        case "table":
            if (selectedText === "") {
                selectedText = "table text"
            }
            table = `| Column 1 | Column 2 |\n| --- | --- |\n| Row 1, Column 1 | Row 1, Column 2 |\n| Row 2, Column 1 | Row 2, Column 2 |\n`
            insertText(table);
            break;
    }
    // rerender markdown
    renderMarkdown();
    const articleDom = document.querySelector('.article-content');
    const outlineList = document.querySelector('.outline-list');
    generateOutline(articleDom, outlineList)
}

AddMarkdownEditorListener();
document.addEventListener('DOMContentLoaded', RenderLocalData);