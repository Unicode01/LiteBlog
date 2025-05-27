if (!window._card_music_player_loaded) {
    window._card_music_player_loaded = true;

    window.addEventListener("load", function () {
        window._cards_for_musics = [];
        card_music_player_init();
    });
}

function card_music_player_init() {
    // select music player cards
    var music_player_cards = document.querySelectorAll(".card-music-player");
    console.log(music_player_cards);
    i = 0
    music_player_cards.forEach(function (card) {
        card.thisMusicIndex = i;
        i++;
        const music_info = get_music_info(card);
        // check bg theme
        if (music_info.imageTheme === "dark") {
            // reset dom colors
            card.querySelector(".player-title").style.color = "white";
            card.querySelector(".player-artist").style.color = "#999";
            card.querySelector(".player-lyric-pre").style.color = "#999";
            card.querySelector(".player-lyric-current").style.color = "white";
            card.querySelector(".player-lyric-next").style.color = "#999";
            card.querySelector(".player-duration").style.color = "white";
            card.querySelector(".player-current-time").style.color = "white";
            card.querySelector(".player-pause").style.color = "white";
            card.style.boxShadow = "0 0 10px #3A3A3A";
        }
        console.log(music_info);
        // set background image
        card.querySelector("#image-container").style.backgroundImage = "url('" + music_info["image"] + "')";

        // add event listeners
        card.addEventListener("mousemove", function (e) {
            music_player_on_move(e, card);
        });
        card.addEventListener("mouseleave", function (e) {
            music_player_on_mouseleave(e, card);
        });
        // init music player
        const music_player = new MusicPlayer(music_info);
        card.thisMusicPlayer = music_player;
        music_player.init();
        music_player.addEventListener("timeupdate", function () {
            // update lyrics
            const currentLyricIndex = music_player.getCurrentLyric();
            const prevLyricDom = card.querySelector(".player-lyric-pre");
            const currentLyricDom = card.querySelector(".player-lyric-current");
            const nextLyricDom = card.querySelector(".player-lyric-next");
            var player_hover_line = card.querySelector(".player-hover-line");
            if (currentLyricIndex != -1) {
                if (currentLyricIndex > 0) {
                    prevLyricDom.textContent = music_player.lyrics[currentLyricIndex - 1].text;
                } else {
                    prevLyricDom.textContent = "";
                }
                currentLyricDom.textContent = music_player.lyrics[currentLyricIndex].text;
                if (currentLyricIndex < music_player.lyrics.length - 1) {
                    nextLyricDom.textContent = music_player.lyrics[currentLyricIndex + 1].text;
                    if (player_hover_line.style.opacity === "0") {
                        player_hover_line.style.opacity = "1";
                    }
                } else {
                    nextLyricDom.textContent = "";
                }
            }
            // update progress bar and player-current-time
            const progress_bar = card.querySelector(".player-progress-bar");
            const progress_bar_width = (music_player.getCurrentTime() / music_player.getDuration()) * 100;
            const progress_bar_current_time = card.querySelector(".player-current-time");
            progress_bar_current_time.textContent = format_time(music_player.getCurrentTime());
            var bg_linear_gradient = "linear-gradient(to right, #999 0%, #999 " + progress_bar_width + "%, #fff " + progress_bar_width + "%, #fff 100%)";
            progress_bar.style.background = bg_linear_gradient;
        });
        // add load event listener
        music_player.addEventListener("loadedmetadata", function () {
            // update player-duration
            const progress_bar_duration = card.querySelector(".player-duration");
            progress_bar_duration.textContent = format_time(music_player.getDuration());
        });
        // add ended event listener
        music_player.addEventListener("ended", function () {
            // update player-current-time
            const progress_bar_current_time = card.querySelector(".player-current-time");
            progress_bar_current_time.textContent = format_time(music_player.getDuration());
            // update player-play-button
            const player_play_button = card.querySelector(".player-pause");
            player_play_button.innerHTML = "▶";
        });
        // add progress bar event listener
        const progress_bar = card.querySelector(".player-progress-bar");
        progress_bar.addEventListener("click", function (e) {
            const progress_bar_width = e.offsetX / progress_bar.offsetWidth;
            const progress_bar_current_time = card.querySelector(".player-current-time");
            const seekTime = progress_bar_width * music_player.getDuration();
            progress_bar_current_time.textContent = format_time(seekTime);
            music_player.seek(seekTime);
        });
        const player_play_button = card.querySelector(".player-pause");
        const player_prev_button = card.querySelector(".player-prev");
        const player_next_button = card.querySelector(".player-next");
        // add event listeners
        player_play_button.addEventListener("click", function () {
            if (music_player.isPaused()) {
                music_player.play();
                player_play_button.innerHTML = "⏸"
            } else {
                music_player.pause();
                player_play_button.innerHTML = "▶"
            }
        });
        // add prev button event listener
        player_prev_button.addEventListener("click", function () {
            const newMusicCardIndex = card.thisMusicIndex - 1;
            console.log(newMusicCardIndex)
            if (newMusicCardIndex >= 0 && newMusicCardIndex <= window._cards_for_musics.length) {
                console.log(window._cards_for_musics)
                window._cards_for_musics[newMusicCardIndex].thisMusicPlayer.play();
                window._cards_for_musics[newMusicCardIndex].querySelector(".player-pause").innerHTML = "⏸";
                // stop this playing
                card.thisMusicPlayer.stop();
                card.querySelector(".player-pause").innerHTML = "▶";
            }
        });
        // add next button event listener
        player_next_button.addEventListener("click", function () {
            const newMusicCardIndex = card.thisMusicIndex + 1;
            console.log(newMusicCardIndex)
            if (newMusicCardIndex >= 0 && newMusicCardIndex < window._cards_for_musics.length) {
                console.log(window._cards_for_musics)
                window._cards_for_musics[newMusicCardIndex].thisMusicPlayer.play();
                window._cards_for_musics[newMusicCardIndex].querySelector(".player-pause").innerHTML = "⏸";
                // stop this playing
                card.thisMusicPlayer.stop();
                card.querySelector(".player-pause").innerHTML = "▶";
            }
        });
        window._cards_for_musics.push(card);
    });
}

