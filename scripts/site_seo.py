"""Add page metadata and merge mdbook-sitemap output with the landing pages."""

from html import escape, unescape
from html.parser import HTMLParser
from pathlib import Path
import re
import xml.etree.ElementTree as ET
from urllib.parse import urlsplit

SITE_URL = "https://cjv.zxilly.com"
ANALYTICS = '<script defer src="https://trail.learningman.top/script.js" data-website-id="0e5c6fa1-6dd8-452d-acf3-70917b9fb499" data-domains="cjv.zxilly.com,cjv.zxilly.dev" data-exclude-search="true" data-exclude-hash="true"></script>'


def summarize(value, limit=200):
    """Prefer complete sentences, then word boundaries, over cut-off words."""
    if len(value) <= limit:
        return value
    prefix = value[:limit - 1]
    endings = list(re.finditer(r'[。！？]|[.!?](?=\s|$)', prefix))
    if endings and endings[-1].end() >= limit // 3:
        return prefix[:endings[-1].end()]
    if re.search(r'[A-Za-z]$', prefix) and re.match(r'[A-Za-z]', value[limit - 1:]):
        prefix = prefix.rsplit(' ', 1)[0]
    return prefix.rstrip(' ,，;；:：') + '…'


class Summary(HTMLParser):
    def __init__(self, text):
        super().__init__(convert_charrefs=True)
        self.in_main = False
        self.in_paragraph = False
        self.paragraphs = []
        self.current = []
        self.feed(text)

    def handle_starttag(self, tag, attrs):
        if tag == "main":
            self.in_main = True
        if tag == "p" and self.in_main:
            self.in_paragraph = True
            self.current = []

    def handle_endtag(self, tag):
        if tag == "p" and self.in_paragraph:
            value = " ".join("".join(self.current).split())
            if value:
                self.paragraphs.append(value)
            self.in_paragraph = False
        if tag == "main":
            self.in_main = False

    def handle_data(self, data):
        if self.in_paragraph:
            self.current.append(data)


def canonical_path(relative):
    value = "/book/" + relative.as_posix()
    if relative.name in ("index.html", "introduction.html"):
        value = value.rsplit("/", 1)[0] + "/"
    return value


def add_site_metadata(output):
    pages = sorted(output.rglob("*.html"))
    for page in pages:
        text = page.read_text(encoding="utf-8")
        # Redirects and whole-book print copies are not independent search results.
        if page.name in ("print.html", "404.html", "toc.html") or re.search(r'http-equiv=["\']refresh', text, re.I):
            text = text.replace("</head>", '<meta name="robots" content="noindex,follow">\n</head>')
            page.write_text(text, encoding="utf-8")
            continue
        relative = page.relative_to(output)
        parts = relative.parts
        if len(parts) < 3 or parts[1] not in ("zh-CN", "en"):
            continue
        title_match = re.search(r"<title>(.*?)</title>", text, re.S)
        if not title_match:
            raise ValueError(f"Missing page title: {page}")
        title = unescape(title_match[1])
        paragraphs = Summary(text).paragraphs
        description = summarize(paragraphs[0] if paragraphs else title)
        if parts[0] == "user-guide" and relative.as_posix().endswith("/installation/index.html"):
            title, description = (
                ("安装仓颉 SDK：Windows、macOS、Linux 与鸿蒙 - cjv",
                 "使用 cjv 安装仓颉 SDK，配置 cjc 与 cjpm 命令入口。包含 Windows PowerShell、macOS、Linux 和鸿蒙安装步骤、GitCode 镜像及自动化安装参数。")
                if parts[1] == "zh-CN" else
                ("Install the Cangjie SDK on Windows, macOS, Linux and HarmonyOS - cjv",
                 "Install the Cangjie SDK with cjv and set up cjc and cjpm. Includes platform-specific commands, GitCode mirrors and unattended installation options.")
            )
        canonical = SITE_URL + canonical_path(relative)
        alternate_lang = "en" if parts[1] == "zh-CN" else "zh-CN"
        alternate = Path(parts[0], alternate_lang, *parts[2:])
        metadata = [f'<link rel="canonical" href="{escape(canonical, quote=True)}">']
        if (output / alternate).is_file():
            for language, path in ((parts[1], relative), (alternate_lang, alternate)):
                metadata.append(f'<link rel="alternate" hreflang="{language}" href="{SITE_URL + canonical_path(path)}">')
        for property_name, value in {
            "og:type": "article", "og:site_name": "cjv", "og:title": title,
            "og:description": description, "og:url": canonical,
            "og:locale": "zh_CN" if parts[1] == "zh-CN" else "en_US",
        }.items():
            metadata.append(f'<meta property="{property_name}" content="{escape(value, quote=True)}">')
        metadata.append('<meta name="twitter:card" content="summary">')
        text = re.sub(r"<title>.*?</title>", lambda _: f"<title>{escape(title)}</title>", text, count=1, flags=re.S)
        text = re.sub(r'<meta name="description"[^>]*>', '', text)
        metadata.append(f'<meta name="description" content="{escape(description, quote=True)}">')
        if 'trail.learningman.top/script.js' not in text:
            metadata.append(ANALYTICS)
        text = text.replace("</head>", "\n".join(metadata) + "\n</head>")
        page.write_text(text, encoding="utf-8")
    count = merge_sitemaps(output)
    (output.parent / "robots.txt").write_text(f"User-agent: *\nAllow: /\n\nSitemap: {SITE_URL}/sitemap.xml\n", encoding="utf-8")
    print(f"Added documentation metadata and merged {count} canonical sitemap URLs.")


def merge_sitemaps(output):
    """Keep plugin chapter discovery; only normalize paths and add homepages."""
    namespace = "http://www.sitemaps.org/schemas/sitemap/0.9"
    ET.register_namespace("", namespace)
    root = ET.Element(f"{{{namespace}}}urlset")
    urls = set()
    files = sorted(output.glob("*/*/sitemap.xml"))
    if not files:
        raise ValueError("Missing mdbook-sitemap output; build with the sitemap backend enabled")
    for file in files:
        tree = ET.parse(file)
        for entry in tree.getroot():
            location = entry.find(f"{{{namespace}}}loc")
            path = urlsplit(location.text).path
            location.text = SITE_URL + canonical_path(Path(path.removeprefix("/book/")))
            if location.text not in urls:
                root.append(entry)
                urls.add(location.text)
        tree.write(file, encoding="utf-8", xml_declaration=True)
    for url in (SITE_URL + "/", SITE_URL + "/en/"):
        ET.SubElement(ET.SubElement(root, f"{{{namespace}}}url"), f"{{{namespace}}}loc").text = url
        urls.add(url)
    ET.ElementTree(root).write(output.parent / "sitemap.xml", encoding="utf-8", xml_declaration=True)
    return len(urls)
