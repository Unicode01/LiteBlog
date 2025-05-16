// render variables from server
var Card_max_width = {{global:card_max_width}}; //w
var Card_max_height = {{global:card_max_height}};
var Card_min_width = {{global:card_min_width}}; //w
var Card_min_height = {{global:card_min_height}};
var Context_menu_html = `{{rendered:context_menu_html}}`
// end render

function init() {
    ResizeCard();
    AddEventListener();
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
    const prevmenu =     document.getElementById('context-menu');
    if (prevmenu) {
        prevmenu.remove();
    }
    
    domParser = new DOMParser();
    var context_menu_doc = domParser.parseFromString(context_menu_norender, "text/html").body.firstChild;
    // console.log(context_menu_doc);
    // -add basic options
    // ---add copy option
    // ---check selection then add copy option
    if (window.getSelection().toString() != '') {
        var menu_copy = document.createElement('div');
        menu_copy.classList.add('menu-item');
        menu_copy.innerHTML = '<a class="link" href="#">Copy</a>';
        menu_copy.firstChild.addEventListener('click', function(event) {
            event.preventDefault();
            selection = window.getSelection();
            copyText(selection.toString());
        });
        context_menu_doc.appendChild(menu_copy);
        // add item line
        var menu_line = document.createElement('div');
        menu_line.classList.add('menu-item-line');
        context_menu_doc.appendChild(menu_line);
    }
    // -add Reload option
    var menu_Reload = document.createElement('div');
    menu_Reload.classList.add('menu-item');
    menu_Reload.innerHTML = '<a class="link" href="#">Reload</a>';
    menu_Reload.firstChild.addEventListener('click', function(event) {
        event.preventDefault();
        location.reload();
    });
    context_menu_doc.appendChild(menu_Reload);

    // add menu line
    var menu_line = document.createElement('div');
    menu_line.classList.add('menu-item-line');
    context_menu_doc.appendChild(menu_line);

    // check if cursor is in card
    const cards = document.querySelectorAll('.card-container');
    cards.forEach(card => {
        const cursorX = event.clientX;
        const cursorY = event.clientY;
        const cardX = card.getBoundingClientRect().left;
        const cardY = card.getBoundingClientRect().top;
        const cardWidth = card.getBoundingClientRect().width;
        const cardHeight = card.getBoundingClientRect().height;
        if (cursorX > cardX && cursorX < cardX + cardWidth && cursorY > cardY && cursorY < cardY + cardHeight) {
            // in card, add get link option
            // add copy link option
            var menu_getlink = document.createElement('div');
            menu_getlink.classList.add('menu-item');
            menu_getlink.innerHTML = '<a class="link" href="#">Copy Link</a>';
            menu_getlink.firstChild.addEventListener('click', function(event) {
                event.preventDefault();
                var link = card.querySelector('.link').href;
                console.log(link);
                // copy link to clipboard
                copyText(link);
            });
            context_menu_doc.appendChild(menu_getlink);

            // add item line
            var menu_line = document.createElement('div');
            menu_line.classList.add('menu-item-line');
            context_menu_doc.appendChild(menu_line);

            // check if in edit mode
            if (window._editMode) {
                // add delete option
                var menu_delete = document.createElement('div');
                menu_delete.classList.add('menu-item');
                menu_delete.innerHTML = '<a class="link" href="#">Delete Card</a>';
                menu_delete.firstChild.addEventListener('click', function(event) {
                    event.preventDefault();
                    if (DeleteCard(card.getAttribute("card-id"))) {
                        console.log("card deleted");
                        // reload window
                        // location.reload();
                    } else {
                        console.log("failed to delete card");
                    }
                });
                context_menu_doc.appendChild(menu_delete);

                // add item line
                var menu_line = document.createElement('div');
                menu_line.classList.add('menu-item-line');
                context_menu_doc.appendChild(menu_line);
            }
            
        }
    });

    // check if edit mode
    if (window._editMode) {
        // add addcard option
        var menu_addcard = document.createElement('div');
        menu_addcard.classList.add('menu-item');
        menu_addcard.innerHTML = '<a class="link" href="#">Add Card</a>';
        menu_addcard.firstChild.addEventListener('click', function(event) {
            event.preventDefault();
            if (AddCard()) {
                console.log("card added");
            };
        });
        context_menu_doc.appendChild(menu_addcard);
        // add item line
        var menu_line = document.createElement('div');
        menu_line.classList.add('menu-item-line');
        context_menu_doc.appendChild(menu_line);
    }
    
    // add editmode option
    var menu_editmode = document.createElement('div');
    menu_editmode.classList.add('menu-item');
    if (window._editMode) {
        menu_editmode.innerHTML = '<a class="link" href="#">Exit Edit Mode</a>';
    } else {
        menu_editmode.innerHTML = '<a class="link" href="#">Edit Mode</a>';
    }
    menu_editmode.firstChild.addEventListener('click', function(event) {
        event.preventDefault();
        var editmode = document.getElementById('edit-button');
        if (editmode) {
            editmode.click();
        }
    });
    context_menu_doc.appendChild(menu_editmode);

    // set position
    var menu_x = event.clientX;
    var menu_y = event.clientY;
    context_menu_doc.style.left = menu_x + 'px';
    context_menu_doc.style.top = menu_y + 'px';
    
    // append to body
    document.body.appendChild(context_menu_doc);
   
}

function AddEventListener() {
    // add resize event listener to window
    window.addEventListener('resize', function() {
        ResizeCard();
    });
    // add contextmenu event listener to body
    document.body.addEventListener('contextmenu', function(event) {
        event.preventDefault();
        OnContextMenu(event);
    });
    // add click event listener to body to hide context menu
    document.addEventListener('click', function() {
        const menu =     document.getElementById('context-menu');
        if (menu) {
            menu.remove();
        }
    });
}

// tool functions
async function copyText(text) {
    try {
      await navigator.clipboard.writeText(text);
      console.log("link copied to clipboard with clipboard api:" + text);
    } catch (err) {
      // 现代 API 失败时回退到旧方法
      const success = document.execCommand("copy");
      if (success) {
        console.log("link copied to clipboard:" + text);
      } else {
        console.log("failed to copy link:" + text);
      }
    }
}


init();