function music_player_on_move(e, card) {
    // query player-shift-container
    var player_shift_container = card.querySelector(".player-shift-container");
    var player_player_container = card.querySelector(".player-container");
    var player_image_container = card.querySelector("#image-container");
    var player_hover_line = card.querySelector(".player-hover-line");

    // check if cursor in last 10px
    const containerRect = player_player_container.getBoundingClientRect();
    const bottomThreshold = containerRect.bottom - 15;
    if (
        e.clientY >= bottomThreshold &&
        e.clientY <= containerRect.bottom &&
        e.clientX >= containerRect.left &&
        e.clientX <= containerRect.right
    ) {
        if (card.thisMusicPlayer.lyricsLoaded) {
            // remove transform
            player_shift_container.style.transform = "translateY(0px)";
            // set player z-index to 1
            // player_player_container.style.zIndex = 1;
            // set image z-index to 2
            // player_image_container.style.zIndex = 2;
            // add filter: blur to image container
            player_image_container.style.filter = "blur(5px)";
            // set player_player_container opacity to 0
            player_player_container.style.opacity = 0;
            player_hover_line.style.opacity = 0;
        }

    }

}

function music_player_on_mouseleave(e, card) {
    if (card.thisMusicPlayer.lyricsLoaded) {
        // query player-shift-container
        var player_shift_container = card.querySelector(".player-shift-container");
        var player_player_container = card.querySelector(".player-container");
        var player_image_container = card.querySelector("#image-container");
        var player_hover_line = card.querySelector(".player-hover-line");
        // add transform
        player_shift_container.style.transform = "translateY(100%)";
        // set player z-index to 2
        // player_player_container.style.zIndex = 2;
        // set image z-index to 1
        // player_image_container.style.zIndex = 1;
        // remove filter: blur from image container
        player_image_container.style.filter = "blur(0px)";
        // set player_player_container opacity to 1
        player_player_container.style.opacity = 1;
        player_hover_line.style.opacity = 1;
    }
}

