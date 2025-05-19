function CardsFliterTag(Tag) {
    // select all cards with the tag
    var cards = document.querySelectorAll('.card-container');
    var cards_tags = {};
    for (var i = 0; i < cards.length; i++) {
        var tags = [];
        // select tags of the card
        const card_tags_arr = cards[i].querySelectorAll('.tag');
        for (var j = 0; j < card_tags_arr.length; j++) {
            tags.push(card_tags_arr[j].querySelector('a').textContent.trim())
        }
        cards_tags[cards[i].getAttribute('card-id')] = tags;
    }
    console.log(cards_tags);
    // filter cards by tag
    var filtered_cards = [];
    for (var card_id in cards_tags) {
        if (cards_tags[card_id].includes(Tag)) {
            filtered_cards.push(card_id);
        }
    }
    console.log(filtered_cards);
    // hide all cards
    for (var i = 0; i < cards.length; i++) {
        cards[i].style.display = 'none';
    }
    // show filtered cards
    for (var i = 0; i < cards.length; i++) {
        for (var j = 0; j < filtered_cards.length; j++) {
            if (cards[i].getAttribute('card-id') == filtered_cards[j]) {
                cards[i].style.display = 'flex';
            }
        }
    }
}

function AddTagListener() {
    // select all tags
    var tags = document.querySelectorAll('.tag');
    for (var i = 0; i < tags.length; i++) {
        tags[i].addEventListener('click', function() {
            const tag = this.querySelector("a").textContent.trim();
            console.log(tag);
            CardsFliterTag(tag);
        });
    }
}

window.onload = function() {
    AddTagListener();
}