var dragging_card = null;

function EnterEditMode() {
    const result = GetAccessPathAndToken();
    if (!result) {
        console.log("Access path and token are required.");
        return;
    }
    const { path, token } = result;
    // console.log("Access path: " + path);
    // console.log("Access token: " + token);

    if (window._editMode) {
        // save changes
        console.log("Save changes");
        const api_dic = window.location.origin + "/" + path;
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
                        card.removeAttribute("draggable");
                        // remove drag and drop event listener
                        card.removeEventListener('dragstart', dragstart_handler);
                        card.removeEventListener('dragover', dragover_handler);
                        card.removeEventListener('drop', drop_handler);
                    });
                    // check to remove add card input box
                    const card_input_box = document.querySelector('.card-input-box');
                    card_input_box?.remove();
                } else {
                    console.log("Save changes failed");
                }
            })
            .catch(error => {
                console.log("Save changes failed: " + error);
            });
        return;
    }

    window._editMode = true;
    // select all cards, record their original index and add edit-mode-border class
    var originalIndex = new Map(); // cardID -> original order
    const cards = document.querySelectorAll('.card-container');
    cards.forEach((card) => {
        const cardID = card.getAttribute('card-id');
        order = card.style.order;
        if (order != -1) { // not the Top up card
            // console.log("Card ID: " + cardID);
            originalIndex.set(cardID, card.style.order);
            card.classList.add('edit-mode-border');
        }

    });


    // add drag and drop event listener to each card
    cards.forEach((card) => {
        if (card.style.order == -1) { // Top up card
            return;
        }
        card.addEventListener('dragstart', dragstart_handler);
        card.addEventListener('dragover', dragover_handler);
        card.addEventListener('drop', drop_handler);
        card.draggable = true;
        card.classList.add('draggable');
    });
}

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
    const draggable_cards = document.querySelectorAll('.draggable');
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

