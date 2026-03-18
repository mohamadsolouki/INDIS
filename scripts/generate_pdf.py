#!/usr/bin/env python3
"""Generate PDF versions of INDIS PRD documents.

Usage:
    python3 scripts/generate_pdf.py

Outputs:
    docs/INDIS_PRD_v1.1_bilingual.pdf   — English/Persian bilingual
    docs/INDIS_PRD_v1.1_fa.pdf          — Full Persian
"""

import os
import re
import sys
from pathlib import Path

import markdown
from weasyprint import CSS, HTML

REPO_ROOT = Path(__file__).parent.parent
DOCS_DIR = REPO_ROOT / "docs"
DOCS_DIR.mkdir(exist_ok=True)

# Vazirmatn font paths (installed in user's VSCode snap font dir)
VAZIRMATN_PATHS = [
    Path.home() / "snap/code/228/.local/share/fonts",
    Path.home() / ".local/share/fonts",
    Path("/usr/share/fonts/truetype/vazirmatn"),
]

def find_font_dir() -> str:
    for p in VAZIRMATN_PATHS:
        if (p / "Vazirmatn-Regular.ttf").exists():
            return str(p)
    return ""


FONT_DIR = find_font_dir()

BILINGUAL_CSS = f"""
@font-face {{
    font-family: 'Vazirmatn';
    src: url('file://{FONT_DIR}/Vazirmatn-Regular.ttf') format('truetype');
    font-weight: 400;
}}
@font-face {{
    font-family: 'Vazirmatn';
    src: url('file://{FONT_DIR}/Vazirmatn-Medium.ttf') format('truetype');
    font-weight: 500;
}}
@font-face {{
    font-family: 'Vazirmatn';
    src: url('file://{FONT_DIR}/Vazirmatn-Black.ttf') format('truetype');
    font-weight: 700;
}}

@page {{
    size: A4;
    margin: 2cm 2.2cm 2.5cm 2.2cm;
    @bottom-center {{
        content: "INDIS PRD v1.1 | IranProsperityProject.org | " counter(page) " / " counter(pages);
        font-size: 8pt;
        color: #888;
        font-family: 'Vazirmatn', sans-serif;
    }}
}}

* {{ box-sizing: border-box; }}

body {{
    font-family: 'Vazirmatn', 'Noto Naskh Arabic', 'Noto Sans Arabic', sans-serif;
    font-size: 10pt;
    line-height: 1.65;
    color: #1a1a1a;
    background: white;
}}

/* Cover page */
.cover {{
    page: cover;
    height: 100vh;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    text-align: center;
    padding: 3cm;
    background: linear-gradient(135deg, #1a3a5c 0%, #0d5c3a 100%);
    color: white;
    page-break-after: always;
}}
.cover h1 {{ font-size: 22pt; margin-bottom: 0.3em; direction: rtl; }}
.cover h2 {{ font-size: 16pt; margin-bottom: 1em; font-weight: 400; }}
.cover .meta {{ font-size: 10pt; opacity: 0.85; line-height: 2; }}
.cover .logo {{ font-size: 32pt; margin-bottom: 0.5em; }}

/* Typography */
h1 {{ font-size: 16pt; color: #1a3a5c; border-bottom: 2px solid #1a3a5c; padding-bottom: 4pt; margin-top: 24pt; page-break-after: avoid; }}
h2 {{ font-size: 13pt; color: #0d5c3a; margin-top: 18pt; page-break-after: avoid; }}
h3 {{ font-size: 11pt; color: #1a3a5c; margin-top: 14pt; page-break-after: avoid; }}
h4 {{ font-size: 10pt; color: #333; font-weight: 600; margin-top: 10pt; page-break-after: avoid; }}

/* Persian content gets RTL */
p, li, td, th {{
    unicode-bidi: embed;
}}

/* Detect RTL paragraphs (start with Arabic/Persian characters) */
p:lang(fa), .rtl {{ direction: rtl; text-align: right; }}

/* Tables */
table {{
    width: 100%;
    border-collapse: collapse;
    margin: 8pt 0 12pt 0;
    font-size: 9pt;
    page-break-inside: avoid;
}}
th {{
    background-color: #1a3a5c;
    color: white;
    padding: 6pt 8pt;
    text-align: left;
    font-weight: 600;
}}
td {{
    padding: 5pt 8pt;
    border: 1px solid #ddd;
    vertical-align: top;
}}
tr:nth-child(even) td {{ background-color: #f7f9fc; }}
tr:hover td {{ background-color: #eef3f8; }}

/* Code blocks */
pre {{
    background-color: #f4f4f4;
    border: 1px solid #ddd;
    border-left: 4px solid #1a3a5c;
    padding: 10pt;
    font-size: 8pt;
    font-family: 'Courier New', 'DejaVu Sans Mono', monospace;
    overflow-x: auto;
    white-space: pre-wrap;
    word-break: break-word;
    direction: ltr;
    text-align: left;
    page-break-inside: avoid;
    margin: 8pt 0;
}}
code {{
    background-color: #f0f0f0;
    padding: 1pt 4pt;
    border-radius: 2pt;
    font-family: 'Courier New', monospace;
    font-size: 8.5pt;
}}

/* Blockquotes */
blockquote {{
    border-left: 4px solid #e8a000;
    margin: 10pt 0;
    padding: 8pt 12pt;
    background: #fffbf0;
    color: #444;
    font-size: 9.5pt;
}}

/* Lists */
ul, ol {{ padding-left: 20pt; margin: 6pt 0; }}
li {{ margin-bottom: 3pt; }}

/* Horizontal rules */
hr {{
    border: none;
    border-top: 1px solid #ddd;
    margin: 16pt 0;
}}

/* Page break utilities */
.page-break {{ page-break-after: always; }}

/* Section labels */
.section-header {{
    background: #1a3a5c;
    color: white;
    padding: 6pt 10pt;
    margin: 16pt 0 8pt 0;
    font-size: 11pt;
    font-weight: 600;
}}
"""