function get_music_info(card) {
    // get music info from card
    var music_info_container = card.querySelector(".music-info-container");
    var returnvar = {}
    returnvar["title"] = music_info_container.getAttribute("data-music-title");
    returnvar["artist"] = music_info_container.getAttribute("data-music-artist");
    returnvar["link"] = music_info_container.getAttribute("data-music-link");
    returnvar["image"] = music_info_container.getAttribute("data-music-image");
    returnvar["lyricLink"] = music_info_container.getAttribute("data-music-lyric");
    returnvar["imageTheme"] = music_info_container.getAttribute("data-image-theme");
    return returnvar;
}

function format_time(seconds) {
    // 处理无效值（负数、NaN、undefined）
    if (isNaN(seconds) || seconds < 0) seconds = 0;

    // 计算分钟和秒数（向下取整确保整数）
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = Math.floor(seconds % 60);
    return `${String(minutes).padStart(2, '0')}:${String(remainingSeconds).padStart(2, '0')}`;
}

class MusicPlayer {
    constructor(music_info) {
        this.music_info = music_info;
        this.player_container = null;
        // 歌词相关
        this.lyrics = [];         // 解析后的歌词数组
        this.lyricsLoaded = false; // 歌词是否已加载
        this.currentLyricIndex = -1; // 当前歌词索引
    }

    init() {
        // 创建音频元素
        this.player_container = document.createElement("audio");
        this.player_container.src = this.music_info.link;
        this.player_container.preload = "metadata";

        // 初始化歌词加载
        if (this.music_info.lyricLink) {
            this.loadLyrics();
        }
        // 初始化音量
        this.setVolume(0.3);
    }

    // 基础控制方法
    play() {
        this.player_container.play();
    }

    pause() {
        this.player_container.pause();
    }

    stop() {
        this.pause();
        this.seek(0);
    }

    seek(time) {
        this.player_container.currentTime = time;
    }

    setVolume(volume) {
        if (volume >= 0 && volume <= 1) {
            this.player_container.volume = volume;
        }
    }

    // 歌词相关方法
    async loadLyrics() {
        try {
            const response = await fetch(this.music_info.lyricLink);
            const text = await response.text();
            // console.log(text);
            this.lyrics = this.parseLyrics(text);
            if (this.lyrics.length > 0) {
                this.lyricsLoaded = true;
            }
        } catch (error) {
            console.error("Failed to load lyrics:", error);
        }
    }

    parseLyrics(rawText) {
        const lines = rawText.split(/\r?\n/); // 处理不同系统的换行符
        const timeRegex = /\[(\d+):(\d+\.?\d*)\]/g; // 改进后的正则表达式

        return lines.flatMap(line => {
            // 获取所有时间标签匹配
            const matches = Array.from(line.matchAll(timeRegex));
            if (matches.length === 0) return [];

            // 提取歌词文本（移除所有时间标签）
            const text = line.replace(timeRegex, '').trim();
            if (!text) return []; // 忽略空文本

            // 为每个时间标签创建歌词对象
            return matches.map(match => {
                const minutes = parseInt(match[1], 10);
                const seconds = parseFloat(match[2]);
                return {
                    time: minutes * 60 + seconds,
                    text: text
                };
            });
        }).sort((a, b) => a.time - b.time); // 按时间排序
    }

    getCurrentLyric() {
        const currentTime = this.player_container.currentTime;
        let foundIndex = -1;

        // 逆向查找第一个时间戳小于当前时间的歌词
        for (let i = this.lyrics.length - 1; i >= 0; i--) {
            if (currentTime >= this.lyrics[i].time) {
                foundIndex = i;
                break;
            }
        }

        // 只有当索引变化时才更新状态
        if (this.currentLyricIndex !== foundIndex) {
            this.currentLyricIndex = foundIndex;
        }

        return foundIndex; // 直接返回当前匹配的索引
    }

    // 事件监听代理方法
    addEventListener(eventName, func) {
        this.player_container.addEventListener(eventName, func);
    }

    // 实用方法
    getCurrentTime() {
        return this.player_container.currentTime;
    }

    getDuration() {
        return this.player_container.duration;
    }

    isPaused() {
        return this.player_container.paused;
    }
}