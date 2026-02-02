/**
 * 主题预加载脚本
 * 在页面加载前设置主题，避免闪烁
 */
(function () {
    function loadStyleString(css) {
        var style = document.createElement("style");
        style.type = "text/css";
        style.id = "preloaded-theme";
        try {
            style.appendChild(document.createTextNode(css));
        } catch (ex) {
            style.textContent = css;
        }
        var head = document.getElementsByTagName("head")[0];
        head.appendChild(style);
    }

    const theme = localStorage.getItem('theme');
    if (theme === 'dark') {
        const css = localStorage.getItem('dark-theme-css');
        if (css) {
            loadStyleString(css);
        }
    }
})();

