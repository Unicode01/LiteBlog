if (!window._card_music_player_loaded) {
    window._card_music_player_loaded = true;

    window.addEventListener("DOMContentLoaded", function () {
        window._cards_for_musics = [];
        card_music_player_init();
    });
}

function card_music_player_init() {
    // select music player cards
    var music_player_cards = document.querySelectorAll(".card-music-player");
    i = 0
    music_player_cards.forEach(function (card) {
        card.thisMusicIndex = i;
        i++;
        const music_info = get_music_info(card);

        // 渲染标题和艺术家信息
        card.querySelector(".player-title").textContent = music_info.title || 'Unknown Title';
        card.querySelector(".player-artist").textContent = music_info.artist || 'Unknown Artist';

        // 应用主题
        applyThemeToCard(card, music_info.imageTheme);

        // set background image
        card.querySelector("#image-container").style.backgroundImage = "url('" + music_info["image"] + "')";

        // init music player
        const music_player = new MusicPlayer(music_info, card);
        card.thisMusicPlayer = music_player;
        music_player.init();
        music_player.addEventListener("timeupdate", function () {
            // update progress bar and player-current-time
            const progress_bar = card.querySelector(".player-progress-bar");
            const progress_handle = card.querySelector(".player-progress-handle");
            const progress_bar_width = (music_player.getCurrentTime() / music_player.getDuration()) * 100;
            const progress_bar_current_time = card.querySelector(".player-current-time");
            progress_bar_current_time.textContent = format_time(music_player.getCurrentTime());
            var bg_linear_gradient = "linear-gradient(to right, #999 0%, #999 " + progress_bar_width + "%, #fff " + progress_bar_width + "%, #fff 100%)";
            progress_bar.style.background = bg_linear_gradient;

            // 更新手柄位置
            if (progress_handle) {
                progress_handle.style.left = progress_bar_width + "%";
            }

            // update lyrics in panel
            const currentLyricIndex = music_player.getCurrentLyric();
            if (currentLyricIndex !== -1) {
                updateLyricsPanel(card, currentLyricIndex);
            }
        });
        // add load event listener
        music_player.addEventListener("loadedmetadata", function () {
            // update player-duration
            const progress_bar_duration = card.querySelector(".player-duration");
            progress_bar_duration.textContent = format_time(music_player.getDuration());

            // 初始化歌词面板
            if (music_player.lyricsLoaded) {
                initLyricsPanel(card, music_player);
            }
        });
        // ended事件在MusicPlayer类中处理
        // 这里只需要更新UI和处理自动播放下一曲
        music_player.addEventListener("ended", function () {
            const player_play_button = card.querySelector(".player-pause");
            const playMode = music_player.getPlayMode();

            if (playMode === 'repeat-one') {
                // 单曲循环由MusicPlayer类内部处理，UI保持播放状态
                return;
            }

            // 其他模式：更新UI并播放下一曲
            player_play_button.querySelector(".icon-play").style.display = "block";
            player_play_button.querySelector(".icon-pause").style.display = "none";
            // 播放结束时移除playing类
            card.classList.remove('playing');

            // 延迟一点再播放下一曲，避免UI闪烁
            setTimeout(function () {
                if (playMode === 'sequence' || playMode === 'repeat-all' || playMode === 'shuffle') {
                    playNextTrack(card);
                }
            }, 100);
        });
        // 进度条点击和拖拽
        initProgressBarDrag(card, music_player);

        // 音量控制
        initVolumeControl(card, music_player);

        // 播放模式控制
        initPlayModeControl(card, music_player);

        // 底部控制面板指示器
        initBottomControlIndicators(card);

        // 歌词交互
        initLyricsInteraction(card, music_player);
        const player_play_button = card.querySelector(".player-pause");
        const player_prev_button = card.querySelector(".player-prev");
        const player_next_button = card.querySelector(".player-next");
        // add event listeners
        player_play_button.addEventListener("click", function () {
            if (music_player.isPaused()) {
                music_player.play();
                player_play_button.querySelector(".icon-play").style.display = "none";
                player_play_button.querySelector(".icon-pause").style.display = "block";
                // 添加playing类，隐藏控制组件
                card.classList.add('playing');
            } else {
                music_player.pause();
                player_play_button.querySelector(".icon-play").style.display = "block";
                player_play_button.querySelector(".icon-pause").style.display = "none";
                // 移除playing类，显示控制组件
                card.classList.remove('playing');
            }
        });
        // add prev button event listener
        player_prev_button.addEventListener("click", function () {
            // 优先使用单卡片内切换
            if (music_player.isMusicList && music_player.previousTrack()) {
                // 成功切换，保持播放状态
                // UI状态由switchTrack内部处理，不在这里强制更改
            } else {
                // 跨卡片切换
                playPreviousTrack(card);
            }
        });

        // add next button event listener
        player_next_button.addEventListener("click", function () {
            // 优先使用单卡片内切换
            if (music_player.isMusicList && music_player.nextTrack()) {
                // 成功切换，保持播放状态
                // UI状态由switchTrack内部处理，不在这里强制更改
            } else {
                // 跨卡片切换
                playNextTrack(card);
            }
        });
        // add auto play event listener
        if (music_info.autoPlay) {
            music_player.addEventListener("loadedmetadata", function () {
                music_player.play();
                player_play_button.querySelector(".icon-play").style.display = "none";
                player_play_button.querySelector(".icon-pause").style.display = "block";
                // 自动播放时添加playing类
                card.classList.add('playing');
            });
        }
        window._cards_for_musics.push(card);
    });

    // 所有卡片初始化完成后，再初始化播放列表
    music_player_cards.forEach(function (card) {
        initPlaylistPanel(card);
    });
}

