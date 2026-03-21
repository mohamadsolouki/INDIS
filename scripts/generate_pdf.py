#!/usr/bin/env python3
"""Generate PDF version of INDIS PRD v2.0.

Usage:
    python3 scripts/generate_pdf.py

Output:
    docs/INDIS_PRD_v2.0.pdf
"""

import re
import sys
from pathlib import Path

import markdown
from weasyprint import CSS, HTML

REPO_ROOT = Path(__file__).parent.parent
DOCS_DIR = REPO_ROOT / "docs"
DOCS_DIR.mkdir(exist_ok=True)

# ---------------------------------------------------------------------------
# Font discovery
# ---------------------------------------------------------------------------

VAZIRMATN_PATHS = [
    Path.home() / "snap/code/228/.local/share/fonts",
    Path.home() / ".local/share/fonts",
    Path("/usr/share/fonts/truetype/vazirmatn"),
    Path("/usr/share/fonts"),
]

def find_font_dir() -> str:
    for p in VAZIRMATN_PATHS:
        if (p / "Vazirmatn-Regular.ttf").exists():
            return str(p)
    return ""

FONT_DIR = find_font_dir()
if not FONT_DIR:
    print("WARNING: Vazirmatn font not found — falling back to system sans-serif.", file=sys.stderr)

def font_src(weight_name: str) -> str:
    """Return a valid CSS src line, or empty string if file missing."""
    if not FONT_DIR:
        return "local('sans-serif')"
    ttf = Path(FONT_DIR) / f"Vazirmatn-{weight_name}.ttf"
    if ttf.exists():
        return f"url('file://{ttf}') format('truetype')"
    return "local('sans-serif')"


# ---------------------------------------------------------------------------
# CSS
# ---------------------------------------------------------------------------

