if (!window._card_template_todo_list_loaded) {
    window._card_template_todo_list_loaded = true;
    window.addEventListener('DOMContentLoaded', function () {
        card_todo_list_init();
    });
}

function card_todo_list_init() {
    console.log('card_todo_list_init');
    // select all card-todo-list elements
    var card_todo_list_elements = document.querySelectorAll('.card-container-todo-list');
    card_todo_list_elements.forEach(function (card_todo_list_element) {
        // get card-todo-list info
        const main_container = card_todo_list_element.querySelector('.main-container');
        const content_container = main_container.querySelector(".card-content")
        var card_todo_list_info = get_card_todo_list_info(card_todo_list_element);
        console.log(card_todo_list_info);
        card_todo_list_info.todo_list.forEach(function (todo_item) {
            var item = document.createElement('div');
            item.classList.add('todo-item');
            item.innerHTML = `
                <div class="todo-item-date">${todo_item.date}</div>
                <div class="todo-item-action">${todo_item.action}</div>
            `;
            content_container.appendChild(item);
        });
    });
}

function get_card_todo_list_info(carddom) {
    var info = {}
    const info_container = carddom.querySelector(".todo-list-info-container");
    info.todo_list = parse_todo_list_info(info_container.getAttribute("data-list"))
    info_container.remove();
    return info;
}

function parse_todo_list_info(todo_list_info) {
    if (!todo_list_info) return [];

    const segments = todo_list_info.split('|').filter(segment => segment.trim() !== '');
    const regex = /\[(.*?)\]\((.*?)\)/;

    return segments.map(segment => {
        const match = segment.match(regex);
        if (match) {
            return {
                date: match[1].trim(),
                action: match[2].trim()
            };
        }
        return null;
    }).filter(item => item !== null);

}

function getAllTodoList() {

}