// 旧的hover逻辑已移除，改为click交互

function get_music_info(card) {
    // get music info from card
    var music_info_container = card.querySelector(".music-info-container");

    // 读取原始数据
    const titleStr = music_info_container.getAttribute("data-music-title");
    const artistStr = music_info_container.getAttribute("data-music-artist");
    const linkStr = music_info_container.getAttribute("data-music-link");
    const imageStr = music_info_container.getAttribute("data-music-image");
    const lyricStr = music_info_container.getAttribute("data-music-lyric");
    const themeStr = music_info_container.getAttribute("data-image-theme");

    // 检查是否是数组格式（包含[]分隔符）
    const isArray = titleStr && titleStr.includes('[') && titleStr.includes(']');

    if (isArray) {
        // 解析数组格式：[歌曲1]|[歌曲2]|[歌曲3]
        const parseArray = (str) => {
            if (!str) return [];
            return str.match(/\[([^\]]*)\]/g)
                ?.map(item => item.slice(1, -1).trim())
                .filter(item => item.length > 0) || [];
        };

        const titles = parseArray(titleStr);
        const artists = parseArray(artistStr);
        const links = parseArray(linkStr);
        const images = parseArray(imageStr);
        const lyrics = parseArray(lyricStr);
        const themes = parseArray(themeStr);

        // 构建歌曲列表
        const musicList = [];
        const maxLength = Math.max(titles.length, artists.length, links.length);

        for (let i = 0; i < maxLength; i++) {
            musicList.push({
                title: titles[i] || 'Unknown Title',
                artist: artists[i] || 'Unknown Artist',
                link: links[i] || '',
                image: images[i] || images[0] || '',
                lyricLink: lyrics[i] || '',
                imageTheme: themes[i] || themes[0] || 'light'  // 每首歌的主题
            });
        }

        return {
            isMusicList: true,
            musicList: musicList,
            currentIndex: 0,
            imageTheme: musicList[0].imageTheme,  // 使用第一首歌的主题
            autoPlay: music_info_container.getAttribute("data-auto-play") === "true",
            // 当前显示的歌曲信息（第一首）
            title: musicList[0].title,
            artist: musicList[0].artist,
            link: musicList[0].link,
            image: musicList[0].image,
            lyricLink: musicList[0].lyricLink
        };
    }

    // 单首歌曲格式（向后兼容）
    var returnvar = {}
    returnvar["title"] = titleStr;
    returnvar["artist"] = artistStr;
    returnvar["link"] = linkStr;
    returnvar["image"] = imageStr;
    returnvar["lyricLink"] = lyricStr;
    returnvar["imageTheme"] = music_info_container.getAttribute("data-image-theme");
    returnvar["autoPlay"] = music_info_container.getAttribute("data-auto-play") === "true";
    returnvar["isMusicList"] = false;
    returnvar["currentIndex"] = 0;
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

// 应用主题样式到卡片
function applyThemeToCard(card, theme) {
    if (theme === "dark") {
        // dark主题：白色文字和灰色次要文字
        card.querySelector(".player-title").style.color = "white";
        card.querySelector(".player-artist").style.color = "#999";
        card.querySelector(".player-duration").style.color = "white";
        card.querySelector(".player-current-time").style.color = "white";
        card.querySelector(".player-pause").style.color = "white";
        card.querySelector(".player-prev").style.color = "#ccc";
        card.querySelector(".player-next").style.color = "#ccc";
        card.style.boxShadow = "0 0 10px #3A3A3A";

        // dark主题：indicator-line使用白色半透明
        const indicatorLines = card.querySelectorAll(".indicator-line");
        indicatorLines.forEach(line => {
            line.style.background = "rgba(255, 255, 255, 0.5)";
            line.style.boxShadow = "0 0 8px rgba(255, 255, 255, 0.3)";
        });
    } else {
        // light主题：恢复默认样式
        card.querySelector(".player-title").style.color = "";
        card.querySelector(".player-artist").style.color = "";
        card.querySelector(".player-duration").style.color = "";
        card.querySelector(".player-current-time").style.color = "";
        card.querySelector(".player-pause").style.color = "";
        card.querySelector(".player-prev").style.color = "";
        card.querySelector(".player-next").style.color = "";
        card.style.boxShadow = "";

        // light主题：indicator-line使用灰色半透明
        const indicatorLines = card.querySelectorAll(".indicator-line");
        indicatorLines.forEach(line => {
            line.style.background = "rgba(100, 100, 100, 0.5)";
            line.style.boxShadow = "0 0 8px rgba(100, 100, 100, 0.3)";
        });
    }
}