MAIN_CSS = f"""
/* ------------------------------------------------------------------ */
/* Font faces                                                           */
/* ------------------------------------------------------------------ */
@font-face {{
    font-family: 'Vazirmatn';
    src: {font_src('Regular')};
    font-weight: 400;
    font-style: normal;
}}
@font-face {{
    font-family: 'Vazirmatn';
    src: {font_src('Medium')};
    font-weight: 500;
    font-style: normal;
}}
@font-face {{
    font-family: 'Vazirmatn';
    src: {font_src('SemiBold')};
    font-weight: 600;
    font-style: normal;
}}
@font-face {{
    font-family: 'Vazirmatn';
    src: {font_src('Bold')};
    font-weight: 700;
    font-style: normal;
}}

/* ------------------------------------------------------------------ */
/* Page layout                                                          */
/* ------------------------------------------------------------------ */
@page {{
    size: A4;
    margin: 2.2cm 2.4cm 2.8cm 2.4cm;

    @bottom-center {{
        content: "INDIS  |  PRD v2.0  |  " counter(page) " / " counter(pages);
        font-family: 'Vazirmatn', sans-serif;
        font-size: 7.5pt;
        color: #8a9ab0;
        letter-spacing: 0.04em;
        border-top: 0.5pt solid #d0d8e4;
        padding-top: 4pt;
    }}
}}

@page cover {{
    margin: 0;
    @bottom-left  {{ content: ""; }}
    @bottom-right {{ content: ""; }}
    @bottom-center {{ content: ""; border: none; }}
}}

@page toc-page {{
    @bottom-center {{
        content: "INDIS  |  PRD v2.0  |  " counter(page) " / " counter(pages);
        font-family: 'Vazirmatn', sans-serif;
        font-size: 7.5pt;
        color: #8a9ab0;
        letter-spacing: 0.04em;
        border-top: 0.5pt solid #d0d8e4;
        padding-top: 4pt;
    }}
}}

/* ------------------------------------------------------------------ */
/* Base                                                                 */
/* ------------------------------------------------------------------ */
* {{ box-sizing: border-box; margin: 0; padding: 0; }}

body {{
    font-family: 'Vazirmatn', 'Noto Sans Arabic', 'Segoe UI', Arial, sans-serif;
    font-size: 10pt;
    line-height: 1.7;
    color: #1c2535;
    background: #ffffff;
}}

/* ------------------------------------------------------------------ */
/* Cover page                                                           */
/* ------------------------------------------------------------------ */
/*
  Layout:
    - Full-page white canvas
    - Solid navy left stripe (decorative, 2cm wide)
    - Top header band: solid #1a3a5c, 5.5cm tall — white text, high contrast
    - Central content area: white, dark text — guaranteed legibility
    - Bottom metadata band: light grey (#f0f4f8), dark text
*/

.cover {{
    page: cover;
    page-break-after: always;
    width: 21cm;
    height: 29.7cm;
    position: relative;
    background: #ffffff;
    display: block;
    overflow: hidden;
}}

/* Top header band — solid colour, no gradient, white text */
.cover-header {{
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 8.5cm;
    background: #1a3a5c;
    display: flex;
    flex-direction: column;
    justify-content: flex-end;
    padding: 0 2.4cm 0.8cm 2.4cm;
}}

/* Thin gold accent rule below header */
.cover-header::after {{
    content: '';
    display: block;
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 4pt;
    background: #c8a84b;
}}

.cover-eyebrow {{
    font-size: 8pt;
    font-weight: 600;
    letter-spacing: 0.18em;
    color: #8fb8d8;
    text-transform: uppercase;
    margin-bottom: 0.5cm;
}}

.cover-title-en {{
    font-size: 24pt;
    font-weight: 700;
    color: #ffffff;
    line-height: 1.15;
    letter-spacing: -0.01em;
    margin-bottom: 0.25cm;
}}

.cover-title-fa {{
    font-size: 14pt;
    font-weight: 400;
    color: #b8d0e8;
    direction: rtl;
    unicode-bidi: embed;
    letter-spacing: 0.01em;
}}

/* Central content area */
.cover-body {{
    position: absolute;
    top: 8.9cm;
    left: 2.4cm;
    right: 2.4cm;
    bottom: 5.5cm;
    display: flex;
    flex-direction: column;
    justify-content: flex-start;
    padding-top: 0.8cm;
    border-left: 4pt solid #c8a84b;
    padding-left: 0.6cm;
}}

.cover-doc-type {{
    font-size: 11pt;
    font-weight: 600;
    color: #1a3a5c;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    margin-bottom: 0.4cm;
}}

.cover-description {{
    font-size: 10pt;
    font-weight: 400;
    color: #3a4a60;
    line-height: 1.65;
    max-width: 14cm;
    margin-bottom: 1cm;
}}

.cover-version-block {{
    display: inline-block;
    border: 1.5pt solid #1a3a5c;
    padding: 0.3cm 0.6cm;
    margin-top: 0.3cm;
}}

.cover-version-label {{
    font-size: 7pt;
    font-weight: 600;
    letter-spacing: 0.15em;
    color: #8a9ab0;
    text-transform: uppercase;
    margin-bottom: 0.15cm;
}}

.cover-version-num {{
    font-size: 16pt;
    font-weight: 700;
    color: #1a3a5c;
}}

/* Bottom metadata band */
.cover-footer {{
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 5cm;
    background: #f0f4f8;
    border-top: 1pt solid #d0d8e4;
    display: flex;
    flex-direction: column;
    justify-content: center;
    padding: 0 2.4cm;
}}

.cover-meta-table {{
    width: 100%;
    border-collapse: collapse;
}}

.cover-meta-table td {{
    border: none;
    padding: 0.15cm 0;
    vertical-align: top;
    background: transparent;
}}

.cover-meta-label {{
    font-size: 7.5pt;
    font-weight: 600;
    letter-spacing: 0.1em;
    color: #6a7a90;
    text-transform: uppercase;
    width: 4cm;
    padding-right: 0.4cm;
}}

.cover-meta-value {{
    font-size: 9pt;
    font-weight: 400;
    color: #1c2535;
}}

.cover-org {{
    font-size: 8pt;
    font-weight: 600;
    letter-spacing: 0.12em;
    color: #4a6a8a;
    text-transform: uppercase;
    margin-top: 0.5cm;
}}

/* Decorative vertical stripe on right edge of cover */
.cover-stripe {{
    position: absolute;
    top: 0;
    right: 0;
    width: 0.5cm;
    height: 100%;
    background: #c8a84b;
    opacity: 0.35;
}}

/* ------------------------------------------------------------------ */
/* Headings                                                             */
/* ------------------------------------------------------------------ */

h1 {{
    font-size: 16pt;
    font-weight: 700;
    color: #1a3a5c;
    border-bottom: 2pt solid #1a3a5c;
    padding-bottom: 5pt;
    margin-top: 28pt;
    margin-bottom: 12pt;
    page-break-after: avoid;
    letter-spacing: -0.01em;
}}

h2 {{
    font-size: 13pt;
    font-weight: 600;
    color: #0d5c3a;
    margin-top: 22pt;
    margin-bottom: 8pt;
    page-break-after: avoid;
    padding-bottom: 3pt;
    border-bottom: 1pt solid #c8e0d4;
}}

h3 {{
    font-size: 11pt;
    font-weight: 600;
    color: #1a3a5c;
    margin-top: 16pt;
    margin-bottom: 6pt;
    page-break-after: avoid;
}}

h4 {{
    font-size: 10pt;
    font-weight: 600;
    color: #3a4a60;
    margin-top: 12pt;
    margin-bottom: 4pt;
    page-break-after: avoid;
}}

/* Part-level banner — injected for PART I / PART II etc. */
.part-banner {{
    background: #1a3a5c;
    color: #ffffff;
    padding: 10pt 14pt;
    margin: 24pt 0 16pt 0;
    page-break-after: avoid;
    page-break-before: always;
}}

.part-banner h1 {{
    color: #ffffff;
    border: none;
    padding: 0;
    margin: 0;
    font-size: 15pt;
    font-weight: 700;
    letter-spacing: 0.02em;
}}

/* ------------------------------------------------------------------ */
/* Body text                                                            */
/* ------------------------------------------------------------------ */

p {{
    margin-bottom: 7pt;
    orphans: 3;
    widows: 3;
}}

strong {{ font-weight: 600; }}

a {{ color: #1a3a5c; text-decoration: none; }}

/* ------------------------------------------------------------------ */
/* Tables                                                               */
/* ------------------------------------------------------------------ */

table {{
    width: 100%;
    border-collapse: collapse;
    margin: 10pt 0 16pt 0;
    font-size: 9pt;
    page-break-inside: auto;
}}

thead tr {{
    background: #1a3a5c;
    color: #ffffff;
}}

th {{
    padding: 6pt 9pt;
    text-align: left;
    font-weight: 600;
    font-size: 8.5pt;
    letter-spacing: 0.03em;
    border: 1pt solid #1a3a5c;
    color: #ffffff;
}}

td {{
    padding: 5pt 9pt;
    border: 1pt solid #d4dce8;
    vertical-align: top;
    color: #1c2535;
}}

tr:nth-child(even) td {{
    background-color: #f5f8fc;
}}

/* Status cells */
td:first-child strong {{
    color: #1a3a5c;
}}

/* ------------------------------------------------------------------ */
/* Code blocks                                                          */
/* ------------------------------------------------------------------ */

pre {{
    background: #f2f5f9;
    border: 1pt solid #d0d8e4;
    border-left: 4pt solid #1a3a5c;
    border-radius: 0 2pt 2pt 0;
    padding: 10pt 12pt;
    font-size: 7.8pt;
    font-family: 'Courier New', 'DejaVu Sans Mono', 'Liberation Mono', monospace;
    white-space: pre-wrap;
    word-break: break-all;
    direction: ltr;
    text-align: left;
    unicode-bidi: embed;
    page-break-inside: avoid;
    margin: 8pt 0 12pt 0;
    line-height: 1.55;
    color: #1c2535;
    overflow: hidden;
}}

code {{
    background: #eef2f7;
    padding: 1pt 4pt;
    border-radius: 2pt;
    font-family: 'Courier New', 'DejaVu Sans Mono', monospace;
    font-size: 8.5pt;
    color: #1a3a5c;
    direction: ltr;
    unicode-bidi: embed;
}}

pre code {{
    background: transparent;
    padding: 0;
    font-size: inherit;
    color: inherit;
}}

/* ------------------------------------------------------------------ */
/* Blockquotes                                                          */
/* ------------------------------------------------------------------ */

blockquote {{
    border-left: 4pt solid #c8a84b;
    margin: 12pt 0;
    padding: 8pt 14pt;
    background: #fdfaf2;
    color: #3a3020;
    font-size: 9.5pt;
    border-radius: 0 2pt 2pt 0;
}}

blockquote p {{
    margin-bottom: 4pt;
}}

blockquote p:last-child {{
    margin-bottom: 0;
}}

/* ------------------------------------------------------------------ */
/* Lists                                                                */
/* ------------------------------------------------------------------ */

ul, ol {{
    padding-left: 18pt;
    margin: 6pt 0 8pt 0;
}}

li {{
    margin-bottom: 3pt;
    padding-left: 2pt;
}}

ul li::marker {{
    color: #c8a84b;
    font-size: 10pt;
}}

ol li::marker {{
    color: #1a3a5c;
    font-weight: 600;
}}

/* ------------------------------------------------------------------ */
/* Horizontal rules                                                     */
/* ------------------------------------------------------------------ */

hr {{
    border: none;
    border-top: 1pt solid #d0d8e4;
    margin: 20pt 0;
}}

/* ------------------------------------------------------------------ */
/* Section dividers injected for PART headings                          */
/* ------------------------------------------------------------------ */

.section-rule {{
    height: 1pt;
    background: linear-gradient(to right, #1a3a5c, #c8a84b, transparent);
    margin: 14pt 0;
}}

/* ------------------------------------------------------------------ */
/* Callout boxes (blockquote with special markers)                      */
/* ------------------------------------------------------------------ */

.callout {{
    border-left: 4pt solid #1a3a5c;
    background: #eef3fa;
    padding: 8pt 12pt;
    margin: 10pt 0;
    font-size: 9.5pt;
}}

/* ------------------------------------------------------------------ */
/* TOC styling                                                          */
/* ------------------------------------------------------------------ */

.toc {{
    page: toc-page;
    page-break-after: always;
}}

/* ------------------------------------------------------------------ */
/* Utilities                                                            */
/* ------------------------------------------------------------------ */

.page-break {{ page-break-after: always; }}
.no-break {{ page-break-inside: avoid; }}

/* Appendix section headers */
.appendix-header {{
    background: #f0f4f8;
    border-left: 4pt solid #c8a84b;
    padding: 6pt 10pt;
    margin: 18pt 0 10pt 0;
    font-size: 10pt;
    font-weight: 600;
    color: #1a3a5c;
    page-break-after: avoid;
}}

/* Final note styling */
.final-note {{
    border: 1pt solid #c8a84b;
    padding: 14pt 18pt;
    margin: 20pt 0;
    background: #fdfaf2;
}}

.final-note p:last-child {{
    margin-bottom: 0;
}}
"""