PERSIAN_CSS = f"""
@font-face {{
    font-family: 'Vazirmatn';
    src: url('file://{FONT_DIR}/Vazirmatn-Regular.ttf') format('truetype');
    font-weight: 400;
}}
@font-face {{
    font-family: 'Vazirmatn';
    src: url('file://{FONT_DIR}/Vazirmatn-Medium.ttf') format('truetype');
    font-weight: 500;
}}
@font-face {{
    font-family: 'Vazirmatn';
    src: url('file://{FONT_DIR}/Vazirmatn-Black.ttf') format('truetype');
    font-weight: 700;
}}

@page {{
    size: A4;
    margin: 2cm 2.2cm 2.5cm 2.2cm;
    @bottom-center {{
        content: "سیستم هویت دیجیتال ملی ایران | INDIS PRD نسخه ۱.۱ | " counter(page) " / " counter(pages);
        font-size: 8pt;
        color: #888;
        font-family: 'Vazirmatn', sans-serif;
    }}
}}

* {{ box-sizing: border-box; }}

body {{
    font-family: 'Vazirmatn', 'Noto Naskh Arabic', 'Noto Sans Arabic', sans-serif;
    font-size: 10.5pt;
    line-height: 1.8;
    color: #1a1a1a;
    background: white;
    direction: rtl;
    text-align: right;
}}

/* Cover page */
.cover {{
    height: 100vh;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    text-align: center;
    padding: 3cm;
    background: linear-gradient(135deg, #1a3a5c 0%, #0d5c3a 100%);
    color: white;
    page-break-after: always;
}}
.cover h1 {{ font-size: 24pt; margin-bottom: 0.3em; direction: rtl; }}
.cover h2 {{ font-size: 14pt; margin-bottom: 1em; font-weight: 400; direction: ltr; }}
.cover .meta {{ font-size: 10pt; opacity: 0.85; line-height: 2.2; direction: rtl; }}

/* Typography */
h1 {{ font-size: 16pt; color: #1a3a5c; border-bottom: 2px solid #1a3a5c; padding-bottom: 4pt; margin-top: 24pt; page-break-after: avoid; }}
h2 {{ font-size: 13pt; color: #0d5c3a; margin-top: 18pt; page-break-after: avoid; }}
h3 {{ font-size: 11pt; color: #1a3a5c; margin-top: 14pt; page-break-after: avoid; }}
h4 {{ font-size: 10.5pt; color: #333; font-weight: 600; margin-top: 10pt; page-break-after: avoid; }}

/* Tables — RTL */
table {{
    width: 100%;
    border-collapse: collapse;
    margin: 8pt 0 12pt 0;
    font-size: 9.5pt;
    page-break-inside: avoid;
    direction: rtl;
}}
th {{
    background-color: #1a3a5c;
    color: white;
    padding: 6pt 8pt;
    text-align: right;
    font-weight: 600;
}}
td {{
    padding: 5pt 8pt;
    border: 1px solid #ddd;
    vertical-align: top;
    text-align: right;
}}
tr:nth-child(even) td {{ background-color: #f7f9fc; }}

/* Code blocks — keep LTR for code */
pre {{
    background-color: #f4f4f4;
    border: 1px solid #ddd;
    border-right: 4px solid #1a3a5c;
    border-left: none;
    padding: 10pt;
    font-size: 8pt;
    font-family: 'Courier New', 'DejaVu Sans Mono', monospace;
    white-space: pre-wrap;
    word-break: break-word;
    direction: ltr;
    text-align: left;
    page-break-inside: avoid;
    margin: 8pt 0;
    unicode-bidi: bidi-override;
}}
code {{
    background-color: #f0f0f0;
    padding: 1pt 4pt;
    border-radius: 2pt;
    font-family: 'Courier New', monospace;
    font-size: 8.5pt;
    direction: ltr;
    unicode-bidi: embed;
}}

/* Blockquotes */
blockquote {{
    border-right: 4px solid #e8a000;
    border-left: none;
    margin: 10pt 0;
    padding: 8pt 12pt 8pt 0;
    padding-right: 12pt;
    background: #fffbf0;
    color: #444;
    font-size: 9.5pt;
}}

/* Lists — RTL */
ul, ol {{
    padding-right: 20pt;
    padding-left: 0;
    margin: 6pt 0;
}}
li {{ margin-bottom: 3pt; }}

/* Horizontal rules */
hr {{
    border: none;
    border-top: 1px solid #ddd;
    margin: 16pt 0;
}}
"""


