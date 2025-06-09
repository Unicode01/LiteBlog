// This file is used to initialize the photo album card template.
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
        card.photo_loaded = {};
        card.photo_list = [];
        card.photo_rect = {};
        card.album_data = data;
        card.album_total = data.album.length;
        var i = 0;
        data.album.forEach(function (photo_url) {
            if (i < 2) { // render first 2 photos
                var photo = document.createElement("div");
                photo.classList.add("photo");
                var photo_img = document.createElement("img");
                photo_img.src = photo_url;
                photo.appendChild(photo_img);
                photo_container.appendChild(photo);
                card.photo_loaded[photo_url] = true;
                if (i == card.album_index) {
                    photo.classList.add("current");
                } else if (i < card.album_index) {
                    photo.classList.add("prev");
                } else {
                    photo.classList.add("next");
                }
                // add event listener load to photo
                photo_img.addEventListener("load", function () {
                    card.photo_rect[photo_url] = {
                        naturalHeight: photo_img.naturalHeight,
                        naturalWidth: photo_img.naturalWidth,
                        height: 300,
                        width: 300 * get_photo_width_height_ratio(photo_img.naturalWidth, photo_img.naturalHeight)
                    };
                    photo.style.width = card.photo_rect[photo_url].width + "px";
                    photo.style.height = card.photo_rect[photo_url].height + "px";
                    // set container rect
                    if (card.photo_list[card.album_index].url == photo_url) {
                        set_photo_album_container_rect(card, card.photo_rect[photo_url].width, card.photo_rect[photo_url].height);
                    }
                });
                card.photo_list.push({
                    dom: photo,
                    url: photo_url
                });
                card.photo_loaded[photo_url] = true;
            } else {
                card.photo_loaded[photo_url] = false;
            }
            i++;
        });
        // add event listener to photo container
        card.querySelector(".controll-container").querySelector(".prev-btn").addEventListener("click", function (event) {
            console.log("prev");

            if (card.album_index < 1) { // if first photo
                console.log("first photo")
                return;
            }
            prevIndex = card.album_index-1;
            // calc new x
            card.current_x -= card.photo_rect[card.photo_list[card.album_index-1].url].width;
            // set photo transfrom
            card.querySelector(".photo-container").style.transform = "translateX(-" + card.current_x + "px)";
            // set current photo class
            card.photo_list[card.album_index].dom.classList.remove("current");
            card.photo_list[card.album_index-1]?.dom.classList.remove("prev");
            card.photo_list[card.album_index+1]?.dom.classList.remove("next");

            card.photo_list[prevIndex].dom.classList.add("current");
            card.photo_list[prevIndex+1]?.dom.classList.add("next");
            card.photo_list[prevIndex-1]?.dom.classList.add("prev");
            // set album rect
            set_photo_album_container_rect(card, card.photo_rect[card.photo_list[prevIndex].url].width, card.photo_rect[card.photo_list[prevIndex].url].height);
            // set album index
            card.album_index=prevIndex;
            
        });
        card.querySelector(".controll-container").querySelector(".next-btn").addEventListener("click", function (event) {
            console.log("next");

            if (card.album_total < card.album_index+2) { // if last photo
                console.log("last photo")
                return;
            }
            nextIndex = card.album_index+1;
            // calc new x
            card.current_x += card.photo_rect[card.photo_list[card.album_index].url].width;
            // set photo transfrom
            card.querySelector(".photo-container").style.transform = "translateX(-" + card.current_x + "px)";
            // set current photo class
            card.photo_list[card.album_index].dom.classList.remove("current");
            card.photo_list[card.album_index-1]?.dom.classList.remove("prev");
            card.photo_list[card.album_index+1]?.dom.classList.remove("next");

            card.photo_list[nextIndex].dom.classList.add("current");
            card.photo_list[nextIndex+1]?.dom.classList.add("next");
            card.photo_list[nextIndex-1]?.dom.classList.add("prev");
            // set album rect
            set_photo_album_container_rect(card, card.photo_rect[card.photo_list[nextIndex].url].width, card.photo_rect[card.photo_list[nextIndex].url].height);
            // set album index
            card.album_index=nextIndex;
            // add new photo
            console.log(card.album_total, card.album_index)
            if (card.album_total > nextIndex+1 && card.photo_loaded[card.album_data.album[nextIndex+1]] == false) {
                var photo = document.createElement("div");
                photo.classList.add("photo");
                photo.classList.add("next");
                var photo_img = document.createElement("img");
                photo_img.src = card.album_data.album[nextIndex+1];
                photo.appendChild(photo_img);
                photo_container.appendChild(photo);
                card.photo_loaded[card.album_data.album[nextIndex+1]] = true;
                // add event listener load to photo
                photo_img.addEventListener("load", function () {
                    card.photo_rect[card.album_data.album[nextIndex+1]] = {
                        naturalHeight: photo_img.naturalHeight,
                        naturalWidth: photo_img.naturalWidth,
                        height: 300,
                        width: 300 * get_photo_width_height_ratio(photo_img.naturalWidth, photo_img.naturalHeight)
                    };
                    photo.style.width = card.photo_rect[card.album_data.album[nextIndex+1]].width + "px";
                    photo.style.height = card.photo_rect[card.album_data.album[nextIndex+1]].height + "px";
                }); 
                card.photo_list.push({
                    dom: photo,
                    url: card.album_data.album[nextIndex+1]
                });
                card.photo_loaded[card.album_data.album[nextIndex+1]] = true;
            }
        });
        if (data.autoShift) {
            autoShift(card, data.autoShift);
        }
    });
}

function set_photo_album_container_rect(card_photo_album, width, height) {
    card_photo_album.style.setProperty("--album-width", width + "px");
    card_photo_album.style.setProperty("--album-height", height + "px");
}

function get_photo_album_data(card_photo_album) {
    // select info container
    var info_container = card_photo_album.querySelector(".album-info-container");
    var ret = {}
    ret.album = info_container.getAttribute("data-album").split("|");
    ret.autoShift = parseInt(info_container.getAttribute("data-auto-shift"));
    // remove info container
    info_container.remove();
    return ret;
}

function get_photo_width_height_ratio(width,height) {
    return width/height;
}

function autoShift(card_photo_album,sec) {
    nextBtn = card_photo_album.querySelector(".next-btn");
    prevBtn = card_photo_album.querySelector(".prev-btn");
    interval = setInterval(function () {
        if (card_photo_album.album_index < card_photo_album.album_total-1) { // if not last photo
            nextBtn.click();
        } else { // if last photo
            for (var i = 0; i < card_photo_album.album_total; i++) {
                prevBtn.click();
            }
        }
    }, sec*1000);
}