# ---------------------------------------------------------------------------
# Cover HTML
# ---------------------------------------------------------------------------

COVER_HTML = """
<div class="cover">
  <div class="cover-stripe"></div>

  <div class="cover-header">
    <div class="cover-eyebrow">Iran Prosperity Project &nbsp;&mdash;&nbsp; Strategic Reference Document</div>
    <div class="cover-title-en">Iran National Digital<br>Identity System</div>
    <div class="cover-title-fa">سیستم هویت دیجیتال ملی ایران</div>
  </div>

  <div class="cover-body">
    <div class="cover-doc-type">Product Requirements &amp; System Design Document</div>
    <div class="cover-description">
      A sovereign, privacy-preserving digital identity infrastructure
      for post-transition Iran &mdash; covering 88&nbsp;million domestic citizens
      and 8&ndash;10&nbsp;million diaspora across 50+ countries.
      Built on W3C&nbsp;DID, Verifiable Credentials, and zero-knowledge proofs.
    </div>
    <div class="cover-version-block">
      <div class="cover-version-label">Version</div>
      <div class="cover-version-num">2.0 &mdash; Definitive Edition</div>
    </div>
  </div>

  <div class="cover-footer">
    <table class="cover-meta-table">
      <tr>
        <td class="cover-meta-label">Date</td>
        <td class="cover-meta-value">March 2026 &nbsp;&nbsp;/&nbsp;&nbsp; فروردین ۱۴۰۵ (۲۵۸۵)</td>
      </tr>
      <tr>
        <td class="cover-meta-label">Classification</td>
        <td class="cover-meta-value">Strategic &mdash; For Transitional Government Leadership and International Partners</td>
      </tr>
      <tr>
        <td class="cover-meta-label">Status</td>
        <td class="cover-meta-value">Authoritative Reference &mdash; System ~97% Complete</td>
      </tr>
      <tr>
        <td class="cover-meta-label">License</td>
        <td class="cover-meta-value">CC BY 4.0 &mdash; All cryptographic components open-source and publicly auditable</td>
      </tr>
    </table>
    <div class="cover-org">Iran Prosperity Project</div>
  </div>
</div>
"""