def md_to_html(md_content: str, rtl: bool = False) -> str:
    """Convert Markdown to HTML with proper extensions."""
    md = markdown.Markdown(
        extensions=[
            "tables",
            "fenced_code",
            "codehilite",
            "toc",
            "nl2br",
            "attr_list",
            "def_list",
        ]
    )
    body = md.convert(md_content)
    lang = "fa" if rtl else "en"
    dir_attr = 'dir="rtl"' if rtl else 'dir="ltr"'
    return f"""<!DOCTYPE html>
<html lang="{lang}" {dir_attr}>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width">
</head>
<body>
{body}
</body>
</html>"""


def generate_cover_bilingual() -> str:
    return """
<div class="cover">
  <div class="logo">IRAN</div>
  <h1>سیستم هویت دیجیتال ملی ایران</h1>
  <h2>Iran National Digital Identity System (INDIS)</h2>
  <h2>سند نیازمندی‌های محصول — Product Requirements Document</h2>
  <div class="meta">
    نسخه / Version: 1.1<br>
    IranProsperityProject.org<br>
    طبقه‌بندی / Classification: Strategic Planning — Public Draft<br>
    زبان / Language: فارسی (Persian) + English (Bilingual)
  </div>
</div>
"""


def generate_cover_persian() -> str:
    return """
<div class="cover">
  <div style="font-size:40pt; margin-bottom:0.3em;">IRAN</div>
  <h1>سیستم هویت دیجیتال ملی ایران</h1>
  <h2>INDIS — Iran National Digital Identity System</h2>
  <h1 style="font-size:18pt; border:none; color:white; margin-top:0.3em;">سند نیازمندی‌های محصول</h1>
  <div class="meta">
    نسخه: ۱.۱<br>
    IranProsperityProject.org<br>
    طبقه‌بندی: برنامه‌ریزی راهبردی — پیش‌نویس عمومی<br>
    زبان سند: فارسی (نسخه معتبر برای اهداف حاکمیتی)
  </div>
</div>
"""


def generate_pdf(
    md_path: Path,
    pdf_path: Path,
    css: str,
    rtl: bool,
    cover_html: str,
) -> None:
    print(f"  Reading {md_path.name} …")
    content = md_path.read_text(encoding="utf-8")

    print("  Converting Markdown → HTML …")
    html = md_to_html(content, rtl=rtl)

    # Inject cover page after <body>
    html = html.replace("<body>", f"<body>\n{cover_html}\n", 1)

    print("  Rendering PDF with WeasyPrint …")
    HTML(string=html, base_url=str(REPO_ROOT)).write_pdf(
        str(pdf_path),
        stylesheets=[CSS(string=css)],
        optimize_images=True,
    )
    size_kb = pdf_path.stat().st_size // 1024
    print(f"  ✓ Written: {pdf_path.relative_to(REPO_ROOT)}  ({size_kb} KB)")


def main() -> None:
    print("INDIS PDF Generator")
    print("=" * 50)

    # --- Bilingual PDF ---
    print("\n[1/2] Bilingual PRD (English + Persian)")
    generate_pdf(
        md_path=REPO_ROOT / "INDIS_PRD_v1.0.md",
        pdf_path=DOCS_DIR / "INDIS_PRD_v1.1_bilingual.pdf",
        css=BILINGUAL_CSS,
        rtl=False,
        cover_html=generate_cover_bilingual(),
    )

    # --- Persian PDF ---
    print("\n[2/2] Persian PRD (فارسی)")
    generate_pdf(
        md_path=REPO_ROOT / "INDIS_PRD_v1.0_fa.md",
        pdf_path=DOCS_DIR / "INDIS_PRD_v1.1_fa.pdf",
        css=PERSIAN_CSS,
        rtl=True,
        cover_html=generate_cover_persian(),
    )

    print("\n✅ Both PDFs generated in docs/")


if __name__ == "__main__":
    main()