// 同步音量到所有卡片
function syncVolumeToAllCards(volume) {
    if (!window._cards_for_musics) return;

    const volumePercent = Math.round(volume * 100);

    window._cards_for_musics.forEach(card => {
        // 更新音量滑块和显示值
        const volumeSlider = card.querySelector("#volume-slider");
        const volumeValue = card.querySelector("#volume-value");

        if (volumeSlider && volumeValue) {
            volumeSlider.value = volumePercent;
            volumeValue.textContent = volumePercent + "%";
        }

        // 更新实际音量
        if (card.thisMusicPlayer) {
            card.thisMusicPlayer.player_container.volume = volume;
        }
    });
}

// 同步播放模式到所有卡片
function syncPlayModeToAllCards(mode) {
    if (!window._cards_for_musics) return;

    window._cards_for_musics.forEach(card => {
        // 更新播放模式按钮状态
        const modeButtons = {
            'sequence': card.querySelector("#mode-sequence"),
            'repeat-one': card.querySelector("#mode-repeat-one"),
            'repeat-all': card.querySelector("#mode-repeat-all"),
            'shuffle': card.querySelector("#mode-shuffle")
        };

        Object.keys(modeButtons).forEach(buttonMode => {
            const btn = modeButtons[buttonMode];
            if (!btn) return;

            if (buttonMode === mode) {
                btn.classList.add('active');
            } else {
                btn.classList.remove('active');
            }
        });

        // 更新实际播放模式
        if (card.thisMusicPlayer) {
            card.thisMusicPlayer.playMode = mode;
        }
    });
}

class MusicPlayer {
    constructor(music_info, card) {
        this.music_info = music_info;
        this.card = card;
        this.player_container = null;
        // 歌词相关
        this.lyrics = [];         // 解析后的歌词数组
        this.lyricsLoaded = false; // 歌词是否已加载
        this.currentLyricIndex = -1; // 当前歌词索引
        // 播放模式：'sequence', 'repeat-one', 'repeat-all', 'shuffle'
        this.playMode = 'repeat-one';
        // 歌词自动滚动控制
        this.autoScrollLyrics = true;  // 是否自动滚动歌词
        this.userScrollTimer = null;    // 用户滚动计时器

        // 多首歌曲支持
        this.isMusicList = music_info.isMusicList || false;
        this.currentTrackIndex = music_info.currentIndex || 0;
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
        // 初始化音量（从localStorage读取，默认0.3）
        const savedVolume = localStorage.getItem('music_player_volume');
        this.setVolume(savedVolume !== null ? parseFloat(savedVolume) : 0.3);

        // 初始化播放模式（从localStorage读取，默认repeat-one）
        const savedMode = localStorage.getItem('music_player_mode');
        this.playMode = savedMode || 'repeat-one';

        // 添加ended事件监听（用于播放模式）
        this.player_container.addEventListener('ended', () => {
            this.handleEnded();
        });
    }

    // 切换到指定曲目（单卡片多歌曲）
    // 切换到指定曲目（单卡片多歌曲）
    // autoPlay: 可选参数，强制指定是否自动播放（用于歌曲结束后自动播放下一首）
    switchTrack(index, autoPlay = null) {
        if (!this.isMusicList) return false;
        if (index < 0 || index >= this.music_info.musicList.length) return false;

        const track = this.music_info.musicList[index];
        // 如果没有指定autoPlay，则根据当前播放状态决定
        const shouldPlay = autoPlay !== null ? autoPlay : !this.isPaused();

        // 停止当前播放
        this.stop();

        // 更新索引
        this.currentTrackIndex = index;

        // 更新音频源
        this.player_container.src = track.link;

        // 更新UI
        if (this.card) {
            this.card.querySelector(".player-title").textContent = track.title;
            this.card.querySelector(".player-artist").textContent = track.artist;
            this.card.querySelector("#image-container").style.backgroundImage = "url('" + track.image + "')";

            // 应用该曲目的主题
            if (track.imageTheme) {
                applyThemeToCard(this.card, track.imageTheme);
            }
        }

        // 更新歌词
        this.lyrics = [];
        this.lyricsLoaded = false;
        if (track.lyricLink) {
            this.music_info.lyricLink = track.lyricLink;
            this.loadLyrics().then(() => {
                if (this.lyricsLoaded && this.card) {
                    initLyricsPanel(this.card, this);
                }
            });
        }

        // 根据播放状态决定是否继续播放
        if (shouldPlay) {
            this.player_container.addEventListener('loadedmetadata', () => {
                this.play();
                // 保持播放图标状态
                if (this.card) {
                    const btn = this.card.querySelector(".player-pause");
                    btn.querySelector(".icon-play").style.display = "none";
                    btn.querySelector(".icon-pause").style.display = "block";
                    this.card.classList.add('playing');
                }
            }, { once: true });
        } else {
            // 如果之前是暂停状态，保持暂停图标
            if (this.card) {
                const btn = this.card.querySelector(".player-pause");
                btn.querySelector(".icon-play").style.display = "block";
                btn.querySelector(".icon-pause").style.display = "none";
                this.card.classList.remove('playing');
            }
        }

        return true;
    }