COVER_HTML_FA = """
<div class="cover">
  <div class="cover-stripe"></div>

  <div class="cover-header">
    <div class="cover-eyebrow" style="direction:rtl;text-align:right;">پروژه شکوفایی ایران &nbsp;&mdash;&nbsp; سند مرجع راهبردی</div>
    <div class="cover-title-en" style="direction:rtl;text-align:right;">سامانه ملی هویت دیجیتال ایران</div>
    <div class="cover-title-fa" style="font-size:12pt;color:#b8d0e8;direction:rtl;text-align:right;">Iran National Digital Identity System</div>
  </div>

  <div class="cover-body" style="text-align:right;">
    <div class="cover-doc-type">سند الزامات محصول و طراحی سیستم</div>
    <div class="cover-description" style="direction:rtl;text-align:right;">
      زیرساختی مستقل و حافظ حریم خصوصی برای هویت دیجیتال
      در ایران پس از گذار &mdash; پوشش‌دهنده بیش از ۹۰ میلیون شهروند داخلی
      و ۸ تا ۱۰ میلیون دیاسپورا در بیش از ۵۰ کشور.
      ساخته‌شده بر پایه W3C&nbsp;DID، مدارک قابل تأیید، و اثبات‌های دانش‌صفر.
    </div>
    <div class="cover-version-block">
      <div class="cover-version-label">نسخه</div>
      <div class="cover-version-num">۲.۰ &mdash; نسخه قطعی</div>
    </div>
  </div>

  <div class="cover-footer">
    <table class="cover-meta-table">
      <tr>
        <td class="cover-meta-label" style="direction:rtl;">تاریخ</td>
        <td class="cover-meta-value" style="direction:rtl;">فروردین ۱۴۰۵ (۲۵۸۵) &nbsp;&nbsp;/&nbsp;&nbsp; مارس ۲۰۲۶</td>
      </tr>
      <tr>
        <td class="cover-meta-label" style="direction:rtl;">طبقه‌بندی</td>
        <td class="cover-meta-value" style="direction:rtl;">راهبردی &mdash; برای رهبری دولت انتقالی و شرکای بین‌المللی</td>
      </tr>
      <tr>
        <td class="cover-meta-label" style="direction:rtl;">وضعیت</td>
        <td class="cover-meta-value" style="direction:rtl;">سند مرجع معتبر &mdash; سیستم ~۹۷٪ کامل</td>
      </tr>
      <tr>
        <td class="cover-meta-label" style="direction:rtl;">مجوز</td>
        <td class="cover-meta-value" style="direction:rtl;">CC BY 4.0 &mdash; همه اجزای رمزنگارانه متن‌باز و قابل حسابرسی عمومی</td>
      </tr>
    </table>
    <div class="cover-org" style="direction:rtl;text-align:right;">پروژه شکوفایی ایران</div>
  </div>
</div>
"""

