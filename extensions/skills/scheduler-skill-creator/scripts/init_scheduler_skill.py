#!/usr/bin/env python3
"""
Scheduler Skill Initializer - Creates a new scheduler-triggered skill from template.

Usage:
    init_scheduler_skill.py <short-name>

Examples:
    init_scheduler_skill.py daily-hn-digest
    init_scheduler_skill.py meeting-reminder

Output:
    ~/.config/agenvoy/skills/scheduler/<short-name>/SKILL.md

Notes:
    - <short-name> is normalized to lowercase, hyphen-case.
    - No 'scheduler-' prefix anywhere: dir, frontmatter name, and add_task /
      add_cron skill_name argument all use the bare short name.
    - If the skill directory already exists, the script prints '[OK] already
      exists' and exits 0 (idempotent), so re-running for re-binding is safe.
"""

import argparse
import re
import sys
from pathlib import Path

MAX_NAME_LENGTH = 64
ROOT = Path.home() / ".config" / "agenvoy" / "skills" / "scheduler"

TEMPLATE = """---
name: {name}
description: [TODO: 一句話描述何時觸發、做什麼。例：抓取 X、提醒 Y、彙整 Z]
---

# {title}

## 任務

[TODO: 描述被觸發時要做的具體行為。
- 引用要呼叫的 tool 名稱與必要參數
- 步驟以祈使式列出
- 不假設對話上下文（subagent session 從零開始）]

## 輸出格式

[TODO: 期望輸出形式。
- 例：1 行總結 + 條列 5 筆 + 結尾時間戳
- 例：JSON object with fields ...]
"""


def normalize(name: str) -> str:
    s = name.strip().lower()
    s = re.sub(r"[^a-z0-9]+", "-", s)
    s = re.sub(r"-{2,}", "-", s).strip("-")
    return s


def title_case(name: str) -> str:
    return " ".join(word.capitalize() for word in name.split("-"))


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Initialize a scheduler-triggered skill directory.",
    )
    parser.add_argument(
        "short_name",
        help="Short name (kebab-case). No 'scheduler-' prefix.",
    )
    args = parser.parse_args()

    raw = args.short_name
    short = normalize(raw)
    if not short:
        print("[ERROR] short name must contain at least one letter or digit", file=sys.stderr)
        return 1
    if len(short) > MAX_NAME_LENGTH:
        print(
            f"[ERROR] short name '{short}' too long ({len(short)} > {MAX_NAME_LENGTH})",
            file=sys.stderr,
        )
        return 1

    skill_dir = ROOT / short
    skill_md = skill_dir / "SKILL.md"

    if raw != short:
        print(f"note: normalized '{raw}' -> '{short}'")

    if skill_md.exists():
        print(f"[OK] already exists: {skill_md}")
        print(f"[OK] skill name    : {short}")
        print()
        print("Skill is ready. If body needs change, edit SKILL.md directly.")
        print("Otherwise skip to:")
        print("  add_task(time, skill_name='{}') or".format(short))
        print("  add_cron(time, skill_name='{}')".format(short))
        return 0

    skill_dir.mkdir(parents=True, exist_ok=True)
    skill_md.write_text(TEMPLATE.format(name=short, title=title_case(short)))

    print(f"[OK] created   : {skill_md}")
    print(f"[OK] skill name: {short}")
    print()
    print("Next: edit SKILL.md to fill in TODOs (description + body).")
    print("Then: add_task(time, skill_name='{}') or".format(short))
    print("      add_cron(time, skill_name='{}')".format(short))
    return 0


if __name__ == "__main__":
    sys.exit(main())
