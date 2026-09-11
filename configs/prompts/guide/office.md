## Office File Guide

Covers every `.docx` / `.xlsx` / `.pptx` you create or modify. Done means the file opens without a repair prompt in Word / Excel / PowerPoint **and** in Pages / Numbers / Keynote. Office silently repairs broken packages; Apple's apps reject the same file as "invalid format" — so "PowerPoint opens it" proves nothing. Real case: a 30-slide deck opened in PowerPoint but Keynote refused it, because `presentation.xml` never listed its notes master.

### 1. Build through the library, not the zip

Create and edit with `python-pptx`, `python-docx` or `openpyxl`, then save through the same library. The library keeps relationships, content types and ID lists in step; hand-written XML or re-zipped parts are where every failure below comes from.

- Editing an existing file → open it with the library, change it, save to a new name.
- The library cannot express the change (merging decks, copying slides between files, touching XML directly) → treat the result as hand-built: §2–§4 all apply and §6 is mandatory.

### 2. Package rules (all three formats)

| Rule | Break it and |
|---|---|
| `[Content_Types].xml` is the first zip entry | some readers cannot identify the package |
| Every part has a type — an `<Override>` for its path or a `<Default>` for its extension | the part is ignored or the file is rejected |
| Every internal `Target` in a `.rels` file points at a part that exists | load failure |
| Every `r:id` / `r:embed` / `r:link` used in a part exists in that part's own `.rels` | load failure or blank objects |
| Every part is reachable from `_rels/.rels` through the relationship chain | orphans get dropped or trigger repair |
| No duplicate zip entries; entries use deflate or stored | archive rejected |
| XML text contains no control characters other than tab / LF / CR; `&` and `<` are escaped | parse failure |

### 3. Registration lists per format

A part that exists and is referenced still has to be **listed** in the main part. This is the failure the libraries cannot catch after a manual edit.

**PowerPoint — `ppt/presentation.xml`**

- Children appear in schema order: `sldMasterIdLst`, `notesMasterIdLst`, `handoutMasterIdLst`, `sldIdLst`, `sldSz`, `notesSz`, then the rest.
- A notes master exists (any slide has speaker notes) → `notesMasterIdLst` lists it. A handout master exists → `handoutMasterIdLst` lists it.
- `sldId@id` values are unique and between 256 and 2147483647; `sldMasterId` and every `sldLayoutId` in the masters share one unique space starting at 2147483648.
- Each slide's `.rels` has exactly one `slideLayout`; each notes slide links its slide and the notes master.
- Shape ids (`p:cNvPr@id`) are unique within one slide.

**Word — `word/document.xml`**

- The last child of `w:body` is `w:sectPr`; every `headerReference` / `footerReference` `r:id` exists.
- Every `w:numId` is defined in `numbering.xml` and points at an existing `abstractNum`; every `w:pStyle` / `w:rStyle` exists in `styles.xml`.
- Footnotes and endnotes keep the separator entries with ids `-1` and `0`; references point at ids that exist.
- `bookmarkStart` / `bookmarkEnd` and `commentRangeStart` / `commentRangeEnd` come in pairs with matching unique ids.

**Excel — `xl/workbook.xml`**

- Each `<sheet>` has a unique `sheetId`, an `r:id` that resolves, and a name of at most 31 characters without `[ ] : * ? / \`.
- Cells, formulas or sheets changed outside the library → delete `xl/calcChain.xml` together with its relationship and content-type entry; a stale calc chain is the most common Excel repair prompt.
- Shared-string indexes (`t="s"`) and style indexes (`s=`) stay inside the table they point to.
- Table `id` and `name` are unique across the workbook and `ref` matches the data; defined names point at sheets that exist; merged ranges do not overlap.
- Rows ascend within a sheet and cells ascend within a row.

### 4. Text content

The file shows text exactly as written — Markdown is not rendered.

- Convert Markdown before inserting: no literal `**`, `*`, `#`, backticks or `[text](url)`. Emphasis becomes a bold/italic run, a list becomes real paragraphs with bullet or numbering levels, a link becomes a hyperlink relationship.
- A line break inside one paragraph is `<a:br/>` (PowerPoint) or `<w:br/>` (Word); a new paragraph is a new `a:p` / `w:p`. A raw `\n` inside a text run is not a line break.
- Strip control characters from source text before writing — `openpyxl` raises `IllegalCharacterError` on them, and hand-written XML becomes unparsable.
- Prefer fonts the reader already has. Embedded fonts add megabytes and Apple's apps ignore them; when a template already embeds fonts, keep `embeddedFontLst` and its font parts consistent, or remove both.

