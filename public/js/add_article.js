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
        case "upload":
            // 触发文件选择对话框，并显示有效期选择
            showFileExpiryDialog();
            return; // 不需要重新渲染，因为是异步操作
    }
    // rerender markdown
    renderMarkdown();
    const articleDom = document.querySelector('.article-content');
    const outlineList = document.querySelector('.outline-list');
    generateOutline(articleDom, outlineList)
}

// 显示文件有效期选择对话框
function showFileExpiryDialog() {
    // 检查是否已存在对话框
    const existingDialog = document.querySelector('.file-expiry-dialog');
    if (existingDialog) {
        existingDialog.remove();
    }

    // 创建对话框
    const dialog = document.createElement('div');
    dialog.className = 'file-expiry-dialog';

    // 根据当前主题添加类名
    const currentTheme = GetTheme ? GetTheme() : localStorage.getItem('theme') || 'light';
    if (currentTheme === 'dark') {
        dialog.classList.add('dark-theme');
    }

    dialog.innerHTML = `
        <div class="file-expiry-overlay"></div>
        <div class="file-expiry-content">
            <h3>Upload File</h3>
            <div class="expiry-option-group">
                <label>Expiry Time:</label>
                <select id="file-expiry-select" class="expiry-select">
                    <option value="never">Never (Permanent)</option>
                    <option value="1">1 Day</option>
                    <option value="7" selected>7 Days</option>
                    <option value="30">30 Days</option>
                    <option value="90">90 Days</option>
                    <option value="365">1 Year</option>
                </select>
            </div>
            <div class="expiry-buttons">
                <button class="btn-cancel" id="file-expiry-cancel">Cancel</button>
                <button class="btn-confirm" id="file-expiry-confirm">Select File</button>
            </div>
        </div>
    `;

    // 添加样式
    if (!document.getElementById('file-expiry-dialog-style')) {
        const style = document.createElement('style');
        style.id = 'file-expiry-dialog-style';
        style.textContent = `
            .file-expiry-dialog {
                position: fixed;
                top: 0;
                left: 0;
                width: 100%;
                height: 100%;
                z-index: 9999;
            }
            .file-expiry-overlay {
                position: absolute;
                top: 0;
                left: 0;
                width: 100%;
                height: 100%;
                background: rgba(0, 0, 0, 0.5);
                backdrop-filter: blur(2px);
            }
            .file-expiry-content {
                position: absolute;
                top: 50%;
                left: 50%;
                transform: translate(-50%, -50%);
                background: #ffffff;
                padding: 24px;
                border-radius: 12px;
                box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
                min-width: 320px;
                max-width: 400px;
                transition: all 0.3s ease;
            }
            .file-expiry-content h3 {
                margin: 0 0 20px 0;
                font-size: 20px;
                font-weight: 600;
                color: #2d3748;
            }
            .expiry-option-group {
                margin-bottom: 20px;
            }
            .expiry-option-group label {
                display: block;
                margin-bottom: 8px;
                font-weight: 500;
                font-size: 14px;
                color: #4a5568;
            }
            .expiry-select {
                width: 100%;
                padding: 10px 12px;
                border: 1px solid #cbd5e0;
                border-radius: 6px;
                font-size: 14px;
                background: #ffffff;
                color: #2d3748;
                cursor: pointer;
                transition: all 0.2s;
            }
            .expiry-select:hover {
                border-color: #4299e1;
            }
            .expiry-select:focus {
                outline: none;
                border-color: #4299e1;
                box-shadow: 0 0 0 3px rgba(66, 153, 225, 0.1);
            }
            .expiry-buttons {
                display: flex;
                gap: 12px;
                justify-content: flex-end;
            }
            .expiry-buttons button {
                padding: 10px 20px;
                border: none;
                border-radius: 6px;
                font-size: 14px;
                font-weight: 500;
                cursor: pointer;
                transition: all 0.2s;
            }
            .btn-cancel {
                background: #e2e8f0;
                color: #4a5568;
            }
            .btn-cancel:hover {
                background: #cbd5e0;
            }
            .btn-cancel:active {
                transform: scale(0.98);
            }
            .btn-confirm {
                background: #4299e1;
                color: white;
            }
            .btn-confirm:hover {
                background: #3182ce;
            }
            .btn-confirm:active {
                transform: scale(0.98);
            }
            
            /* 暗色模式 */
            .file-expiry-dialog.dark-theme .file-expiry-overlay {
                background: rgba(0, 0, 0, 0.7);
            }
            .file-expiry-dialog.dark-theme .file-expiry-content {
                background: #1a202c;
                box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
            }
            .file-expiry-dialog.dark-theme .file-expiry-content h3 {
                color: #f7fafc;
            }
            .file-expiry-dialog.dark-theme .expiry-option-group label {
                color: #e2e8f0;
            }
            .file-expiry-dialog.dark-theme .expiry-select {
                background: #2d3748;
                border-color: #4a5568;
                color: #e2e8f0;
            }
            .file-expiry-dialog.dark-theme .expiry-select:hover {
                border-color: #4299e1;
                background: #374151;
            }
            .file-expiry-dialog.dark-theme .expiry-select:focus {
                border-color: #4299e1;
                box-shadow: 0 0 0 3px rgba(66, 153, 225, 0.2);
            }
            .file-expiry-dialog.dark-theme .expiry-select option {
                background: #2d3748;
                color: #e2e8f0;
            }
            .file-expiry-dialog.dark-theme .btn-cancel {
                background: #374151;
                color: #e2e8f0;
            }
            .file-expiry-dialog.dark-theme .btn-cancel:hover {
                background: #4b5563;
            }
            .file-expiry-dialog.dark-theme .btn-confirm {
                background: #3b82f6;
                color: white;
            }
            .file-expiry-dialog.dark-theme .btn-confirm:hover {
                background: #2563eb;
            }
        `;
        document.head.appendChild(style);
    }

    document.body.appendChild(dialog);

    // 绑定事件
    const overlay = dialog.querySelector('.file-expiry-overlay');
    const cancelBtn = dialog.querySelector('#file-expiry-cancel');
    const confirmBtn = dialog.querySelector('#file-expiry-confirm');

    const closeDialog = () => {
        dialog.remove();
    };

    overlay.addEventListener('click', closeDialog);
    cancelBtn.addEventListener('click', closeDialog);

    confirmBtn.addEventListener('click', () => {
        const expiryDays = document.getElementById('file-expiry-select').value;
        closeDialog();

        // 触发文件选择
        const fileInput = document.getElementById('file-input');
        if (fileInput) {
            fileInput.onchange = (event) => handleFileUpload(event, expiryDays);
            fileInput.click();
        }
    });
}

