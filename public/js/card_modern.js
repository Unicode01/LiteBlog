if (!window._card_modern_loaded) {
    window._card_modern_loaded = true;

    window.addEventListener("DOMContentLoaded", function () {
        card_modern_init();
    });
}

function card_modern_init() {
    // select modern cards
    var modern_cards = document.querySelectorAll('.card-modern');
    for (var i = 0; i < modern_cards.length; i++) {
        render_modern_card_content(modern_cards[i]);
        render_modern_card_tags(modern_cards[i]);
    }
    console.log('card_modern_init');
}

function render_modern_card_content(dom_card) {
    // get info from data container
    const infoContainer = dom_card.querySelector('.card-info-container');
    if (!infoContainer) return;

    const cardTitle = infoContainer.dataset.cardTitle || '';
    const cardLink = infoContainer.dataset.cardLink || '';
    const cardDescription = infoContainer.dataset.cardDescription || '';

    // render title
    const titleDom = dom_card.querySelector('.post-card-header');
    if (titleDom) {
        titleDom.textContent = cardTitle;
    }

    // render link
    const linkDom = dom_card.querySelector('.card-link');
    if (linkDom) {
        linkDom.href = cardLink;
    }

    // render description
    const descDom = dom_card.querySelector('.card-description');
    if (descDom) {
        descDom.textContent = cardDescription;
    }

    // set alt attribute
    dom_card.setAttribute('alt', cardTitle);
}

function render_modern_card_tags(dom_card) {
    // get info from data container
    const infoContainer = dom_card.querySelector('.card-info-container');
    if (!infoContainer) return;

    const tagsStr = infoContainer.dataset.cardTags || '';
    const tagDom = dom_card.querySelector('.card-tags');
    if (!tagDom) return;

    // parse and render tags
    const tags = tagsStr.split(' ').filter(t => t.trim() !== '');
    let newTagHtml = '';
    for (var i = 0; i < tags.length; i++) {
        const tag_name = tags[i];
        newTagHtml += `
<div class="tag padding-10px font-size-14px">
    <a href="#tag-${tag_name}" class="link">${tag_name}</a>
</div>`;
    }
    tagDom.innerHTML = newTagHtml;
}
