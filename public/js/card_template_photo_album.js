

if (!window._card_photo_album_loaded) {
    window._card_photo_album_loaded = true;

    window.addEventListener("DOMContentLoaded", function () {
        card_photo_album_init();
    });
}

function card_photo_album_init() {

    var card_photo_album = document.querySelectorAll(".card-photo-album");
    card_photo_album.forEach(function (card) {
        var photo_container = card.querySelector(".photo-container");
        var data = get_photo_album_data(card);
        console.log(data);

        // set album variables
        card.album_index = 0;
        card.current_x = 0;
        card.album_total = data.album.length;

        data.album.forEach(function (photo_url) {
            var photo = document.createElement("div");
            photo.classList.add("photo");
            var photo_img = document.createElement("img");
            photo_img.src = photo_url;
            photo.appendChild(photo_img);
            photo_container.appendChild(photo);

        });
        // add event listener to photo container
        card.querySelector(".controll-container").querySelector(".prev-btn").addEventListener("click", function (event) {
            console.log("prev");
        });
        card.querySelector(".controll-container").querySelector(".next-btn").addEventListener("click", function (event) {
            console.log("next");
        });
    });
}

function get_photo_album_data(card_photo_album) {
    // select info container
    var info_container = card_photo_album.querySelector(".album-info-container");
    var ret = {}
    ret.album = info_container.getAttribute("data-album").split("|");
    // remove info container
    info_container.remove();
    return ret;
}
