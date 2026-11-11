import contextlib
import io
from pathlib import Path
import tempfile
import unittest
import xml.etree.ElementTree as ET

from site_map import NAMESPACE, SITE_URL, write_site_map


class SiteMapTest(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.site = Path(self.directory.name)
        self.book = self.site / "book/user-guide/en"
        self.book.mkdir(parents=True)
        for name, canonical in [("index.html", "/book/user-guide/en/"), ("start.html", "/book/user-guide/en/"), ("chapter.html", "/book/user-guide/en/chapter.html")]:
            self.page(self.book / name, SITE_URL + canonical)
        self.page(self.book / "private.html", SITE_URL + "/book/user-guide/en/private.html", noindex=True)
        root = ET.Element(f"{{{NAMESPACE}}}urlset")
        for name in ("start.html", "chapter.html", "private.html"):
            ET.SubElement(ET.SubElement(root, f"{{{NAMESPACE}}}url"), f"{{{NAMESPACE}}}loc").text = SITE_URL + "/book/user-guide/en/" + name
        ET.ElementTree(root).write(self.book / "sitemap.xml", encoding="utf-8")

    def page(self, path, canonical, noindex=False):
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(f'<html><head><link href="{canonical}" rel="canonical">' + ('<meta name="robots" content="noindex,follow">' if noindex else '') + '</head><body>Body</body></html>', encoding="utf-8")

    def build(self):
        with contextlib.redirect_stdout(io.StringIO()):
            return write_site_map(self.site / "book")

    def urls(self, path):
        return [node.text for node in ET.parse(path).findall('.//{*}loc')]

    def test_uses_rendered_canonicals_excludes_noindex_and_preserves_html(self):
        original = (self.book / "start.html").read_bytes()
        self.assertEqual(self.build(), 2)
        self.assertEqual(self.urls(self.site / "sitemap.xml"), [SITE_URL + "/book/user-guide/en/", SITE_URL + "/book/user-guide/en/chapter.html"])
        self.assertEqual(self.urls(self.book / "sitemap.xml"), self.urls(self.site / "sitemap.xml"))
        self.assertEqual((self.book / "start.html").read_bytes(), original)
        self.assertEqual(self.build(), 2)

    def test_includes_only_real_prerendered_pages_and_deduplicates(self):
        for route in ("", "en/", "blog/", "en/blog/", "blog/example/", "en/blog/example/"):
            self.page(self.site / route / "index.html", SITE_URL + "/" + route)
        self.assertEqual(self.build(), 8)
        urls = self.urls(self.site / "sitemap.xml")
        self.assertEqual(len(urls), len(set(urls)))
        self.assertIn(SITE_URL + "/en/blog/example/", urls)
        self.assertIn("Sitemap: " + SITE_URL + "/sitemap.xml", (self.site / "robots.txt").read_text())

    def test_rejects_wrong_domain_and_missing_or_noncanonical_targets(self):
        for canonical in ("https://wrong.example/chapter.html", SITE_URL + "/missing.html", SITE_URL + "/book/user-guide/en/start.html"):
            self.page(self.book / "chapter.html", canonical)
            with self.assertRaises((ValueError, FileNotFoundError)):
                self.build()

    def test_missing_sitemap_fails(self):
        (self.book / "sitemap.xml").unlink()
        with self.assertRaisesRegex(ValueError, "Missing mdbook-sitemap"):
            self.build()


if __name__ == "__main__":
    unittest.main()
