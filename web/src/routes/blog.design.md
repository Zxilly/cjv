---
name: cjv blog reading surface
description: The implemented bilingual article index and reading pages within the existing cjv website.
colors:
  brand: "#0fa968"
  reading-link: "#087b4b"
  dark-link: "#34d399"
typography:
  title:
    fontFamily: '"Noto Sans SC Variable", system-ui, -apple-system, sans-serif'
    fontSize: "clamp(2rem, 4vw, 3rem)"
    fontWeight: 650
    lineHeight: 1.3
    letterSpacing: "-.025em"
  body:
    fontFamily: '"Noto Sans SC Variable", system-ui, -apple-system, sans-serif'
    fontSize: "1.0625rem"
    lineHeight: 1.9
  metadata:
    fontSize: ".875rem"
    lineHeight: 1.6
---

# Design notes: cjv blog

## Overview

This record applies only to the blog index, article pages, and their shared shell. It describes the built reading extension; it does not establish a new site-wide identity. The existing cjv fonts, green accent, neutral backgrounds, and subdued supporting text remain the visual authority.

Evidence: `blog.tsx`, `blog-post.tsx`, `../components/blog-layout.tsx`, the `.blog-*` rules and inherited tokens in `../style.css`, and the body styles in `../root.tsx`. Supplied desktop index, desktop article, and mobile article captures show the Chinese light appearance. English layout, dark appearance, keyboard behavior, and post-capture changes are documented from source rather than independently exercised by this documentation pass.

## Colors

Brand green identifies the cjv home link. Darker reading green distinguishes the back link, table of contents, body links, and related articles on the light surface. The blog's system-dark media query switches those links and the wordmark to the inherited light-green accent.

Dividers, code backgrounds, and supporting copy reuse `--border`, `--muted`, and `--muted-foreground`; retain their existing light/dark definitions. The page background and main text come from the root layout. This surface does not define its own theme switcher.

## Typography

Noto Sans SC is the shared reading face. Patua One is limited to the cjv wordmark (2.5rem); JetBrains Mono distinguishes inline commands and code blocks. Page titles use the fluid title role above. Article section headings step down to 1.65rem and 1.25rem, both at weight 600. Index item titles use 1.5rem, reduced to 1.3rem on small screens.

Introductions use 1.125rem with a 1.85 line height. Metadata stays smaller and muted, with tabular date numerals. Balanced title wrapping and generous paragraph leading support the bilingual content without a separate typography system for each language.

## Layout

**Reading-column rule.** Keep the article, its contents list, and its related links in one document-flow column. The shared shell is at most 70rem wide; the index is at most 50rem and the article at most 46rem. Horizontal gutters are 2rem, becoming 1.25rem at the 600px breakpoint.

The index presents a heading and introduction followed by separated article rows. An article presents its back link, title, introduction, metadata, numbered contents list, prose, and related articles in that order. The header and footer span the shell, wider than the reading column. On small screens the header retains inline navigation while content wraps naturally; article metadata can wrap onto additional lines.

## Elevation & Depth

The blog uses whitespace and thin dividers for grouping, with no blog-specific shadows. Code examples gain separation through a muted background and border rather than elevation.

## Shapes

Article rows and the contents section use straight separators. Inline code has small rounded corners (.2rem); fenced code blocks have more generous corners (.75rem). These code treatments do not imply that the article list should become a card grid.

## Components

- **Shared navigation:** cjv links to the current language's home with a full document request so edge-provided installation defaults are refreshed; the navigation links to the blog, user guide, and paired language page. `aria-current="page"` is set on the blog navigation link only on the index. Switching language retains the article slug.
- **Article row:** category and date precede a linked title and description. Rows use top dividers and 2rem vertical padding.
- **Contents navigation:** a labeled navigation region contains numbered heading anchors between horizontal dividers. Article headings reserve 2rem of scroll margin.
- **Reading links:** prose, contents, and related links remain underlined at rest. Other shell links gain an underline on hover. Focus-visible links receive a green outline (2px) with a 5px offset. A skip link appears on keyboard focus and targets the main content.
- **Code examples:** inline code remains within prose; fenced blocks preserve command whitespace and scroll horizontally when needed. Their padding reduces from 1.25rem to 1rem at the small-screen breakpoint.
- **Related articles:** a separated section after the article lists the other articles in the current language.

## Do's and Don'ts

- **Do** preserve the shared reading column and clear hierarchy between title, introduction, metadata, and prose.
- **Do** keep translation links paired to the same article and retain the semantic article, navigation, time, and heading elements.
- **Do** inherit existing site tokens and verify both language lengths and dark appearance when changing this surface.
- **Don't** promote this article composition or its local spacing choices into rules for the homepage or documentation book.
- **Don't** treat these supplied light screenshots as verification of dark mode, focus states, or every article's responsive layout.