RTL_CSS = """
/* Cover page must stay LTR — it uses absolute positioning */
.cover {
    direction: ltr;
}

body {
    direction: rtl;
    text-align: right;
    unicode-bidi: embed;
}

ul, ol {
    padding-right: 18pt;
    padding-left: 0;
}

blockquote {
    border-left: none;
    border-right: 4pt solid #c8a84b;
}

pre {
    direction: ltr;
    text-align: left;
}

code {
    direction: ltr;
    unicode-bidi: embed;
}

.cover-body {
    border-left: none;
    border-right: 4pt solid #c8a84b;
    padding-left: 0;
    padding-right: 0.6cm;
}

th {
    text-align: right;
}
"""


# ---------------------------------------------------------------------------
# Markdown preprocessing
# ---------------------------------------------------------------------------

# Strip emoji (Unicode ranges covering common emoji blocks)
_EMOJI_RE = re.compile(
    "["
    "\U0001F300-\U0001F9FF"   # Misc symbols and pictographs, supplemental, transport
    "\U00002702-\U000027B0"   # Dingbats
    "\U0001FA00-\U0001FA6F"   # Chess/game symbols
    "\U0001FA70-\U0001FAFF"   # Extended symbols
    "\U00002600-\U000026FF"   # Misc symbols
    "\U00002702-\U000027B0"
    "\u2640-\u2642"
    "\u2194-\u2199"
    "\u2300-\u23FF"           # Misc technical
    "\u24C2"
    "\u25AA-\u25FE"
    "\u2614-\u2615"
    "\u2648-\u2653"
    "\u267F"
    "\u2693"
    "\u26A1"
    "\u26AA-\u26AB"
    "\u26BD-\u26BE"
    "\u26C4-\u26C5"
    "\u26CE"
    "\u26D4"
    "\u26EA"
    "\u26F2-\u26F3"
    "\u26F5"
    "\u26FA"
    "\u26FD"
    "\u2702"
    "\u2705"
    "\u2708-\u270D"
    "\u270F"
    "\u2712"
    "\u2714"
    "\u2716"
    "\u271D"
    "\u2721"
    "\u2728"
    "\u2733-\u2734"
    "\u2744"
    "\u2747"
    "\u274C"
    "\u274E"
    "\u2753-\u2755"
    "\u2757"
    "\u2763-\u2764"
    "\u2795-\u2797"
    "\u27A1"
    "\u27B0"
    "\u27BF"
    "\u231A-\u231B"
    "\u23E9-\u23F3"
    "\u23F8-\u23FA"
    "]+",
    flags=re.UNICODE,
)

