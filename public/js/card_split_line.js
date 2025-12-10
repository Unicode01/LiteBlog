if (!window._card_split_line_loaded) {
    window._card_split_line_loaded = true;

    window.addEventListener("DOMContentLoaded", function () {
        card_split_line_init();
    });
}

function card_split_line_init() {
    // select split line cards
    var split_line_cards = document.querySelectorAll('.card-container-split-line');
    for (var i = 0; i < split_line_cards.length; i++) {
        render_split_line_content(split_line_cards[i]);
    }
    console.log('card_split_line_init');
}

function render_split_line_content(dom_card) {
    // get info from data container
    const infoContainer = dom_card.querySelector('.card-info-container');
    if (!infoContainer) return;

    const cardId = infoContainer.dataset.cardId || '';
    const cardTitle = infoContainer.dataset.cardTitle || '';

    // render title
    const titleDom = dom_card.querySelector('.split-line-title');
    if (titleDom) {
        titleDom.textContent = cardTitle;
    }

    // bind stack button click event
    const stackButton = dom_card.querySelector('.stack-button');
    if (stackButton && cardId) {
        stackButton.onclick = function () {
            stackCard(document.querySelector('#card-container[card-id=\'' + cardId + '\']'), null);
        };
    }
}

