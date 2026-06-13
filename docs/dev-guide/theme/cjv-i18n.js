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
    btn.innerHTML = '<i class="fa fa-globe"></i>';

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
