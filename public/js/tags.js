var cards_display = {};
var currentFilterTag = '';

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

        // console.log(window.getComputedStyle(cards[i]).display)
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
                cards[i].style.display = cards_display[filtered_cards[j]] || 'flex';
            }
        }
    }
    currentFilterTag = Tag;
}

function RemoveFilter() {
    // select all cards
    var cards = document.querySelectorAll('.card-container');
    cards.forEach(function(card) {
        // console.log(cards_display);
        const display = cards_display[card.getAttribute('card-id')] || 'flex';
        card.style.display = display;
        currentFilterTag = '';
    });
}

function GetCurrentTag() {
    if (currentFilterTag != '') {
        return currentFilterTag;
    }
    var current_tag = window.location.hash.slice(1);
    // check and remove 'tag-'
    if (current_tag.startsWith('tag-')) {
        current_tag = current_tag.slice(4);
    }
    return current_tag;
}

function AddTagListener() {
    // select all tags
    var tags = document.querySelectorAll('.tag');
    for (var i = 0; i < tags.length; i++) {
        tags[i].querySelector('a').addEventListener('click', function(event) {
            const currentTag = GetCurrentTag();
            const tag = this.textContent.trim();
            console.log(currentTag, tag)
            RemoveFilter();
            if (currentTag == tag) {
                // prevent default behavior to avoid page reload
                event.preventDefault();
                console.log("removed filter")
                const cleanUrl = window.location.href.split('#')[0];
                window.history.replaceState(null, null, cleanUrl);
            } else {
                console.log(tag)
                CardsFliterTag(tag);
            }
        });
    }
}

function logCardsDisplay() {
    var cards = document.querySelectorAll('.card-container');
    cards.forEach(function(card) {
        cards_display[card.getAttribute('card-id')] = window.getComputedStyle(card).display;
    });
}

window.addEventListener("DOMContentLoaded", function() {
    logCardsDisplay();
    AddTagListener();
    // get current tag and filter cards
    const currentTag = GetCurrentTag();
    if (currentTag.slice(0, 4) == 'tag-') {
        CardsFliterTag(currentTag);
    };
});