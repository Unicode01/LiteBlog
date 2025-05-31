if (!window._card_template_blogger_info_loaded) {
    window._card_template_blogger_info_loaded = true;

    window.addEventListener('DOMContentLoaded', function () {
        init_card_blogger_info();
    });

}

function getBloggerInfo(container) {
    const info_container = container.querySelector('.blogger-info-container');
    const blogger_name = info_container.getAttribute('data-blogger-name');
    const blogger_avatar = info_container.getAttribute('data-blogger-avatar');
    const blogger_bio = info_container.getAttribute('data-blogger-bio');
    const blogger_contact_info = info_container.getAttribute('data-blogger-contact-info');
    console.log(blogger_name, blogger_avatar, blogger_bio, blogger_contact_info);
    // tackle the contact info
    // structure: [type](link)|[type](link)|...
    function parseLinkData(input) {
        if (!input) return [];

        const segments = input.split('|').filter(segment => segment.trim() !== '');
        const regex = /\[(.*?)\]\((.*?)\)/;

        return segments.map(segment => {
            const match = segment.match(regex);
            if (match) {
                return {
                    type: match[1].trim(),
                    link: match[2].trim()
                };
            }
            return null;
        }).filter(item => item !== null);
    }
    const contact_info = parseLinkData(blogger_contact_info);
    console.log(contact_info);
    return {
        name: blogger_name,
        avatar: blogger_avatar,
        bio: blogger_bio,
        contact_info: contact_info
    }
}

function init_card_blogger_info() {
    const all_blogger_info_containers = document.querySelectorAll('.card-blogger-info-container');
    const type2icon = {
        "github": "/img/github-mark.svg",
        "bilibili": "/img/bilibili-logo.svg",
        "telegram": "/img/telegram-logo.svg",
        "x": "/img/x-logo.svg",
        "youtube": "/img/youtube-logo.svg",
        "facebook": "/img/facebook-logo.svg",
        "instagram": "/img/instagram-logo.svg",
        "email": "/img/email-icon.svg",
        "steam": "/img/steam-logo.svg",
    }
    all_blogger_info_containers.forEach(function (blogger_info_container) {
        const blogger_info = getBloggerInfo(blogger_info_container);
        const blogger_info_container_inner = blogger_info_container.querySelector('.blogger-info-container');
        const blogger_contact_container = blogger_info_container.querySelector('.card-blogger-contacter');
        blogger_info.contact_info.forEach(function (contact) {
            const icon = type2icon[contact.type] || "/img/link-icon.svg";
            const link_container = document.createElement('a');
            link_container.href = contact.link;
            link_container.classList.add('card-blogger-contact-icon');
            link_container.target = '_blank';
            const icon_container = document.createElement('img');
            icon_container.src = icon;
            icon_container.alt = contact.type;
            link_container.appendChild(icon_container);
            blogger_contact_container.appendChild(link_container);
            // if need to invert style, add theme switch listener, set inverting style
            if (contact.type === 'github' || contact.type === 'telegram' || contact.type === 'bilibili' || contact.type === 'youtube' || contact.type === 'facebook' || contact.type === 'instagram' || contact.type === 'email' || contact.type === 'steam') {
                if (GetTheme() === 'dark') {
                    link_container.style.filter = 'invert(100%)';
                }
                addThemeSwitchBroadcastListener(function(theme){
                    if (theme === 'dark') {
                        link_container.style.filter = 'invert(100%)';
                    } else {
                        link_container.style.filter = 'none';
                    }
                })
            } else if (contact.type === "x") {
                if (GetTheme() === 'light') {
                    link_container.style.filter = 'invert(100%)';
                }
                addThemeSwitchBroadcastListener(function(theme){
                    if (theme === 'dark') {
                        link_container.style.filter = 'none';
                    } else {
                        link_container.style.filter = 'invert(100%)';
                    }
                })
            }
            
        });

        // done, remove the info container
        blogger_info_container_inner.remove()
    });
}