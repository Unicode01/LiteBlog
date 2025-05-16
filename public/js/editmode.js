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
                    card.classList.remove('draggable');
                    card.draggable = false;
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

function DeleteCard(cardID) {
    const result = GetAccessPathAndToken();
    if (!result) {
        console.log("Access path and token are required.");
        return false;
    }
    const { path, token } = result;
    console.log("Access path: " + path);
    console.log("Access token: " + token);
    const api_dic = window.location.origin+"/" + path;
    const api_delete_card = api_dic + "/delete_card";
    const data = {
        token: token,
        cardID: cardID
    }
    fetch(api_delete_card, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
            },
        body: JSON.stringify(data)
        },
    )
    .then(response => {
        if (response.ok) {
            console.log("Delete card successfully");
            const card = document.querySelector(`[card-id="${cardID}"]`);
            card.remove();
            return true;
        } else {
            console.log("Delete card failed");
            return false;
        }
    })
   .catch(error => {
        console.log("Delete card failed: " + error);
        return false;
   });
    return true;
}

function AddCard() {
    if (window._addingCard) {
        return;
    } else {
        window._addingCard = true;
    }
    // this func is called to generate the card json and add it to the backend
    const add_card_input_box_html = `
<div class="card-input-box">
    <div class="input-group">
        <label for="add-card-title">Title</label>
        <input type="text" id="add-card-title" required>
    </div>

    <div class="input-group">
        <label for="add-card-description">Description</label>
        <textarea id="add-card-description" rows="3" required></textarea>
    </div>

    <div class="input-group">
        <label for="add-card-link">Link</label>
        <textarea id="add-card-link" rows="2" required></textarea>
    </div>

    <div class="input-group">
        <label for="add-card-template">Template</label>
        <input type="text" id="add-card-template" value="card_template_classical" required>
    </div>

    <div class="input-group">
        <label for="add-card-tags">Tags</label>
        <input type="text" id="add-card-tags" value="mytag1 mytag2" required>
    </div>

    <div class="custom-fields">
        <button type="button" onclick="OncreateCustomFieldButtonClick()">
            <span>+</span>
            <span>Add Custom Field</span>
        </button>
        <div id="custom-fields-container"></div>
    </div>

    <div class="button-group">
        <button type="button" class="add-card-button" onclick="OnAddCardButtonClick()">Add Card</button>
        <button type="button" class="cancel-button" onclick="CancleInputBox()">Cancel</button>
    </div>
</div>
    `;
    domP = new DOMParser();
    const add_card_box_doc = domP.parseFromString(add_card_input_box_html, "text/html").body.firstChild;
    document.body.appendChild(add_card_box_doc);
}

function OnAddCardButtonClick() {
    function GetCardJson() {
        const customFields = {};
        
        // 获取所有自定义字段
        document.querySelectorAll('.custom-field-group').forEach(group => {
            const name = group.querySelector('.field-name').value;
            const value = group.querySelector('.field-value').value;
            if (name && value) {
                customFields[name] = value;
            }
        });
        json =  {
            card_title: document.getElementById('add-card-title').value.toString(),
            card_description: document.getElementById('add-card-description').value.toString(),
            card_link: document.getElementById('add-card-link').value.toString(),
            template: document.getElementById('add-card-template').value.toString(),
            tags: document.getElementById('add-card-tags').value.toString(),
            order: document.querySelectorAll('.card-container').length.toString()
        };
        for (const key in customFields) {
            json[key] = customFields[key].toString();
        }
        return json;
    }
    AddCardAPI(GetCardJson())
    CancleInputBox();
}

function OncreateCustomFieldButtonClick() {
    const container = document.getElementById('custom-fields-container');
    
    const fieldGroup = document.createElement('div');
    fieldGroup.className = 'custom-field-group';
    
    // 字段名称输入框
    const nameInput = document.createElement('input');
    nameInput.type = "text";
    nameInput.placeholder = "Field Name (e.g.: link)";
    nameInput.className = "field-name";
    
    // 字段值输入框
    const valueInput = document.createElement('input');
    valueInput.type = "text";
    valueInput.placeholder = "field value";
    valueInput.className = "field-value";
    
    // 删除按钮
    const removeBtn = document.createElement('button');
    removeBtn.type = "button";
    removeBtn.textContent = "×";
    removeBtn.onclick = () => fieldGroup.remove();
    
    fieldGroup.appendChild(nameInput);
    fieldGroup.appendChild(valueInput);
    fieldGroup.appendChild(removeBtn);
    container.appendChild(fieldGroup);
}

function CancleInputBox() {
    const add_card_box = document.querySelector(".card-input-box");
    add_card_box.remove();
    window._addingCard = false;
}

function AddCardAPI(cardJson) {
    const result = GetAccessPathAndToken();
    if (!result) {
        console.log("Access path and token are required.");
        return false;
    }
    const { path, token } = result;
    console.log("Access path: " + path);
    console.log("Access token: " + token);
    const api_dic = window.location.origin+"/" + path;
    const api_add_card = api_dic + "/add_card";
    const data = {
        token: token,
        card: cardJson
    }
    console.log(JSON.stringify(data));
    fetch(api_add_card, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
            },
        body: JSON.stringify(data)
        },
    )
    .then(response => {
        if (response.ok) {
            console.log("Add card successfully");
            return true;
        } else {
            console.log("Add card failed");
            return false;
        }
    })
   .catch(error => {
        console.log("Add card failed: " + error);
        return false;
   });
    return true;
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