    // 播放下一首（单卡片内）
    nextTrack() {
        if (!this.isMusicList) return false;

        let nextIndex = this.currentTrackIndex + 1;
        if (nextIndex >= this.music_info.musicList.length) {
            if (this.playMode === 'repeat-all') {
                nextIndex = 0;
            } else {
                return false; // 到达列表末尾
            }
        }
        return this.switchTrack(nextIndex);
    }

    // 播放上一首（单卡片内）
    previousTrack() {
        if (!this.isMusicList) return false;

        let prevIndex = this.currentTrackIndex - 1;
        if (prevIndex < 0) {
            if (this.playMode === 'repeat-all') {
                prevIndex = this.music_info.musicList.length - 1;
            } else {
                return false; // 到达列表开头
            }
        }
        return this.switchTrack(prevIndex);
    }

    // 处理播放结束
    handleEnded() {
        switch (this.playMode) {
            case 'repeat-one':
                // 单曲循环
                this.seek(0);
                this.play();
                break;
            case 'repeat-all':
            case 'sequence':
                // 如果是多曲目模式，尝试播放下一首（自动播放）
                if (this.isMusicList) {
                    // 传入autoPlay=true确保歌曲结束后自动播放下一首
                    const nextIndex = this.currentTrackIndex + 1;
                    if (nextIndex < this.music_info.musicList.length || this.playMode === 'repeat-all') {
                        const targetIndex = nextIndex >= this.music_info.musicList.length ? 0 : nextIndex;
                        this.switchTrack(targetIndex, true);
                    }
                }
                // 列表循环或顺序播放
                // 这部分逻辑在card级别处理
                break;
            case 'shuffle':
                // 如果是多曲目模式，随机播放（自动播放）
                if (this.isMusicList) {
                    const randomIndex = Math.floor(Math.random() * this.music_info.musicList.length);
                    // 传入autoPlay=true确保歌曲结束后自动播放下一首
                    this.switchTrack(randomIndex, true);
                }
                // 随机播放
                // 这部分逻辑在card级别处理
                break;
        }
    }

    // 设置播放模式
    setPlayMode(mode) {
        this.playMode = mode;
        // 保存到localStorage
        localStorage.setItem('music_player_mode', mode);

        // 同步所有其他播放卡片的播放模式
        syncPlayModeToAllCards(mode);
    }