function DeleteCard(cardID) {
    const result = GetAccessPathAndToken();
    if (!result) {
        console.log("Access path and token are required.");
        return false;
    }
    const { path, token } = result;
    // console.log("Access path: " + path);
    // console.log("Access token: " + token);
    const api_dic = window.location.origin + "/" + path;
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
        {{file:add_card_input_box}}
    `;
    domP = new DOMParser();
    const add_card_box_doc = domP.parseFromString(add_card_input_box_html, "text/html").body.firstChild;
    document.body.appendChild(add_card_box_doc);
}

function GetCardJsonAPI(cardID, callback) {
    const result = GetAccessPathAndToken();
    if (!result) {
        console.log("Access path and token are required.");
        return false;
    }
    const { path, token } = result;
    // console.log("Access path: " + path);
    // console.log("Access token: " + token);
    const api_dic = window.location.origin + "/" + path;
    const api_get_card = api_dic + "/get_card";
    const data = {
        token: token,
        cardID: cardID
    }
    fetch(api_get_card, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
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

function EditCardAPI(cardJson, callback) {
    const result = GetAccessPathAndToken();
    if (!result) {
        console.log("Access path and token are required.");
        return false;
    }
    const { path, token } = result;
    // console.log("Access path: " + path);
    // console.log("Access token: " + token);
    const api_dic = window.location.origin + "/" + path;
    const api_edit_card = api_dic + "/edit_card";
    const data = {
        token: token,
        card: cardJson
    }
    console.log(JSON.stringify(data));
    fetch(api_edit_card, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
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

function GetCustomSettingsAPI(callback) {
    const result = GetAccessPathAndToken();
    if (!result) {
        console.log("Access path and token are required.");
        return false;
    }
    const { path, token } = result;
    // console.log("Access path: " + path);
    // console.log("Access token: " + token);
    const api_dic = window.location.origin + "/" + path;
    const api_get_custom_settings = api_dic + "/get_custom_settings";
    const data = {
        token: token
    }
    fetch(api_get_custom_settings, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
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

function EditCustomSettingsAPI(customSettings, callback) {
    const result = GetAccessPathAndToken();
    if (!result) {
        console.log("Access path and token are required.");
        return false;
    }
    const { path, token } = result;
    // console.log("Access path: " + path);
    // console.log("Access token: " + token);
    const api_dic = window.location.origin + "/" + path;
    const api_edit_custom_settings = api_dic + "/edit_custom_settings";
    const data = {
        token: token,
        custom_settings: customSettings
    }
    console.log(JSON.stringify(data));
    fetch(api_edit_custom_settings, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
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

function EditCustomSettings() {
    GetCustomSettingsAPI(function (data) {
        if (data === "") {
            console.log("Get custom settings failed");
            return;
        } else {
            if (window._edittingCustomSettings) {
                return;
            } else {
                window._edittingCustomSettings = true;
            }
            // this func is called to generate the card json and add it to the backend
            const edit_custom_settings_html = `
            {{file:edit_custom_settings_box}}
            `;
            domP = new DOMParser();
            const edit_custom_settings_doc = domP.parseFromString(edit_custom_settings_html, "text/html").body.firstChild;
            edit_custom_settings_doc.querySelector("#edit-settings-style").value = data.custom_style;
            edit_custom_settings_doc.querySelector("#edit-settings-script").value = data.custom_script;
            for (const key in data.global_settings) {
                if (key != "custom_style" && key != "custom_script") {
                    const customFieldGroup = document.createElement('div');
                    customFieldGroup.className = 'custom-field-group';

                    // 字段名称输入框
                    const nameInput = document.createElement('input');
                    nameInput.type = "text";
                    nameInput.placeholder = "Field Name (e.g.: link)";
                    nameInput.className = "field-name";
                    nameInput.value = key

                    // 字段值输入框
                    const valueInput = document.createElement('input');
                    valueInput.type = "text";
                    valueInput.placeholder = "field value";
                    valueInput.className = "field-value";
                    valueInput.value = data.global_settings[key];

                    // 删除按钮
                    const removeBtn = document.createElement('button');
                    removeBtn.type = "button";
                    removeBtn.textContent = "×";
                    removeBtn.onclick = () => customFieldGroup.remove();

                    customFieldGroup.appendChild(nameInput);
                    customFieldGroup.appendChild(valueInput);
                    customFieldGroup.appendChild(removeBtn);
                    edit_custom_settings_doc.querySelector("#custom-fields-container").appendChild(customFieldGroup);
                }
            }
            document.body.appendChild(edit_custom_settings_doc);
        }
    });
}

function EditCard(cardId) {
    GetCardJsonAPI(cardId, (cardJson) => {
        if (cardJson === "") {
            console.log("Get card json failed");
            return;
        } else {
            if (window._edittingCard) {
                return;
            } else {
                window._edittingCard = true;
                window._edittingCardID = cardId;
            }
            // this func is called to generate the card json and add it to the backend
            const add_card_input_box_html = `
            {{file:edit_card_input_box}}
            `;
            domP = new DOMParser();
            const add_card_box_doc = domP.parseFromString(add_card_input_box_html, "text/html").body.firstChild;
            add_card_box_doc.querySelector("#add-card-title").value = cardJson.card_title;
            add_card_box_doc.querySelector("#add-card-description").value = cardJson.card_description;
            add_card_box_doc.querySelector("#add-card-link").value = cardJson.card_link;
            add_card_box_doc.querySelector("#add-card-template").value = cardJson.template;
            add_card_box_doc.querySelector("#add-card-tags").value = cardJson.tags;
            for (const key in cardJson) {
                if (key != "card_title" && key != "card_description" && key != "card_link" && key != "template" && key != "tags" && key != "id") {
                    const customFieldGroup = document.createElement('div');
                    customFieldGroup.className = 'custom-field-group';

                    // 字段名称输入框
                    const nameInput = document.createElement('input');
                    nameInput.type = "text";
                    nameInput.placeholder = "Field Name (e.g.: link)";
                    nameInput.className = "field-name";
                    nameInput.value = key.replace("custom_", "");

                    // 字段值输入框
                    const valueInput = document.createElement('input');
                    valueInput.type = "text";
                    valueInput.placeholder = "field value";
                    valueInput.className = "field-value";
                    valueInput.value = cardJson[key];

                    // 删除按钮
                    const removeBtn = document.createElement('button');
                    removeBtn.type = "button";
                    removeBtn.textContent = "×";
                    removeBtn.onclick = () => customFieldGroup.remove();

                    customFieldGroup.appendChild(nameInput);
                    customFieldGroup.appendChild(valueInput);
                    customFieldGroup.appendChild(removeBtn);
                    add_card_box_doc.querySelector("#custom-fields-container").appendChild(customFieldGroup);
                }
            }
            document.body.appendChild(add_card_box_doc);
        }
    });
}

function OnAddCardButtonClick(editMode = false) {
    if (editMode) {
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
            json = {
                card_title: document.getElementById('add-card-title').value.toString(),
                card_description: document.getElementById('add-card-description').value.toString(),
                card_link: document.getElementById('add-card-link').value.toString(),
                template: document.getElementById('add-card-template').value.toString(),
                tags: document.getElementById('add-card-tags').value.toString(),
                order: document.querySelectorAll('.card-container').length.toString(),
                id: window._edittingCardID.toString()
            };
            for (const key in customFields) {
                json[key] = customFields[key].toString();
            }
            return json;
        }
        EditCardAPI(GetCardJson(), function (data) {
            console.log(data);
            CancleInputBox();
        })
    } else {
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
            json = {
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
        AddCardAPI(GetCardJson(), function (data) {
            console.log(data);
            CancleInputBox();
        })
    }
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

function OnEditSettingsButtonClick() {
    function GetGlobalSettings() {
        const customFields = {};

        // 获取所有自定义字段
        document.querySelectorAll('.custom-field-group').forEach(group => {
            const name = group.querySelector('.field-name').value;
            const value = group.querySelector('.field-value').value;
            if (name && value) {
                customFields[name] = value;
            }
        });
        json = {
            custom_style: document.getElementById('edit-settings-style').value.toString(),
            custom_script: document.getElementById('edit-settings-script').value.toString(),
            global_settings: {}
        };
        for (const key in customFields) {
            json.global_settings[key] = customFields[key].toString();
        }
        return json;
    }
    EditCustomSettingsAPI(GetGlobalSettings(), function (data) {
        console.log(data);
        CancleInputBox();
    });

}

function onImportButtonClick() {
    // require user to enter the codes of base64 encoded json data
    const input = prompt("Enter the base64 encoded json data:");
    if (input === null) return;
    const jsonData = JSON.parse(decodeURIComponent(atob(input)));
    console.log(jsonData);
    const customFields = {};
    const card_title = document.getElementById('add-card-title')
    const card_description = document.getElementById('add-card-description')
    const card_link = document.getElementById('add-card-link')
    const template = document.getElementById('add-card-template')
    const tags = document.getElementById('add-card-tags')
    for (const key in jsonData) {
        switch (key) {
            case "card_title":
                card_title.value = jsonData[key];
                break;
            case "card_description":
                card_description.value = jsonData[key];
                break;
            case "card_link":
                card_link.value = jsonData[key];
                break;
            case "template":
                template.value = jsonData[key];
                break;
            case "tags":
                tags.value = jsonData[key];
                break;
            default:
                customFields[key] = jsonData[key];
                break;
        }
    }
    for (const key in customFields) {
        if (key == "id" || key == "order") continue;
        // check if key already exists
        const existingField = document.querySelectorAll(`.field-name`);
        let keyExist = false;
        existingField.forEach(field => {
            if (field.value == key) {
                console.log("Key already exists: " + key)
                keyExist = true;
            }
        });
        if (keyExist) continue;
        
        
        const customFieldGroup = document.createElement('div');
        customFieldGroup.className = 'custom-field-group';

        // 字段名称输入框
        const nameInput = document.createElement('input');
        nameInput.type = "text";
        nameInput.placeholder = "Field Name (e.g.: link)";
        nameInput.className = "field-name";
        nameInput.value = key;

        // 字段值输入框
        const valueInput = document.createElement('input');
        valueInput.type = "text";
        valueInput.placeholder = "field value";
        valueInput.className = "field-value";
        valueInput.value = customFields[key];

        // 删除按钮
        const removeBtn = document.createElement('button');
        removeBtn.type = "button";
        removeBtn.textContent = "×";
        removeBtn.onclick = () => customFieldGroup.remove();

        customFieldGroup.appendChild(nameInput);
        customFieldGroup.appendChild(valueInput);
        customFieldGroup.appendChild(removeBtn);
        document.querySelector("#custom-fields-container").appendChild(customFieldGroup);
    }
}

function onExportButtonClick() {
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
        json = {
            card_title: document.getElementById('add-card-title').value.toString(),
            card_description: document.getElementById('add-card-description').value.toString(),
            card_link: document.getElementById('add-card-link').value.toString(),
            template: document.getElementById('add-card-template').value.toString(),
            tags: document.getElementById('add-card-tags').value.toString(),
            order: document.querySelectorAll('.card-container').length.toString(),
            // id: window._edittingCardID.toString()
        };
        for (const key in customFields) {
            json[key] = customFields[key].toString();
        }
        return json;
    }
    const jsondata = JSON.stringify(GetCardJson())
    console.log(jsondata);
    copyText(btoa(encodeURIComponent(jsondata)));
    
}

function CancleInputBox() {
    const add_card_box = document.querySelector(".card-input-box");
    const edit_settings_box = document.querySelector(".edit-settings-box");
    add_card_box?.remove();
    edit_settings_box?.remove();
    window._addingCard = false;
    window._edittingCard = false;
    window._edittingCustomSettings = false;
    window._edittingCardID = "";
}

function AddCardAPI(cardJson, callback) {
    const result = GetAccessPathAndToken();
    if (!result) {
        console.log("Access path and token are required.");
        return false;
    }
    const { path, token } = result;
    // console.log("Access path: " + path);
    // console.log("Access token: " + token);
    const api_dic = window.location.origin + "/" + path;
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

function GetAccessPathAndToken(DisableAsk) {
    let path = localStorage.getItem("access_path");
    let token = localStorage.getItem("access_token");
    if (path === null || token === null) {
        if (DisableAsk) {
            return null;
        }
        const newPath = prompt("Enter the access path (e.g. accessBackend):");
        if (newPath === null) return null;
        const newToken = prompt("Enter the access token:");
        if (newToken === null) return null;

        localStorage.setItem("access_path", newPath);
        localStorage.setItem("access_token", newToken);
        path = newPath;
        token = newToken;
    }
    token = generateEncryptToken(token);
    return { path, token };
}

function generateEncryptToken(token) {
    var encryptKey = `{{rendered:token_encrypt_key}}`;
    const timestamp = parseInt((new Date().getTime())/10000); // 时间梯度10s
    // console.log(timestamp);
    const timestampB64 = btoa(timestamp.toString());
    // console.log(timestampB64);
    encryptKey = encryptKey + timestampB64;
    let tokenArray = Array.from(btoa(token + "|" + encryptKey));
    let XorshiftSeed = 2166136261 >>> 0;

    for (let i = 0; i < tokenArray.length; i++) {
        XorshiftSeed = Math.imul(XorshiftSeed, 16777619);
        XorshiftSeed = (XorshiftSeed ^ tokenArray[i].charCodeAt(0)) >>> 0;
    }
    // console.log("XorshiftSeed: " + XorshiftSeed);
    const xorshift = new Xorshift32(XorshiftSeed);
    
    const getRandomChar = (seed) => String.fromCharCode(33 + ((seed+xorshift.next()) % 94));

    for (let i = 0; i < encryptKey.length; i++) {
        const charCode = encryptKey.charCodeAt(i);
        const operation = charCode % 5;

        switch (operation) {
            case 0:
                tokenArray.unshift(getRandomChar(charCode + i));
                break;

            case 1:
                if (tokenArray.length > 0) {
                    const pos = (charCode * i) % tokenArray.length;
                    tokenArray[pos] = getRandomChar(charCode ^ tokenArray[pos].charCodeAt(0));
                }
                break;

            case 2:
                mod = xorshift.next() % (tokenArray.length+1);
                if (mod == 0) {
                    mod = 1;
                }
                insertPos = charCode % mod
                // console.log("insertPos: " + insertPos);
                tokenArray.splice(insertPos, 0,
                    getRandomChar(charCode),
                    getRandomChar(charCode + 997)
                );
                break;

            case 3:
                if (tokenArray.length > 1) {
                    const pos1 = charCode % tokenArray.length;
                    const pos2 = tokenArray.length - 1 - pos1;
                    [tokenArray[pos1], tokenArray[pos2]] = [tokenArray[pos2], tokenArray[pos1]];
                }
                break;

            default:
                const pseudo = ['==', '=', '=A', 'B='][charCode % 4];
                tokenArray.push(...Array.from(pseudo));
        }
    }

    const finalShuffle = [];
    while (tokenArray.length > 0) {
        const randIndex = xorshift.next() % tokenArray.length;
        finalShuffle.push(tokenArray.splice(randIndex, 1)[0]);
    }

    return finalShuffle.join('');
}

class Xorshift32 {
    constructor(seed) {
        if (seed === 0) throw new Error("Seed cannot be zero");
        this.state = seed >>> 0;
    }

    next() {
        let x = this.state;
        x ^= x << 13;
        x ^= x >>> 17;
        x ^= x << 5;
        this.state = x >>> 0;
        return this.state;
    }

    random() {
        return this.next() / 0x100000000;
    }
}

function AddEditButtonListener() {
    const editButtons = document.querySelectorAll(".edit-button");
    const saveButtons = document.querySelectorAll(".save-button");
    // console.log(location.pathname)
    if (location.pathname.startsWith("/articles/")) {

        editButtons.forEach(button => {
            button.addEventListener("click", function (event) {
                event.preventDefault();
                location.href = '/editarticle.html?article_id=' + location.pathname.split('/')[2];
            });
        });
    } else if (location.pathname == "/index.html") {
        editButtons.forEach(button => {
            button.addEventListener("click", function (event) {
                event.preventDefault();
                EnterEditMode();
            });
        });
    }
    if (location.pathname == "/editarticle.html" || location.pathname == "/addarticle.html") {
        saveButtons.forEach(button => {
            button.addEventListener("click", function (event) {
                event.preventDefault();
                SaveArticle();
            })
        });
    }
}

AddEditButtonListener();