# Unicode checkmarks / crosses commonly used as status indicators
_STATUS_EMOJI_MAP = {
    "\u2705": "[DONE]",   # green checkmark box
    "\u274C": "[NO]",     # red X
    "\u26A0": "[NOTE]",   # warning sign
    "\u2714": "[+]",      # heavy checkmark
    "\u2716": "[-]",      # heavy X
    "\u2B50": "[*]",      # star
    "\u26A1": "[!]",      # lightning
}


def strip_emoji(text: str) -> str:
    """Replace status emoji with text tokens, then strip remaining emoji."""
    for char, replacement in _STATUS_EMOJI_MAP.items():
        text = text.replace(char, replacement)
    text = _EMOJI_RE.sub("", text)
    return text


def preprocess_markdown(content: str) -> str:
    """Clean up the markdown before conversion."""
    content = strip_emoji(content)

    # Collapse runs of blank lines to at most 2
    content = re.sub(r"\n{3,}", "\n\n", content)

    return content


# ---------------------------------------------------------------------------
# HTML generation
# ---------------------------------------------------------------------------

def md_to_html(md_content: str) -> str:
    """Convert Markdown to an HTML fragment."""
    md = markdown.Markdown(
        extensions=[
            "tables",
            "fenced_code",
            "toc",
            "nl2br",
            "attr_list",
            "def_list",
            "sane_lists",
        ],
        extension_configs={
            "toc": {
                "title": "",
                "toc_depth": 3,
            }
        },
    )
    return md.convert(md_content)


