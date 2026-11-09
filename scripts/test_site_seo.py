import contextlib
import io
from pathlib import Path
import tempfile
import unittest
import xml.etree.ElementTree as ET

from site_seo import add_site_metadata


class SiteSEOTest(unittest.TestCase):
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
            # A page without a translation must not advertise a missing alternate.
            (output / "user-guide/zh-CN/only.html").write_text(page, encoding="utf-8")
            with contextlib.redirect_stdout(io.StringIO()):
                add_site_metadata(output)
            html = (output / "user-guide/zh-CN/toolchain-file.html").read_text(encoding="utf-8")
            self.assertIn('content="Use &lt;SDK&gt; &amp; &quot;tools&quot;."', html)
            self.assertIn('hreflang="en" href="https://cjv.zxilly.com/book/user-guide/en/toolchain-file.html"', html)
            self.assertEqual(html.count('name="description"'), 1)
            self.assertNotIn('hreflang=', (output / "user-guide/zh-CN/only.html").read_text(encoding="utf-8"))
            sitemap = ET.parse(output.parent / "sitemap.xml")
            urls = [node.text for node in sitemap.findall('.//{*}loc')]
            self.assertEqual(len(urls), len(set(urls)))
            self.assertIn('https://cjv.zxilly.com/book/user-guide/zh-CN/', urls)
            self.assertFalse(any(name in url for name in ('introduction.html', 'index.html', 'print.html', 'toc.html', 'old.html') for url in urls))
            for name in ('print.html', 'toc.html', 'old.html'):
                self.assertIn('noindex,follow', (output / 'user-guide/en' / name).read_text(encoding='utf-8'))


if __name__ == '__main__':
    unittest.main()