### 5. Hand-built structure check

Run this on the output before delivering; every line it prints is a defect to fix. It covers §2 and the `notesMasterIdLst` case from §3 — the rest of §3 still needs a manual look when you touched those parts.

```python
import posixpath, re, sys, zipfile
import xml.etree.ElementTree as ET

z = zipfile.ZipFile(sys.argv[1])
names = z.namelist()
ct = ET.fromstring(z.read("[Content_Types].xml"))
ns = "{http://schemas.openxmlformats.org/package/2006/content-types}"
over = {o.get("PartName").lstrip("/") for o in ct.findall(ns + "Override")}
dflt = {d.get("Extension").lower() for d in ct.findall(ns + "Default")}
if names[0] != "[Content_Types].xml":
    print("first entry is", names[0])
if len(names) != len(set(names)):
    print("duplicate entries")
for n in names:
    if not n.endswith("/") and n != "[Content_Types].xml" and n not in over and n.rsplit(".", 1)[-1].lower() not in dflt:
        print("no content type:", n)
for n in names:
    if n.endswith((".xml", ".rels")):
        try:
            ET.fromstring(z.read(n))
        except ET.ParseError as e:
            print("bad xml:", n, e)
for n in names:
    if not n.endswith(".rels"):
        continue
    base = posixpath.dirname(posixpath.dirname(n))
    owner = posixpath.join(base, posixpath.basename(n)[:-5])
    ids = set()
    for r in ET.fromstring(z.read(n)):
        ids.add(r.get("Id"))
        t = r.get("Target")
        if r.get("TargetMode") != "External":
            p = t.lstrip("/") if t.startswith("/") else posixpath.normpath(posixpath.join(base, t))
            if p not in names:
                print("missing target:", n, t)
    if owner in names:
        used = set(re.findall(r'r:(?:id|embed|link|pict)="([^"]+)"', z.read(owner).decode("utf-8", "replace")))
        for rid in sorted(used - ids):
            print("dangling", rid, "in", owner)
if "ppt/presentation.xml" in names:
    pres = z.read("ppt/presentation.xml").decode()
    if any(n.startswith("ppt/notesMasters/") for n in names) and "notesMasterIdLst" not in pres:
        print("notes master exists but presentation.xml has no notesMasterIdLst")
if "xl/calcChain.xml" in names:
    print("calcChain.xml present: delete it if cells were changed outside openpyxl")
```

### 6. Verify before delivering

1. Re-open the saved file with the library that matches it (`Presentation(path)`, `Document(path)`, `load_workbook(path)`). This catches parse errors only.
2. Hand-built parts or merged files → run the §5 check and fix everything it prints.
3. On macOS with Keynote, Pages or Numbers installed, open the file in the matching app as the final gate — it is the strict reader. It launches the app on the user's screen, so run it once on the finished file, not on every iteration:

```bash
osascript -e 'tell application id "com.apple.Keynote"
  set d to open (POSIX file "/abs/path/deck.pptx")
  close d saving no
end tell'
```

Use `com.apple.Pages` for `.docx` and `com.apple.Numbers` for `.xlsx`. An error `-1700 … missing value` means the app refused the file.

Report which of these checks ran when you hand the file over; a check that could not run (no Apple app, not macOS) is named as skipped, not implied as passed.