    // 获取播放模式
    getPlayMode() {
        return this.playMode;
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
            // 保存到localStorage
            localStorage.setItem('music_player_volume', volume);

            // 同步所有其他播放卡片的音量设置
            syncVolumeToAllCards(volume);
        }
    }

    // 歌词相关方法
    async loadLyrics() {
        try {
            const lyricLink = this.music_info.lyricLink;
            if (!lyricLink) return;

            const response = await fetch(lyricLink);
            const text = await response.text();
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

// ==================== 新功能实现 ====================

// 初始化进度条拖拽
function initProgressBarDrag(card, music_player) {
    const progress_bar = card.querySelector(".player-progress-bar");
    const progress_handle = card.querySelector(".player-progress-handle");
    let isDragging = false;
    let wasPlaying = false;

    // 禁用浏览器默认的拖放行为
    const preventDrag = function (e) {
        e.preventDefault();
        return false;
    };

    if (progress_bar) {
        progress_bar.setAttribute('draggable', 'false');
        progress_bar.addEventListener('dragstart', preventDrag);
        progress_bar.addEventListener('drag', preventDrag);
    }

    if (progress_handle) {
        progress_handle.setAttribute('draggable', 'false');
        progress_handle.addEventListener('dragstart', preventDrag);
        progress_handle.addEventListener('drag', preventDrag);
    }

    // 计算并设置进度的函数
    const setProgress = function (clientX) {
        const rect = progress_bar.getBoundingClientRect();
        let clickX = clientX - rect.left;
        // 限制在有效范围内
        clickX = Math.max(0, Math.min(clickX, rect.width));
        const progress = clickX / rect.width;
        const duration = music_player.getDuration();

        // 检查duration是否有效
        if (isNaN(duration) || duration <= 0) {
            return;
        }

        const seekTime = progress * duration;
        music_player.seek(seekTime);

        // 立即更新手柄位置（拖拽时实时反馈）
        if (progress_handle) {
            progress_handle.style.left = (progress * 100) + "%";
        }
    };

    // 开始拖拽的函数
    const startDrag = function (e) {
        isDragging = true;
        wasPlaying = !music_player.isPaused();

        // 拖拽时暂停播放以获得更好的体验
        if (wasPlaying) {
            music_player.pause();
        }

        // 添加拖拽中的视觉样式
        progress_bar.classList.add('dragging');
        document.body.style.userSelect = 'none'; // 防止选中文本

        // 强制设置元素尺寸和透明度，防止浏览器缩放和透明度变化
        if (progress_handle) {
            progress_handle.style.width = '14px';
            progress_handle.style.height = '14px';
            progress_handle.style.transform = 'translate(-50%, -50%) scale(1)';
            progress_handle.style.opacity = '1';  // 强制设置为完全不透明
            progress_handle.style.transition = 'none';  // 禁用所有transition
        }
        progress_bar.style.height = '5px';
        progress_bar.style.opacity = '1';  // 强制设置为完全不透明
        progress_bar.style.transition = 'none';  // 禁用所有transition

        e.preventDefault();
        e.stopPropagation();
    };

    // 拖拽中的函数
    const doDrag = function (e) {
        if (!isDragging) return;
        setProgress(e.clientX);
        e.preventDefault();
    };

    // 结束拖拽的函数
    const endDrag = function (e) {
        if (!isDragging) return;

        isDragging = false;

        // 移除拖拽样式
        progress_bar.classList.remove('dragging');
        document.body.style.userSelect = '';

        // 清除强制设置的样式，恢复CSS控制
        if (progress_handle) {
            progress_handle.style.opacity = '';  // 清除，让CSS控制（hover显示）
            progress_handle.style.transition = '';  // 恢复transition
        }
        progress_bar.style.opacity = '';
        progress_bar.style.transition = '';

        // 如果之前在播放，恢复播放
        if (wasPlaying) {
            music_player.play();
            const player_play_button = card.querySelector(".player-pause");
            if (player_play_button) {
                player_play_button.querySelector(".icon-play").style.display = "none";
                player_play_button.querySelector(".icon-pause").style.display = "block";
            }
        }
    };

    // 点击进度条直接跳转
    progress_bar.addEventListener("mousedown", function (e) {
        // 如果点击的是手柄，则启动拖拽；否则直接跳转
        if (e.target === progress_handle || progress_handle.contains(e.target)) {
            startDrag(e);
        } else {
            // 直接点击进度条跳转
            setProgress(e.clientX);
            // 如果正在播放，保持播放状态
            if (!music_player.isPaused()) {
                // 不需要额外操作
            }
        }
    });

    // 在进度条上按下鼠标时也可以拖拽
    progress_bar.addEventListener("mousedown", function (e) {
        if (e.button !== 0) return; // 只响应左键
        startDrag(e);
        setProgress(e.clientX);
    });

    // 全局鼠标移动事件
    document.addEventListener("mousemove", doDrag);

    // 全局鼠标释放事件
    document.addEventListener("mouseup", endDrag);

    // 触摸事件支持（移动端）
    progress_bar.addEventListener("touchstart", function (e) {
        if (e.touches.length !== 1) return;
        const touch = e.touches[0];
        startDrag({ clientX: touch.clientX, preventDefault: () => e.preventDefault(), stopPropagation: () => e.stopPropagation() });
        setProgress(touch.clientX);
    }, { passive: false });

    document.addEventListener("touchmove", function (e) {
        if (!isDragging || e.touches.length !== 1) return;
        const touch = e.touches[0];
        setProgress(touch.clientX);
        e.preventDefault();
    }, { passive: false });

    document.addEventListener("touchend", function (e) {
        if (!isDragging) return;
        endDrag({ preventDefault: () => { } });
    });
}

// 初始化音量控制
function initVolumeControl(card, music_player) {
    const volumeSlider = card.querySelector("#volume-slider");
    const volumeValue = card.querySelector("#volume-value");

    if (!volumeSlider || !volumeValue) return;

    // 从localStorage读取并设置初始值
    const savedVolume = localStorage.getItem('music_player_volume');
    const volumePercent = savedVolume !== null ? Math.round(parseFloat(savedVolume) * 100) : 30;
    volumeSlider.value = volumePercent;
    volumeValue.textContent = volumePercent + "%";

    volumeSlider.addEventListener("input", function () {
        const volume = volumeSlider.value / 100;
        music_player.setVolume(volume);
        volumeValue.textContent = volumeSlider.value + "%";
    });
}

// 初始化播放模式控制
function initPlayModeControl(card, music_player) {
    const modeButtons = {
        'sequence': card.querySelector("#mode-sequence"),
        'repeat-one': card.querySelector("#mode-repeat-one"),
        'repeat-all': card.querySelector("#mode-repeat-all"),
        'shuffle': card.querySelector("#mode-shuffle")
    };

    // 从localStorage读取并设置初始激活状态
    const savedMode = music_player.getPlayMode();
    Object.keys(modeButtons).forEach(mode => {
        const btn = modeButtons[mode];
        if (!btn) return;

        // 设置初始激活状态
        if (mode === savedMode) {
            btn.classList.add('active');
        } else {
            btn.classList.remove('active');
        }

        btn.addEventListener("click", function () {
            // 移除所有active类
            Object.values(modeButtons).forEach(b => b && b.classList.remove('active'));
            // 添加当前active
            btn.classList.add('active');
            // 设置播放模式
            music_player.setPlayMode(mode);
        });
    });
}

// 初始化底部控制指示器（点击打开/关闭）
function initBottomControlIndicators(card) {
    const leftIndicator = card.querySelector("#left-control-indicator");
    const rightIndicator = card.querySelector("#right-control-indicator");
    const leftPanel = card.querySelector("#left-control-panel");
    const rightPanel = card.querySelector("#right-control-panel");

    if (leftIndicator && leftPanel) {
        leftIndicator.addEventListener("click", function (e) {
            e.stopPropagation();
            leftPanel.classList.toggle('active');
            // 关闭右侧面板
            if (rightPanel) rightPanel.classList.remove('active');
        });
    }

    if (rightIndicator && rightPanel) {
        rightIndicator.addEventListener("click", function (e) {
            e.stopPropagation();
            rightPanel.classList.toggle('active');
            // 关闭左侧面板
            if (leftPanel) leftPanel.classList.remove('active');
        });
    }

    // 点击卡片其他区域关闭面板
    card.addEventListener("click", function (e) {
        if (!e.target.closest('.control-panel') && !e.target.closest('.control-indicator')) {
            if (leftPanel) leftPanel.classList.remove('active');
            if (rightPanel) rightPanel.classList.remove('active');
        }
    });
}

// 初始化播放列表面板
function initPlaylistPanel(card) {
    const playlistContainer = card.querySelector("#playlist-container");
    if (!playlistContainer) return;

    const music_player = card.thisMusicPlayer;

    // 如果是单卡片多歌曲模式
    if (music_player.isMusicList) {
        const musicList = music_player.music_info.musicList;
        musicList.forEach((track, index) => {
            const item = document.createElement('div');
            item.className = 'playlist-item';
            if (index === music_player.currentTrackIndex) {
                item.classList.add('playing');
            }

            item.innerHTML = `
                <div class="playlist-item-title">${track.title}</div>
                <div class="playlist-item-artist">${track.artist}</div>
            `;

            item.addEventListener('click', function () {
                // 切换歌曲，保持当前播放状态
                music_player.switchTrack(index);

                // 更新播放列表样式
                playlistContainer.querySelectorAll('.playlist-item').forEach(i => i.classList.remove('playing'));
                item.classList.add('playing');
            });

            playlistContainer.appendChild(item);
        });

        // 根据歌曲数量决定滚动样式
        checkPlaylistScroll(playlistContainer, musicList.length);
        return;
    }

    // 多卡片模式：构建所有卡片的播放列表
    window._cards_for_musics.forEach((musicCard, index) => {
        const musicInfo = musicCard.thisMusicPlayer.music_info;
        const item = document.createElement('div');
        item.className = 'playlist-item';
        if (musicCard === card) {
            item.classList.add('playing');
        }

        item.innerHTML = `
            <div class="playlist-item-title">${musicInfo.title}</div>
            <div class="playlist-item-artist">${musicInfo.artist}</div>
        `;

        item.addEventListener('click', function () {
            // 停止当前播放
            window._cards_for_musics.forEach(c => {
                if (c.thisMusicPlayer && !c.thisMusicPlayer.isPaused()) {
                    c.thisMusicPlayer.stop();
                    const btn = c.querySelector(".player-pause");
                    btn.querySelector(".icon-play").style.display = "block";
                    btn.querySelector(".icon-pause").style.display = "none";
                    // 移除playing类
                    c.classList.remove('playing');
                }
            });

            // 播放选中的音乐
            musicCard.thisMusicPlayer.play();
            const btn = musicCard.querySelector(".player-pause");
            btn.querySelector(".icon-play").style.display = "none";
            btn.querySelector(".icon-pause").style.display = "block";
            // 添加playing类
            musicCard.classList.add('playing');

            // 更新播放列表样式
            playlistContainer.querySelectorAll('.playlist-item').forEach(i => i.classList.remove('playing'));
            item.classList.add('playing');
        });

        playlistContainer.appendChild(item);
    });

    // 根据歌曲数量决定滚动样式
    checkPlaylistScroll(playlistContainer, window._cards_for_musics.length);
}

// 检查并设置播放列表滚动
function checkPlaylistScroll(playlistContainer, itemCount) {
    // 如果歌曲数量 >= 2，启用滚动防止溢出
    if (itemCount >= 2) {
        playlistContainer.style.maxHeight = "200px";
        playlistContainer.style.overflowY = "auto";
    } else {
        // 只有1首歌时不需要固定高度
        playlistContainer.style.maxHeight = "none";
        playlistContainer.style.overflowY = "visible";
    }
}

// 初始化歌词面板
function initLyricsPanel(card, music_player) {
    const lyricsPanel = card.querySelector("#lyrics-panel");
    const lyricsContainer = card.querySelector("#lyrics-scroll-container");

    if (!lyricsPanel || !lyricsContainer || !music_player.lyricsLoaded) return;

    // 清空容器
    lyricsContainer.innerHTML = '';

    // 添加所有歌词行
    music_player.lyrics.forEach((lyric, index) => {
        const lyricLine = document.createElement('div');
        lyricLine.className = 'lyric-line';
        lyricLine.textContent = lyric.text;
        lyricLine.dataset.index = index;
        lyricLine.dataset.time = lyric.time;

        // 点击歌词跳转
        lyricLine.addEventListener('click', function () {
            music_player.seek(lyric.time);
            if (music_player.isPaused()) {
                music_player.play();
                const btn = card.querySelector(".player-pause");
                btn.querySelector(".icon-play").style.display = "none";
                btn.querySelector(".icon-pause").style.display = "block";
            }
        });

        lyricsContainer.appendChild(lyricLine);
    });
}

// 更新歌词面板高亮
function updateLyricsPanel(card, currentIndex) {
    const lyricsContainer = card.querySelector("#lyrics-scroll-container");
    const lyricsPanel = card.querySelector("#lyrics-panel");
    const musicPlayer = card.thisMusicPlayer;
    if (!lyricsContainer || !musicPlayer) return;

    const lyricLines = lyricsContainer.querySelectorAll('.lyric-line');
    lyricLines.forEach((line, index) => {
        if (index === currentIndex) {
            line.classList.add('active');
            // 只有当歌词面板打开、用户允许自动滚动时才滚动
            if (lyricsPanel && lyricsPanel.classList.contains('active') && musicPlayer.autoScrollLyrics) {
                const lineTop = line.offsetTop;
                const containerHeight = lyricsContainer.clientHeight;
                const lineHeight = line.offsetHeight;
                const scrollPosition = lineTop - (containerHeight / 2) + (lineHeight / 2);
                lyricsContainer.scrollTo({
                    top: scrollPosition,
                    behavior: 'smooth'
                });
            }
        } else {
            line.classList.remove('active');
        }
    });
}

// 初始化歌词交互（点击中下指示条打开/关闭歌词面板）
function initLyricsInteraction(card, music_player) {
    const centerIndicator = card.querySelector("#center-control-indicator");
    const lyricsPanel = card.querySelector("#lyrics-panel");
    const lyricsCloseBtn = card.querySelector(".lyrics-close-btn");
    const lyricsScrollContainer = card.querySelector("#lyrics-scroll-container");

    if (!centerIndicator || !lyricsPanel) return;

    // 点击中下指示条切换歌词面板
    centerIndicator.addEventListener("click", function (e) {
        e.stopPropagation();
        // 只有当加载了歌词时才显示
        if (music_player.lyricsLoaded) {
            const isOpening = !lyricsPanel.classList.contains('active');
            lyricsPanel.classList.toggle('active');

            // 打开歌词面板时恢复自动滚动
            if (isOpening) {
                music_player.autoScrollLyrics = true;
            }
        }
    });

    // 关闭按钮
    if (lyricsCloseBtn) {
        lyricsCloseBtn.addEventListener('click', function (e) {
            e.stopPropagation();
            lyricsPanel.classList.remove('active');
            // 清除用户滚动定时器
            if (music_player.userScrollTimer) {
                clearTimeout(music_player.userScrollTimer);
                music_player.userScrollTimer = null;
            }
            // 恢复自动滚动状态
            music_player.autoScrollLyrics = true;
        });
    }

    // 在歌词面板本身阻止滚动事件冒泡
    if (lyricsPanel) {
        lyricsPanel.addEventListener('wheel', function (e) {
            e.stopPropagation();
        }, { passive: true });
    }

    // 防止歌词滚动时触发页面滚动 + 智能滚动控制
    if (lyricsScrollContainer) {
        const autoScrollIndicator = card.querySelector("#auto-scroll-indicator");

        // 用户滚动歌词时暂停自动滚动的处理函数
        const handleUserScroll = function () {
            // 标记为用户手动滚动，暂停自动滚动
            music_player.autoScrollLyrics = false;

            // 显示暂停提示
            if (autoScrollIndicator) {
                autoScrollIndicator.classList.add('show', 'paused');
            }

            // 清除之前的计时器
            if (music_player.userScrollTimer) {
                clearTimeout(music_player.userScrollTimer);
            }

            // 3秒后恢复自动滚动
            music_player.userScrollTimer = setTimeout(function () {
                music_player.autoScrollLyrics = true;

                // 隐藏提示
                if (autoScrollIndicator) {
                    autoScrollIndicator.classList.remove('show', 'paused');
                }
            }, 3000);
        };

        // 使用wheel事件阻止滚动传播 + 检测用户滚动
        lyricsScrollContainer.addEventListener('wheel', function (e) {
            e.stopPropagation();

            // 标记用户正在滚动
            handleUserScroll();

            const scrollTop = lyricsScrollContainer.scrollTop;
            const scrollHeight = lyricsScrollContainer.scrollHeight;
            const clientHeight = lyricsScrollContainer.clientHeight;
            const deltaY = e.deltaY;

            // 如果滚动到顶部且继续向上滚，或滚动到底部且继续向下滚，则阻止事件
            if ((scrollTop === 0 && deltaY < 0) ||
                (scrollTop + clientHeight >= scrollHeight && deltaY > 0)) {
                e.preventDefault();
            }
        }, { passive: false });

        // 同时处理触摸滚动（移动端）
        lyricsScrollContainer.addEventListener('touchmove', function (e) {
            e.stopPropagation();
            // 标记用户正在滚动
            handleUserScroll();
        }, { passive: true });

        // 检测用户拖动滚动条
        let isUserScrolling = false;
        lyricsScrollContainer.addEventListener('scroll', function (e) {
            e.stopPropagation();

            // 如果不是自动滚动触发的，则是用户操作
            if (!music_player.autoScrollLyrics || isUserScrolling) {
                return;
            }

            // 检测是否是用户拖动滚动条
            if (e.isTrusted && !music_player.autoScrollLyrics) {
                handleUserScroll();
            }
        }, { passive: true });

        // 检测鼠标按下（可能是拖动滚动条）
        lyricsScrollContainer.addEventListener('mousedown', function () {
            isUserScrolling = true;
            handleUserScroll();
        });

        // 鼠标释放
        document.addEventListener('mouseup', function () {
            isUserScrolling = false;
        });
    }
}

// 播放上一曲
function playPreviousTrack(card) {
    const playMode = card.thisMusicPlayer.getPlayMode();
    let newIndex;

    if (playMode === 'shuffle') {
        // 随机模式
        newIndex = Math.floor(Math.random() * window._cards_for_musics.length);
    } else {
        // 顺序/循环模式
        newIndex = card.thisMusicIndex - 1;
        if (newIndex < 0) {
            if (playMode === 'repeat-all') {
                newIndex = window._cards_for_musics.length - 1; // 循环到最后一首
            } else {
                // sequence模式，到第一首就停止
                return;
            }
        }
    }

    switchToTrack(card, newIndex);
}

// 播放下一曲
function playNextTrack(card) {
    const playMode = card.thisMusicPlayer.getPlayMode();
    let newIndex;

    if (playMode === 'shuffle') {
        // 随机模式
        newIndex = Math.floor(Math.random() * window._cards_for_musics.length);
    } else {
        // 顺序/循环模式
        newIndex = card.thisMusicIndex + 1;
        if (newIndex >= window._cards_for_musics.length) {
            if (playMode === 'repeat-all') {
                newIndex = 0; // 循环到第一首
            } else {
                // sequence模式，到最后一首就停止
                return;
            }
        }
    }

    switchToTrack(card, newIndex);
}

// 切换到指定曲目
function switchToTrack(fromCard, toIndex) {
    if (toIndex < 0 || toIndex >= window._cards_for_musics.length) return;

    const toCard = window._cards_for_musics[toIndex];

    // 停止当前播放
    if (fromCard && fromCard.thisMusicPlayer) {
        fromCard.thisMusicPlayer.stop();
        const fromButton = fromCard.querySelector(".player-pause");
        fromButton.querySelector(".icon-play").style.display = "block";
        fromButton.querySelector(".icon-pause").style.display = "none";
        // 移除playing类
        fromCard.classList.remove('playing');
    }

    // 播放新曲目
    toCard.thisMusicPlayer.play();
    const toButton = toCard.querySelector(".player-pause");
    toButton.querySelector(".icon-play").style.display = "none";
    toButton.querySelector(".icon-pause").style.display = "block";
    // 添加playing类
    toCard.classList.add('playing');

    // 更新所有播放列表的playing状态
    updateAllPlaylistsUI(toIndex);
}

// 更新所有卡片的播放列表UI
function updateAllPlaylistsUI(playingIndex) {
    window._cards_for_musics.forEach(card => {
        const playlistContainer = card.querySelector("#playlist-container");
        if (playlistContainer) {
            const items = playlistContainer.querySelectorAll('.playlist-item');
            items.forEach((item, index) => {
                if (index === playingIndex) {
                    item.classList.add('playing');
                } else {
                    item.classList.remove('playing');
                }
            });
        }
    });
}