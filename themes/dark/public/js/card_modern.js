if (!window._card_modern_loaded) {
    window._card_modern_loaded = true;
    
    window.addEventListener("load", function() {
        card_modern_init();
    });
}

function card_modern_init() {
    // select modern cards
    var modern_cards = document.querySelectorAll('.card-modern');
    for (var i = 0; i < modern_cards.length; i++) {
        init_card_tags(modern_cards[i]);
    }
    console.log('card_modern_init');
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