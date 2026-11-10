import contextlib
import io
from pathlib import Path
import tempfile
import unittest
import xml.etree.ElementTree as ET

from site_seo import add_site_metadata, summarize


class SiteSEOTest(unittest.TestCase):
    def test_summary_preserves_sentence_or_word_boundary(self):
        self.assertEqual(summarize('A useful sentence. ' + 'additional ' * 30, 40), 'A useful sentence.')
        self.assertEqual(summarize('word ' * 20, 18), 'word word word…')

    def test_metadata_and_sitemap_match_real_content_and_language_pairs(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "book"
            page = '<html><head><title>Guide &amp; tools</title><meta name="description" content=""></head><body><main><p>Use &lt;SDK&gt; &amp; "tools".</p></main></body></html>'
            for language in ("zh-CN", "en"):
                base = output / "user-guide" / language
                base.mkdir(parents=True)
                for name in ("index.html", "introduction.html", "toolchain-file.html", "print.html", "toc.html"):
                    (base / name).write_text(page, encoding="utf-8")
                (base / "old.html").write_text('<html><head><meta http-equiv="refresh" content="0; URL=index.html"></head></html>', encoding="utf-8")
                # Chapter discovery belongs to mdbook-sitemap, not the HTML scan.
                names = ["introduction.html", "toolchain-file.html"]
                if language == "zh-CN":
                    names.append("only.html")
                sitemap = ET.Element("urlset", xmlns="http://www.sitemaps.org/schemas/sitemap/0.9")
                for name in names:
                    ET.SubElement(ET.SubElement(sitemap, "url"), "loc").text = f"https://cjv.zxilly.com/book/user-guide/{language}/{name}"
                ET.ElementTree(sitemap).write(base / "sitemap.xml", encoding="utf-8")
            # A page without a translation must not advertise a missing alternate.
            (output / "user-guide/zh-CN/only.html").write_text(page, encoding="utf-8")
            (output / "user-guide/zh-CN/unlisted.html").write_text(page, encoding="utf-8")
            with contextlib.redirect_stdout(io.StringIO()):
                add_site_metadata(output)
            html = (output / "user-guide/zh-CN/toolchain-file.html").read_text(encoding="utf-8")
            self.assertIn('content="Use &lt;SDK&gt; &amp; &quot;tools&quot;."', html)
            self.assertIn('hreflang="en" href="https://cjv.zxilly.com/book/user-guide/en/toolchain-file.html"', html)
            self.assertEqual(html.count('name="description"'), 1)
            self.assertEqual(html.count('trail.learningman.top/script.js'), 1)
            self.assertIn('data-domains="cjv.zxilly.com,cjv.zxilly.dev"', html)
            self.assertNotIn('hreflang=', (output / "user-guide/zh-CN/only.html").read_text(encoding="utf-8"))
            sitemap = ET.parse(output.parent / "sitemap.xml")
            urls = [node.text for node in sitemap.findall('.//{*}loc')]
            self.assertEqual(len(urls), len(set(urls)))
            self.assertIn('https://cjv.zxilly.com/book/user-guide/zh-CN/', urls)
            self.assertFalse(any('unlisted.html' in url for url in urls))
            self.assertFalse(any(name in url for name in ('introduction.html', 'index.html', 'print.html', 'toc.html', 'old.html') for url in urls))
            for name in ('print.html', 'toc.html', 'old.html'):
                self.assertIn('noindex,follow', (output / 'user-guide/en' / name).read_text(encoding='utf-8'))

    def test_missing_plugin_output_fails_instead_of_scanning_html(self):
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaisesRegex(ValueError, 'Missing mdbook-sitemap output'):
                add_site_metadata(Path(directory))


if __name__ == '__main__':
    unittest.main()
