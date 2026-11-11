"""Assemble the site sitemap from mdBook and prerendered website outputs.

Page metadata belongs to mdbook-seo and React Router. This module only reads it;
canonical paths are never guessed from chapter filenames.
"""

from html.parser import HTMLParser
from urllib.parse import unquote, urlsplit
import xml.etree.ElementTree as ET

SITE_URL = "https://cjv.zxilly.com"
NAMESPACE = "http://www.sitemaps.org/schemas/sitemap/0.9"


class Metadata(HTMLParser):
    def __init__(self, text):
        super().__init__(convert_charrefs=True)
        self.in_head = False
        self.canonicals = []
        self.noindex = False
        self.feed(text)

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if tag == "head":
            self.in_head = True
        if not self.in_head:
            return
        if tag == "link" and attrs.get("rel") == "canonical":
            self.canonicals.append(attrs.get("href", ""))
        if tag == "meta" and attrs.get("name", "").lower() == "robots":
            self.noindex |= "noindex" in attrs.get("content", "").lower().replace(",", " ").split()

    def handle_endtag(self, tag):
        if tag == "head":
            self.in_head = False


def local_page(site, url):
    parsed = urlsplit(url)
    if parsed.scheme != "https" or parsed.netloc != "cjv.zxilly.com" or parsed.query or parsed.fragment:
        raise ValueError(f"Invalid canonical site URL: {url}")
    path = (site / unquote(parsed.path).lstrip("/")).resolve()
    path.relative_to(site.resolve())
    return path / "index.html" if parsed.path.endswith("/") else path


def canonical_for(site, page):
    metadata = Metadata(page.read_text(encoding="utf-8"))
    if metadata.noindex:
        return None
    if len(metadata.canonicals) != 1:
        raise ValueError(f"Expected one canonical in {page}")
    canonical = metadata.canonicals[0]
    target = local_page(site, canonical)
    target_metadata = Metadata(target.read_text(encoding="utf-8"))
    if target_metadata.noindex or target_metadata.canonicals != [canonical]:
        raise ValueError(f"Canonical target is not self-canonical and indexable: {canonical}")
    return canonical


def write_site_map(output):
    """Normalize chapter sitemaps to rendered canonical URLs and merge the site."""
    site = output.parent
    ET.register_namespace("", NAMESPACE)
    root = ET.Element(f"{{{NAMESPACE}}}urlset")
    urls = set()
    files = sorted(output.glob("*/*/sitemap.xml"))
    if not files:
        raise ValueError("Missing mdbook-sitemap output; build with the sitemap backend enabled")
    for file in files:
        tree = ET.parse(file)
        book_urls = set()
        for entry in list(tree.getroot()):
            location = entry.find(f"{{{NAMESPACE}}}loc")
            if location is None or not location.text:
                raise ValueError(f"Missing sitemap location: {file}")
            source = local_page(site, location.text)
            source.relative_to(file.parent.resolve())
            canonical = canonical_for(site, source)
            if canonical is None or canonical in book_urls:
                tree.getroot().remove(entry)
                continue
            location.text = canonical
            book_urls.add(canonical)
            if canonical not in urls:
                root.append(entry)
                urls.add(canonical)
        tree.write(file, encoding="utf-8", xml_declaration=True)
    # Documentation-only builds need not advertise a missing frontend.
    pages = [site / "index.html", site / "en/index.html"]
    for prefix in ("blog", "en/blog"):
        pages.extend(sorted((site / prefix).rglob("index.html")))
    for page in pages:
        if not page.is_file():
            continue
        canonical = canonical_for(site, page)
        relative = page.parent.relative_to(site).as_posix()
        expected = SITE_URL + "/" + ("" if relative == "." else relative)
        expected = expected.rstrip("/") + "/"
        if canonical != expected:
            raise ValueError(f"Inconsistent website canonical: {page}")
        if canonical not in urls:
            ET.SubElement(ET.SubElement(root, f"{{{NAMESPACE}}}url"), f"{{{NAMESPACE}}}loc").text = canonical
            urls.add(canonical)
    ET.ElementTree(root).write(site / "sitemap.xml", encoding="utf-8", xml_declaration=True)
    (site / "robots.txt").write_text(f"User-agent: *\nAllow: /\n\nSitemap: {SITE_URL}/sitemap.xml\n", encoding="utf-8")
    print(f"Merged {len(urls)} rendered canonical sitemap URLs.")
    return len(urls)
