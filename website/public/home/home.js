(function () {
    "use strict";

    var header = document.querySelector("[data-header]");
    var menuToggle = document.querySelector("[data-menu-toggle]");
    var navigation = document.querySelector("[data-nav]");
    var clock = document.querySelector("[data-clock]");
    var revealItems = Array.prototype.slice.call(document.querySelectorAll(".reveal"));
    var reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)");

    /**
     * 根据页面滚动状态切换页头的背景样式。
     */
    function updateHeader() {
        if (!header) {
            return;
        }
        header.classList.toggle("is-scrolled", window.scrollY > 24);
    }

    /**
     * 展开或收起移动端导航。
     */
    function toggleNavigation(forceOpen) {
        if (!menuToggle || !navigation) {
            return;
        }
        var nextOpen = typeof forceOpen === "boolean" ? forceOpen : menuToggle.getAttribute("aria-expanded") !== "true";
        menuToggle.setAttribute("aria-expanded", nextOpen ? "true" : "false");
        navigation.classList.toggle("is-open", nextOpen);
    }

    /**
     * 更新页面上展示的上海时区时间。
     */
    function updateClock() {
        if (!clock) {
            return;
        }
        try {
            clock.textContent = new Intl.DateTimeFormat("zh-CN", {
                timeZone: "Asia/Shanghai",
                hour: "2-digit",
                minute: "2-digit",
                hour12: false
            }).format(new Date()) + " CST";
        } catch (error) {
            clock.textContent = new Date().toTimeString().slice(0, 5);
        }
    }

    /**
     * 为卡片绑定跟随指针的细微高光效果。
     */
    function bindPointerEffects() {
        if (reducedMotion.matches) {
            return;
        }

        Array.prototype.forEach.call(document.querySelectorAll("[data-tilt-surface]"), function (surface) {
            var card = surface.querySelector(".hero-card, .system-frame");
            if (!card) {
                return;
            }

            surface.addEventListener("pointermove", function (event) {
                if (event.pointerType === "touch") {
                    return;
                }
                var rect = surface.getBoundingClientRect();
                var x = ((event.clientX - rect.left) / rect.width) * 100;
                var y = ((event.clientY - rect.top) / rect.height) * 100;
                card.style.setProperty("--pointer-x", x + "%");
                card.style.setProperty("--pointer-y", y + "%");
            });
        });
    }

    /**
     * 初始化滚动进入视口时的淡入动画。
     */
    function bindRevealAnimations() {
        if (!("IntersectionObserver" in window) || reducedMotion.matches) {
            return;
        }

        var observer = new IntersectionObserver(function (entries) {
            entries.forEach(function (entry) {
                if (!entry.isIntersecting) {
                    return;
                }
                entry.target.classList.add("is-visible");
                observer.unobserve(entry.target);
            });
        }, {
            rootMargin: "0px 0px -8% 0px",
            threshold: 0.12
        });

        revealItems.forEach(function (item) {
            observer.observe(item);
        });
    }

    updateHeader();
    updateClock();
    bindPointerEffects();
    bindRevealAnimations();

    window.addEventListener("scroll", updateHeader, { passive: true });
    window.setInterval(updateClock, 30000);

    if (menuToggle && navigation) {
        menuToggle.addEventListener("click", function () {
            toggleNavigation();
        });

        navigation.addEventListener("click", function (event) {
            if (event.target.tagName === "A") {
                toggleNavigation(false);
            }
        });

        document.addEventListener("keydown", function (event) {
            if (event.key === "Escape") {
                toggleNavigation(false);
            }
        });

        document.addEventListener("click", function (event) {
            if (!navigation.contains(event.target) && !menuToggle.contains(event.target)) {
                toggleNavigation(false);
            }
        });
    }
}());
