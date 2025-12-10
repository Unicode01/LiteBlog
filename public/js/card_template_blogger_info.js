if (!window._card_template_blogger_info_loaded) {
    window._card_template_blogger_info_loaded = true;

    window.addEventListener('DOMContentLoaded', function () {
        init_card_blogger_info();
    });
}

function getBloggerInfo(container) {
    const info_container = container.querySelector('.blogger-info-container');
    const blogger_name = info_container.dataset.bloggerName || '';
    const blogger_avatar = info_container.dataset.bloggerAvatar || '';
    const blogger_bio = info_container.dataset.bloggerBio || '';
    const blogger_contact_info = info_container.dataset.bloggerContactInfo || '';

    // parse contact info
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

        // avatar is pre-rendered in HTML template, no need to set here

        // render name
        const nameContainer = blogger_info_container.querySelector('.card-blogger-name');
        if (nameContainer) {
            // insert name text before bio div
            const bioDiv = nameContainer.querySelector('.card-blogger-bio');
            if (bioDiv) {
                // create text node for name
                const nameText = document.createTextNode(blogger_info.name);
                nameContainer.insertBefore(nameText, bioDiv);
            }
        }

        // render bio
        const bioContainer = blogger_info_container.querySelector('.card-blogger-bio');
        if (bioContainer) {
            bioContainer.textContent = blogger_info.bio;
        }

        // render contact info
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

            // handle theme-based icon inversion
            if (contact.type === 'github' || contact.type === 'telegram' || contact.type === 'bilibili' ||
                contact.type === 'youtube' || contact.type === 'facebook' || contact.type === 'instagram' ||
                contact.type === 'email' || contact.type === 'steam') {
                if (GetTheme() === 'dark') {
                    link_container.style.filter = 'invert(100%)';
                }
                addThemeSwitchBroadcastListener(function (theme) {
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
                addThemeSwitchBroadcastListener(function (theme) {
                    if (theme === 'dark') {
                        link_container.style.filter = 'none';
                    } else {
                        link_container.style.filter = 'invert(100%)';
                    }
                })
            }
        });

        // done, remove the info container
        const blogger_info_container_inner = blogger_info_container.querySelector('.blogger-info-container');
        if (blogger_info_container_inner) {
            blogger_info_container_inner.remove();
        }
    });
}
