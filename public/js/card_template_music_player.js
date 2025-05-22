if (!window._card_music_player_loaded) {
    window._card_music_player_loaded = true;
    
    window.addEventListener("load", function() {
        card_music_player_init();
    });
}

function card_music_player_init() {
    // select music player cards
    var music_player_cards = document.querySelectorAll(".card-music-player");
    console.log(music_player_cards);
    music_player_cards.forEach(function(card) {
        const music_info = get_music_info(card);
        console.log(music_info);
        // set background image
        card.querySelector("#image-container").style.backgroundImage = "url('" + music_info["image"] + "')";
        
        // add event listeners
        card.addEventListener("mouseover", function (e) {
            music_player_on_hover(e,card);
        });
        card.addEventListener("mouseout", function (e) {
            music_player_on_mouseout(e,card);
        });
    });
}

function music_player_on_hover(e,card) {
    // query player-shift-container
    var player_shift_container = card.querySelector(".player-shift-container");
    var player_player_container = card.querySelector(".player-container");
    var player_image_container = card.querySelector("#image-container");
    player_shift_container.style.display = "block";
    // remove transform
    player_shift_container.style.transform = "translateX(0px)";
    // set player z-index to 1
    // player_player_container.style.zIndex = 1;
    // set image z-index to 2
    // player_image_container.style.zIndex = 2;
    // add filter: blur to image container
    card.querySelector("#image-container").style.filter = "blur(5px)";
}

function music_player_on_mouseout(e,card) {
    // query player-shift-container
    var player_shift_container = card.querySelector(".player-shift-container");
    var player_player_container = card.querySelector(".player-container");
    var player_image_container = card.querySelector("#image-container");
    player_shift_container.style.display = "none";
    // add transform
    player_shift_container.style.transform = "translateY(100%)";
    // set player z-index to 2
    // player_player_container.style.zIndex = 2;
    // set image z-index to 1
    // player_image_container.style.zIndex = 1;
    // remove filter: blur from image container
    card.querySelector("#image-container").style.filter = "blur(0px)";
}

function get_music_info(card) {
    // get music info from card
    var music_info_container = card.querySelector(".music-info-container");
    var returnvar = {}
    returnvar["title"] = music_info_container.getAttribute("data-music-title");
    returnvar["artist"] = music_info_container.getAttribute("data-music-artist");
    returnvar["link"] = music_info_container.getAttribute("data-music-link");
    returnvar["image"] = music_info_container.getAttribute("data-music-image");
    returnvar["lyricLink"]= music_info_container.getAttribute("data-music-lyric");
    return returnvar;
}

class MusicPlayer {
    constructor(music_info) {
        this.music_info = music_info;
        // music_info should include title, artist, link
    }
    init() {

    }
}