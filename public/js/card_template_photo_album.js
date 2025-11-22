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
            if (data.preloadAll || i < 2) { // render first 2 photos or all photos
                var photo = document.createElement("div");
                photo.classList.add("photo");
                var photo_img = document.createElement("img");
                photo_img.src = photo_url;
                photo_img.alt = "photo";
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
                photo_img.addEventListener("click", function (e) {
                    show_big_photo(photo_url);
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

            if (card.album_index <= 0) { // if first photo
                console.log("first photo - already at beginning")
                return;
            }

            var prevIndex = card.album_index - 1;
            // calc new x
            card.current_x -= card.photo_rect[card.photo_list[prevIndex].url].width;
            // set photo transfrom
            card.querySelector(".photo-container").style.transform = "translateX(-" + card.current_x + "px)";
            // set current photo class
            card.photo_list[card.album_index].dom.classList.remove("current");
            card.photo_list[card.album_index - 1]?.dom.classList.remove("prev");
            card.photo_list[card.album_index + 1]?.dom.classList.remove("next");

            card.photo_list[prevIndex].dom.classList.add("current");
            card.photo_list[prevIndex + 1]?.dom.classList.add("next");
            card.photo_list[prevIndex - 1]?.dom.classList.add("prev");
            // set album rect
            set_photo_album_container_rect(card, card.photo_rect[card.photo_list[prevIndex].url].width, card.photo_rect[card.photo_list[prevIndex].url].height);
            // set album index
            card.album_index = prevIndex;
        });
        card.querySelector(".controll-container").querySelector(".next-btn").addEventListener("click", function (event) {
            console.log("next");

            if (card.album_index >= card.album_total - 1) { // if last photo
                console.log("last photo - already at end")
                return;
            }

            var nextIndex = card.album_index + 1;
            // calc new x
            card.current_x += card.photo_rect[card.photo_list[card.album_index].url].width;
            // set photo transfrom
            card.querySelector(".photo-container").style.transform = "translateX(-" + card.current_x + "px)";
            // set current photo class
            card.photo_list[card.album_index].dom.classList.remove("current");
            card.photo_list[card.album_index - 1]?.dom.classList.remove("prev");
            card.photo_list[card.album_index + 1]?.dom.classList.remove("next");

            card.photo_list[nextIndex].dom.classList.add("current");
            card.photo_list[nextIndex + 1]?.dom.classList.add("next");
            card.photo_list[nextIndex - 1]?.dom.classList.add("prev");
            // set album rect
            set_photo_album_container_rect(card, card.photo_rect[card.photo_list[nextIndex].url].width, card.photo_rect[card.photo_list[nextIndex].url].height);
            // set album index
            card.album_index = nextIndex;
            // add new photo (lazy loading)
            console.log(card.album_total, card.album_index)
            if (card.album_total > nextIndex + 1 && card.photo_loaded[card.album_data.album[nextIndex + 1]] == false) {
                var photo = document.createElement("div");
                photo.classList.add("photo");
                photo.classList.add("next");
                var photo_img = document.createElement("img");
                photo_img.src = card.album_data.album[nextIndex + 1];
                photo.appendChild(photo_img);
                photo_container.appendChild(photo);
                card.photo_loaded[card.album_data.album[nextIndex + 1]] = true;
                // add event listener load to photo
                photo_img.addEventListener("load", function () {
                    card.photo_rect[card.album_data.album[nextIndex + 1]] = {
                        naturalHeight: photo_img.naturalHeight,
                        naturalWidth: photo_img.naturalWidth,
                        height: 300,
                        width: 300 * get_photo_width_height_ratio(photo_img.naturalWidth, photo_img.naturalHeight)
                    };
                    photo.style.width = card.photo_rect[card.album_data.album[nextIndex + 1]].width + "px";
                    photo.style.height = card.photo_rect[card.album_data.album[nextIndex + 1]].height + "px";
                });
                photo.addEventListener("click", function () {
                    show_big_photo(card.album_data.album[nextIndex + 1]);
                });
                card.photo_list.push({
                    dom: photo,
                    url: card.album_data.album[nextIndex + 1]
                });
                card.photo_loaded[card.album_data.album[nextIndex + 1]] = true;
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
    ret.preloadAll = info_container.getAttribute("data-preload-all") == "true";
    // remove info container
    info_container.remove();
    return ret;
}

function get_photo_width_height_ratio(width, height) {
    return width / height;
}

function autoShift(card_photo_album, sec) {
    var nextBtn = card_photo_album.querySelector(".next-btn");
    var prevBtn = card_photo_album.querySelector(".prev-btn");
    var interval = setInterval(function () {
        if (card_photo_album.album_index < card_photo_album.album_total - 1) { // if not last photo
            nextBtn.click();
        } else { // if last photo, go back to first
            // Click prev button multiple times to return to first photo
            for (var i = 0; i < card_photo_album.album_total - 1; i++) {
                prevBtn.click();
            }
        }
    }, sec * 1000);
}

function show_big_photo(photo_url) {
    // remove big photo container if exist
    const big_photo_container_old = document.querySelector(".big-photo-container");
    big_photo_container_old?.remove();
    console.log("show big photo", photo_url);

    // create big photo container
    var big_photo_container = document.createElement("div");
    big_photo_container.classList.add("big-photo-container");

    // create image
    var big_photo_img = document.createElement("img");
    big_photo_img.src = photo_url;
    big_photo_img.classList.add("big-photo-image");

    // rotation state
    var rotation = 0;
    var scale = 1;

    // drag state
    var isDragging = false;
    var dragStart = { x: 0, y: 0 };
    var imgPosition = { x: 0, y: 0 };

    // create control bar
    var control_bar = document.createElement("div");
    control_bar.classList.add("big-photo-control-bar");

    // close button
    var close_btn = document.createElement("div");
    close_btn.classList.add("control-btn", "close-btn");
    close_btn.innerHTML = `
        <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
        </svg>
    `;
    close_btn.title = "Close (Esc)";

    // rotate left button
    var rotate_left_btn = document.createElement("div");
    rotate_left_btn.classList.add("control-btn", "rotate-left-btn");
    rotate_left_btn.innerHTML = `
        <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M7.11 8.53L5.7 7.11C4.8 8.27 4.24 9.61 4.07 11h2.02c.14-.87.49-1.72 1.02-2.47zM6.09 13H4.07c.17 1.39.72 2.73 1.62 3.89l1.41-1.42c-.52-.75-.87-1.59-1.01-2.47zm1.01 5.32c1.16.9 2.51 1.44 3.9 1.61V17.9c-.87-.15-1.71-.49-2.46-1.03L7.1 18.32zM13 4.07V1L8.45 5.55 13 10V6.09c2.84.48 5 2.94 5 5.91s-2.16 5.43-5 5.91v2.02c3.95-.49 7-3.85 7-7.93s-3.05-7.44-7-7.93z"/>
        </svg>
    `;
    rotate_left_btn.title = "Rotate left (←)";

    // rotate right button
    var rotate_right_btn = document.createElement("div");
    rotate_right_btn.classList.add("control-btn", "rotate-right-btn");
    rotate_right_btn.innerHTML = `
        <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M15.55 5.55L11 1v3.07C7.06 4.56 4 7.92 4 12s3.05 7.44 7 7.93v-2.02c-2.84-.48-5-2.94-5-5.91s2.16-5.43 5-5.91V10l4.55-4.45zM19.93 11c-.17-1.39-.72-2.73-1.62-3.89l-1.42 1.42c.54.75.88 1.6 1.02 2.47h2.02zM13 17.9v2.02c1.39-.17 2.74-.71 3.9-1.61l-1.44-1.44c-.75.54-1.59.89-2.46 1.03zm3.89-2.42l1.42 1.41c.9-1.16 1.45-2.5 1.62-3.89h-2.02c-.14.87-.48 1.72-1.02 2.48z"/>
        </svg>
    `;
    rotate_right_btn.title = "Rotate right (→)";

    // zoom out button
    var zoom_out_btn = document.createElement("div");
    zoom_out_btn.classList.add("control-btn", "zoom-out-btn");
    zoom_out_btn.innerHTML = `
        <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M15.5 14h-.79l-.28-.27C15.41 12.59 16 11.11 16 9.5 16 5.91 13.09 3 9.5 3S3 5.91 3 9.5 5.91 16 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14zM7 9h5v1H7z"/>
        </svg>
    `;
    zoom_out_btn.title = "Zoom out (-)";

    // zoom in button
    var zoom_in_btn = document.createElement("div");
    zoom_in_btn.classList.add("control-btn", "zoom-in-btn");
    zoom_in_btn.innerHTML = `
        <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M15.5 14h-.79l-.28-.27C15.41 12.59 16 11.11 16 9.5 16 5.91 13.09 3 9.5 3S3 5.91 3 9.5 5.91 16 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14zm.5-7H9v2H7v1h2v2h1v-2h2V9h-2z"/>
        </svg>
    `;
    zoom_in_btn.title = "Zoom in (+)";

    // reset button
    var reset_btn = document.createElement("div");
    reset_btn.classList.add("control-btn", "reset-btn");
    reset_btn.innerHTML = `
        <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 5V1L7 6l5 5V7c3.31 0 6 2.69 6 6s-2.69 6-6 6-6-2.69-6-6H4c0 4.42 3.58 8 8 8s8-3.58 8-8-3.58-8-8-8z"/>
        </svg>
    `;
    reset_btn.title = "Reset (R)";

    // assemble control bar
    control_bar.appendChild(close_btn);
    control_bar.appendChild(rotate_left_btn);
    control_bar.appendChild(rotate_right_btn);
    control_bar.appendChild(zoom_out_btn);
    control_bar.appendChild(zoom_in_btn);
    control_bar.appendChild(reset_btn);

    // assemble container
    big_photo_container.appendChild(big_photo_img);
    big_photo_container.appendChild(control_bar);
    document.body.appendChild(big_photo_container);

    // update image transform
    function updateTransform() {
        big_photo_img.style.transform = `translate(${imgPosition.x}px, ${imgPosition.y}px) rotate(${rotation}deg) scale(${scale})`;
    }

    // close handler
    function closePhoto() {
        big_photo_container.remove();
    }

    // rotate left
    rotate_left_btn.addEventListener("click", function (event) {
        event.stopPropagation();
        rotation -= 90;
        updateTransform();
    });

    // rotate right
    rotate_right_btn.addEventListener("click", function (event) {
        event.stopPropagation();
        rotation += 90;
        updateTransform();
    });

    // zoom in
    zoom_in_btn.addEventListener("click", function (event) {
        event.stopPropagation();
        scale = Math.min(scale + 0.2, 3);
        updateTransform();
    });

    // zoom out
    zoom_out_btn.addEventListener("click", function (event) {
        event.stopPropagation();
        scale = Math.max(scale - 0.2, 0.5);
        updateTransform();
    });

    // reset
    reset_btn.addEventListener("click", function (event) {
        event.stopPropagation();
        imgPosition.x = 0;
        imgPosition.y = 0;
        rotation = 0;
        scale = 1;
        updateTransform();
    });

    // close button
    close_btn.addEventListener("click", function (event) {
        event.stopPropagation();
        closePhoto();
    });

    // drag image
    big_photo_img.addEventListener("mousedown", function (event) {
        event.preventDefault();
        isDragging = true;
        dragStart.x = event.clientX - imgPosition.x;
        dragStart.y = event.clientY - imgPosition.y;
        big_photo_img.style.cursor = "grabbing";
    });

    document.addEventListener("mousemove", function (event) {
        if (!isDragging) return;
        imgPosition.x = event.clientX - dragStart.x;
        imgPosition.y = event.clientY - dragStart.y;
        updateTransform();
    });

    document.addEventListener("mouseup", function () {
        if (isDragging) {
            isDragging = false;
            big_photo_img.style.cursor = "grab";
        }
    });

    // set initial cursor
    big_photo_img.style.cursor = "grab";

    // click background to close
    big_photo_container.addEventListener("click", function (event) {
        if (event.target === big_photo_container) {
            closePhoto();
        }
    });

    // keyboard shortcuts
    function handleKeyDown(event) {
        switch (event.key) {
            case 'Escape':
                closePhoto();
                break;
            case 'ArrowLeft':
                rotation -= 90;
                updateTransform();
                break;
            case 'ArrowRight':
                rotation += 90;
                updateTransform();
                break;
            case '+':
            case '=':
                scale = Math.min(scale + 0.2, 3);
                updateTransform();
                break;
            case '-':
            case '_':
                scale = Math.max(scale - 0.2, 0.5);
                updateTransform();
                break;
            case 'r':
            case 'R':
                // reset position, rotation and scale
                imgPosition.x = 0;
                imgPosition.y = 0;
                rotation = 0;
                scale = 1;
                updateTransform();
                break;
        }
    }

    document.addEventListener('keydown', handleKeyDown);

    // cleanup on remove
    const observer = new MutationObserver(function (mutations) {
        mutations.forEach(function (mutation) {
            mutation.removedNodes.forEach(function (node) {
                if (node === big_photo_container) {
                    document.removeEventListener('keydown', handleKeyDown);
                    observer.disconnect();
                }
            });
        });
    });
    observer.observe(document.body, { childList: true });

    big_photo_container.addEventListener("contextmenu", function (event) {
        event.stopPropagation();
    });
}