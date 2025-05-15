function EnterEditMode() {
    const result = GetAccessPathAndToken();
    if (!result) {
        console.log("Access path and token are required.");
        return;
    }
    const { path, token } = result;
    console.log("Access path: " + path);
    console.log("Access token: " + token);

    if (window._editMode) {
        // save changes
        console.log("Save changes");
        const api_dic = window.location.origin+"/" + path;
        const api_edit_order = api_dic + "/edit_order";
        const data = {
            token: token,
            changes: [
            ]
        }
        const cards = document.querySelectorAll('.card-container');
        cards.forEach((card) => {
            const cardID = card.getAttribute('card-id');
            const order = parseInt(card.style.order, 10);
            const newChange = {
                cardID: cardID,
                order: order
            }
            data.changes.push(newChange);
        });
        console.log(JSON.stringify(data));
        fetch(api_edit_order, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
                },
            body: JSON.stringify(data)
            },
        )
       .then(response => {
            if (response.ok) {
                console.log("Save changes successfully");
                window._editMode = false;
                // remove edit-mode-border class
                cards.forEach((card) => {
                    card.classList.remove('edit-mode-border');
                });
            } else {
                console.log("Save changes failed");
            }
        })
       .catch(error => {
            console.log("Save changes failed: " + error);
       });
    }

    window._editMode = true;
    // select all cards, record their original index and add edit-mode-border class
    var originalIndex = new Map(); // cardID -> original order
    const cards = document.querySelectorAll('.card-container');
    cards.forEach((card) => {
        cardID = card.getAttribute('card-id');
        order = card.style.order;
        if (order != -1) { // not the Top up card
            // console.log("Card ID: " + cardID);
            originalIndex.set(cardID, card.style.order);
            card.classList.add('edit-mode-border');
        }
        
    });

    let dragging_card = null;
    // add drag and drop event listener to each card
    cards.forEach((card) => {
        if (card.style.order == -1) { // Top up card
            return;
        }
        card.addEventListener('dragstart', (event) => {
            dragstart_handler(event);
        });
        card.addEventListener('dragover', (event) => {
            dragover_handler(event);
        });
        card.addEventListener('drop', (event) => {
            drop_handler(event);
        });
        card.draggable = true;
        card.classList.add('draggable');
    });
    
    let draggable_cards = document.querySelectorAll('.draggable');

    function dragstart_handler(event) {
        console.log("Drag start");
        dragging_card = event.target;
        dragging_card.classList.add('dragging');
    }

    function dragover_handler(event) {
        event.preventDefault();
        console.log("Drag over");
        
    }

    function drop_handler(event) {
        event.preventDefault();
        console.log("Drop");
        updateViewCardOrder(event);
        dragging_card.classList.remove('dragging');
        // const cardID = event.target.getAttribute('card-id');
    }

    function updateViewCardOrder(event) {
        // get cursor position
        const cursorPositionX = event.clientX + window.scrollX; // event.pageX
        const cursorPositionY = event.clientY + window.scrollY; // event.pageY
        let localted_card = dragging_card;
        // check where cursor located
        for (let i = 0; i < draggable_cards.length; i++) {
            const card = draggable_cards[i];
            const card_positionX = card.offsetLeft;
            const card_positionY = card.offsetTop;
            const card_width = card.offsetWidth;
            const card_height = card.offsetHeight;
            if (card.style.order != -1 && card != dragging_card && (cursorPositionX >= card_positionX && cursorPositionX <= card_positionX + card_width && cursorPositionY >= card_positionY && cursorPositionY <= card_positionY + card_height)) {
                // cursor located in this card
                localted_card = card;
                break;
            }
        }
        if (localted_card && (localted_card != dragging_card)) { // cursor located in a card
            // change dragged card order to the position of cursor
            const beforeOrder = parseInt(dragging_card.style.order, 10);
            const afterOrder = parseInt(localted_card.style.order, 10);
            // calc offset order
            // offsetOrder used to adjust the order of cards after dragging
            const offsetOrder = afterOrder - beforeOrder;
            draggable_cards.forEach((card) => {
                const currentOrder = parseInt(card.style.order, 10);
                
                if (offsetOrder > 0) { // 向下拖动
                    // 位于原位置和目标位置之间的卡片，order 减 1
                    if (card !== dragging_card && currentOrder > beforeOrder && currentOrder <= afterOrder) {
                        card.style.order = currentOrder - 1;
                    }
                } else if (offsetOrder < 0) { // 向上拖动
                    // 位于目标位置和原位置之间的卡片，order 加 1
                    if (card !== dragging_card && currentOrder >= afterOrder && currentOrder < beforeOrder) {
                        card.style.order = currentOrder + 1;
                    }
                }
            });
        
            // update order
            dragging_card.style.order = afterOrder;
        }
    }
}

function GetAccessPathAndToken() {
    let path = localStorage.getItem("access_path");
    let token = localStorage.getItem("access_token");
    if (path === null || token === null) {
        const newPath = prompt("Enter the access path (e.g. accessBackend):");
        if (newPath === null) return null;
        const newToken = prompt("Enter the access token:");
        if (newToken === null) return null;
        
        localStorage.setItem("access_path", newPath);
        localStorage.setItem("access_token", newToken);
        path = newPath;
        token = newToken;
    }
    return { path, token };
}

function AddEditButtonListener() {
    const editButtons = document.querySelectorAll(".edit-button");
    editButtons.forEach(button => {
        button.addEventListener("click", EnterEditMode);
        });
}

AddEditButtonListener();