def build_full_html(body_fragment: str, cover_html: str, lang: str = "en", direction: str = "ltr", title: str = "INDIS PRD v2.0 — Iran National Digital Identity System") -> str:
    """Wrap body fragment in a complete HTML document."""
    return f"""<!DOCTYPE html>
<html lang="{lang}" dir="{direction}">
<head>
<meta charset="utf-8">
<title>{title}</title>
</head>
<body>
{cover_html}
{body_fragment}
</body>
</html>"""


# ---------------------------------------------------------------------------
# PDF generation
# ---------------------------------------------------------------------------

def generate_pdf(md_path: Path, pdf_path: Path, *, rtl: bool = False) -> None:
    print(f"  Reading   {md_path.name}")
    raw = md_path.read_text(encoding="utf-8")

    print("  Cleaning  markdown")
    cleaned = preprocess_markdown(raw)

    print("  Converting  Markdown to HTML")
    body = md_to_html(cleaned)

    if rtl:
        cover = COVER_HTML_FA
        title = "INDIS PRD v2.0 — سامانه ملی هویت دیجیتال ایران"
        html = build_full_html(body, cover, lang="fa", direction="rtl", title=title)
        stylesheets = [CSS(string=MAIN_CSS), CSS(string=RTL_CSS)]
    else:
        html = build_full_html(body, COVER_HTML)
        stylesheets = [CSS(string=MAIN_CSS)]

    print("  Rendering   PDF (WeasyPrint) ...")
    HTML(string=html, base_url=str(REPO_ROOT)).write_pdf(
        str(pdf_path),
        stylesheets=stylesheets,
        optimize_images=True,
        uncompressed_pdf=False,
    )

    size_kb = pdf_path.stat().st_size // 1024
    print(f"  Written   {pdf_path.relative_to(REPO_ROOT)}  ({size_kb:,} KB)")


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

def main() -> None:
    print()
    print("INDIS PDF Generator")
    print("=" * 52)

    jobs = [
        (REPO_ROOT / "INDIS_PRD_v2.0.md",    DOCS_DIR / "INDIS_PRD_v2.0.pdf",    False),
        (REPO_ROOT / "INDIS_PRD_v2.0_fa.md", DOCS_DIR / "INDIS_PRD_v2.0_fa.pdf", True),
    ]

    for md_path, pdf_path, rtl in jobs:
        if not md_path.exists():
            print(f"  SKIP  : {md_path.name} not found", file=sys.stderr)
            continue
        print(f"\n  Source  : {md_path.name}")
        print(f"  Output  : {pdf_path.relative_to(REPO_ROOT)}")
        print()
        generate_pdf(md_path, pdf_path, rtl=rtl)

    print()
    print("Done.")
    print()


if __name__ == "__main__":
    main()
