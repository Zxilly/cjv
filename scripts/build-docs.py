#!/usr/bin/env python3
"""Build the four mdBooks with the shared theme, then check local links."""

import argparse
from html.parser import HTMLParser
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
from urllib.parse import unquote, urlsplit


ROOT = Path(__file__).resolve().parent.parent
BOOKS = {
    "user-guide": {"zh-CN": "cjv 用户手册", "en": "cjv User Guide"},
    "dev-guide": {"zh-CN": "cjv 开发指南", "en": "cjv Developer Guide"},
}
SITE_HOSTS = {"cjv.zxilly.com", "cjv.zxilly.dev"}


class Page(HTMLParser):
    def __init__(self, text):
        super().__init__(convert_charrefs=True)
        self.ids = set()
        self.links = []
        self.feed(text)

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if attrs.get("id"):
            self.ids.add(attrs["id"])
        if tag == "a" and attrs.get("name"):
            self.ids.add(attrs["name"])
        attr = {"a": "href", "link": "href", "script": "src", "img": "src"}.get(tag)
        if attr and attrs.get(attr):
            self.links.append((attrs[attr], self.getpos()[0]))


def check_links(output):
    pages = {
        path.resolve(): Page(path.read_text(encoding="utf-8"))
        for path in output.rglob("*.html")
    }
    if not pages:
        raise ValueError(f"no HTML pages found in {output}")
    errors = []
    for path, page in pages.items():
        for link, line in page.links:
            url = urlsplit(link)
            # Cross-book links to either public documentation host are local.
            if url.scheme or url.netloc:
                if url.scheme not in ("http", "https") or url.hostname not in SITE_HOSTS:
                    continue
                if not url.path.startswith("/book/"):
                    continue
            target_path = unquote(url.path)
            if target_path.startswith("/book/"):
                target = output / target_path.removeprefix("/book/")
            elif target_path.startswith("/"):
                continue
            else:
                target = path.parent / target_path if target_path else path
            target = target.resolve()
            if target.is_dir():
                target /= "index.html"
            try:
                target.relative_to(output)
            except ValueError:
                reason = "outside the documentation output"
            else:
                if not target.is_file():
                    reason = "missing file"
                elif url.fragment and target in pages and unquote(url.fragment) not in pages[target].ids:
                    reason = "missing anchor"
                else:
                    continue
            errors.append(f"{path.relative_to(output)}:{line}: {reason}: {link}")
    if errors:
        raise ValueError("\n".join(errors))
    print(f"Checked local links and anchors in {len(pages)} HTML pages.")


def build(output):
    # Stage source inputs so mdBook can fingerprint the shared theme under each
    # language's output without duplicated source assets or ../ asset URLs.
    with tempfile.TemporaryDirectory(prefix="cjv-docs-") as temporary:
        staging = Path(temporary)
        for book, titles in BOOKS.items():
            source = ROOT / "docs" / book
            staged = staging / book
            staged.mkdir()
            shutil.copy2(source / "book.toml", staged / "book.toml")
            for language_dir in ("src", "en"):
                shutil.copytree(source / language_dir, staged / language_dir)
            shutil.copytree(ROOT / "docs" / "theme", staged / "theme")
            for language, title in titles.items():
                env = os.environ.copy()
                env.update({
                    "MDBOOK_BOOK__SRC": "src" if language == "zh-CN" else "en",
                    "MDBOOK_BOOK__LANGUAGE": language,
                    "MDBOOK_BOOK__TITLE": title,
                    "MDBOOK_OUTPUT__HTML__SITE_URL": f"/book/{book}/{language}/",
                    "MDBOOK_OUTPUT__HTML__ADDITIONAL_JS": json.dumps(["theme/cjv-i18n.js"]),
                    "MDBOOK_OUTPUT__HTML__ADDITIONAL_CSS": json.dumps(["theme/cjv-i18n.css"]),
                })
                subprocess.run(
                    ["mdbook", "build", str(staged), "--dest-dir", str(output / book / language)],
                    env=env,
                    check=True,
                )


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, default=ROOT / "web" / "dist" / "book")
    parser.add_argument("--check-only", action="store_true", help="check an existing documentation build")
    args = parser.parse_args()
    output = args.output.resolve()
    if not args.check_only:
        build(output)
    check_links(output)


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        print(error, file=sys.stderr)
        sys.exit(1)
