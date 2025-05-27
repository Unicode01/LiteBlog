if (!window._card_classical_loaded) {
    window._card_classical_loaded = true;
    
    window.addEventListener("load", function() {
        card_classical_init();
    });
}

function card_classical_init() {
    // select classical cards
    var classical_cards = document.querySelectorAll('.card-classical');
    for (var i = 0; i < classical_cards.length; i++) {
        init_card_tags(classical_cards[i]);
    }
    console.log('card_classical_init');
}

function init_card_tags(dom_card) {
    // select tags
    tagDom = dom_card.querySelector('.card-tags')
    tags = tagDom.innerHTML.split(' ');
    console.log(tags);
    // rerender tags
    newTagHtml = '';
    for (var i = 0; i < tags.length; i++) {
        tag_name = tags[i];
        taghtml = `
<div class="tag padding-10px font-size-14px">
    <a href="#tag-${tag_name}" class="link">${tag_name}</a>
</div>
        `
        newTagHtml += taghtml;
    }
    tagDom.innerHTML = newTagHtml;
    
}