// Inject a language switcher (globe dropdown) into the mdBook top bar.
// Each language is built as its own book under /book/<book>/<lang>/, so this
// just rewrites the language segment of the current path to the chosen one.
(function () {
  "use strict";

  var LANGS = [
    { code: "zh-CN", label: "中文" },
    { code: "en", label: "English" }
  ];

  function currentLang() {
    var path = window.location.pathname;
    for (var i = 0; i < LANGS.length; i++) {
      if (path.indexOf("/" + LANGS[i].code + "/") !== -1) return LANGS[i].code;
    }
    return document.documentElement.getAttribute("lang") || LANGS[0].code;
  }

  function targetUrl(code, current) {
    var loc = window.location;
    if (loc.pathname.indexOf("/" + current + "/") === -1) return null;
    return loc.pathname.replace("/" + current + "/", "/" + code + "/") + loc.search + loc.hash;
  }

  function build() {
    var bar = document.querySelector(".right-buttons");
    if (!bar || document.querySelector(".cjv-lang")) return;

    var current = currentLang();

    var wrap = document.createElement("div");
    wrap.className = "cjv-lang";

    var btn = document.createElement("button");
    btn.className = "icon-button cjv-lang-toggle";
    btn.type = "button";
    btn.title = "Language";
    btn.setAttribute("aria-label", "Select language");
    // mdBook 0.5 renders top-bar icons as inline SVG (.fa-svg), not a webfont,
    // so a Font Awesome <i> would be invisible. Use the same markup it uses.
    btn.innerHTML = '<span class="fa-svg"><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><path d="M352 256c0 22.2-1.2 43.6-3.3 64H163.3c-2.2-20.4-3.3-41.8-3.3-64s1.2-43.6 3.3-64H348.7c2.2 20.4 3.3 41.8 3.3 64zm28.8-64H503.9c5.3 20.5 8.1 41.9 8.1 64s-2.8 43.5-8.1 64H380.8c2.1-20.6 3.2-42 3.2-64s-1.1-43.4-3.2-64zm112.6-32H376.7c-10-63.9-29.8-117.4-55.3-151.6c78.3 20.7 142 77.5 171.9 151.6zm-149.1 0H167.7c6.1-36.4 15.5-68.6 27-94.7c10.5-23.6 22.2-40.7 33.5-51.5C239.4 3.2 248.7 0 256 0s16.6 3.2 27.8 13.8c11.3 10.8 23 27.9 33.5 51.5c11.6 26 20.9 58.2 27 94.7zm-209 0H18.6C48.6 85.9 112.2 29.1 190.6 8.4C165.1 42.6 145.3 96.1 135.3 160zM8.1 192H131.2c-2.1 20.6-3.2 42-3.2 64s1.1 43.4 3.2 64H8.1C2.8 299.5 0 278.1 0 256s2.8-43.5 8.1-64zM194.7 446.6c-11.6-26-20.9-58.2-27-94.6H344.3c-6.1 36.4-15.5 68.6-27 94.6c-10.5 23.6-22.2 40.7-33.5 51.5C272.6 508.8 263.3 512 256 512s-16.6-3.2-27.8-13.8c-11.3-10.8-23-27.9-33.5-51.5zM135.3 352c10 63.9 29.8 117.4 55.3 151.6C112.2 482.9 48.6 426.1 18.6 352H135.3zm358.1 0c-30 74.1-93.6 130.9-171.9 151.6c25.5-34.2 45.3-87.7 55.3-151.6H493.4z"/></svg></span>';

    var menu = document.createElement("ul");
    menu.className = "cjv-lang-menu";
    menu.setAttribute("role", "menu");

    LANGS.forEach(function (l) {
      var li = document.createElement("li");
      if (l.code === current) li.className = "active";
      var a = document.createElement("a");
      a.setAttribute("role", "menuitem");
      a.textContent = l.label;
      var href = targetUrl(l.code, current);
      if (href) {
        a.href = href;
      } else {
        a.setAttribute("aria-disabled", "true");
      }
      a.addEventListener("click", function () {
        try { localStorage.setItem("cjv-doc-lang", l.code); } catch (e) { /* ignore */ }
      });
      li.appendChild(a);
      menu.appendChild(li);
    });

    wrap.appendChild(btn);
    wrap.appendChild(menu);

    btn.addEventListener("click", function (e) {
      e.stopPropagation();
      wrap.classList.toggle("open");
    });
    document.addEventListener("click", function () {
      wrap.classList.remove("open");
    });

    bar.insertBefore(wrap, bar.firstChild);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", build);
  } else {
    build();
  }
})();