// 处理文件上传
async function handleFileUpload(event, expiryDays = '7') {
    const file = event.target.files[0];
    if (!file) {
        return;
    }

    // 检查文件大小（32MB限制）
    if (file.size > 32 * 1024 * 1024) {
        window.Notify.add("File size exceeds 32MB", { type: "error" });
        return;
    }

    // 获取登录凭证
    const result = GetAccessPathAndToken(true);
    if (!result) {
        window.Notify.add("Please login first before uploading files", { type: "error" });
        window.open("/login.html?blank=true&redirect=" + location.href, "_blank");
        return;
    }

    const { path, token } = result;
    const uploadAPI = window.location.origin + "/" + path + "/upload_file";

    // 创建 FormData
    const formData = new FormData();
    formData.append('token', token);
    formData.append('file', file);
    formData.append('expiry_days', expiryDays); // 添加有效期参数

    // 显示上传进度提示
    const expiryText = expiryDays === 'never' ? 'permanent' : `${expiryDays} days`;
    window.Notify.add(`Uploading file (expiry: ${expiryText})...`, { type: "info" });

    try {
        const response = await fetch(uploadAPI, {
            method: 'POST',
            body: formData
        });

        if (!response.ok) {
            throw new Error(`Upload failed: ${response.status}`);
        }

        const data = await response.json();

        // 上传成功，插入文件链接
        const fileURL = data.url;
        const fileName = data.original_name;

        // 生成完整的URL（包含域名）
        const fullURL = window.location.origin + fileURL;

        // 根据文件类型插入不同的 Markdown 格式
        let insertedText = '';
        const fileExt = fileName.split('.').pop().toLowerCase();
        const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp', 'svg'];

        if (imageExts.includes(fileExt)) {
            // 图片文件，使用 ![alt](url) 格式
            insertedText = `![${fileName}](${fileURL})`;
        } else {
            // 其他文件，使用 [text](url) 格式
            insertedText = `[${fileName}](${fileURL})`;
        }

        // 插入到编辑器
        const editor_content = document.querySelector('.markdown-textarea');
        if (editor_content) {
            const selectionStart = editor_content.selectionStart;
            const selectionEnd = editor_content.selectionEnd;

            editor_content.value =
                editor_content.value.substring(0, selectionStart) +
                insertedText +
                editor_content.value.substring(selectionEnd);

            // 设置光标位置到插入内容之后
            const newCursorPos = selectionStart + insertedText.length;
            editor_content.selectionStart = newCursorPos;
            editor_content.selectionEnd = newCursorPos;

            // 触发焦点
            editor_content.focus();

            // 重新渲染
            renderMarkdown();
            const articleDom = document.querySelector('.article-content');
            const outlineList = document.querySelector('.outline-list');
            generateOutline(articleDom, outlineList);
        }

        // 复制完整URL到剪贴板
        try {
            await navigator.clipboard.writeText(fullURL);
            window.Notify.add("File uploaded successfully! Full URL has been copied to clipboard", { type: "success" });
        } catch (err) {
            // 如果复制失败，使用备用方法
            copyText(fullURL);
            window.Notify.add("File uploaded successfully! URL has been inserted into the editor", { type: "success" });
        }

    } catch (error) {
        console.error('File upload failed:', error);
        window.Notify.add("File upload failed: " + error.message, { type: "error" });
    } finally {
        // 清空文件输入，允许重复上传同一文件
        event.target.value = '';
    }
}

AddMarkdownEditorListener();
document.addEventListener('DOMContentLoaded', RenderLocalData);