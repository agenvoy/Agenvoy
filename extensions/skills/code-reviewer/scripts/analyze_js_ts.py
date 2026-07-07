"""JavaScript/TypeScript analyzer. Combines a lightweight brace-based
structural scan (function boundaries + nesting depth) with project-local
eslint (when available) and string-level scans."""
import json
import re
import subprocess
from pathlib import Path

from common import (
    FunctionInfo,
    IGNORE_DIRS,
    Issue,
    ProjectAnalysis,
    detect_command_injection,
    detect_commented_code,
    detect_hardcoded_credentials,
    detect_sql_injection,
)

LONG_FUNCTION_THRESHOLD = 50
DEEP_NESTING_THRESHOLD = 3


_ESLINT_CANDIDATES = (
    "node_modules/.bin/eslint",
    "node_modules/.bin/eslint.cmd",
)

_ESLINT_SEVERITY_MAP = {
    2: "high",
    1: "medium",
}


def _find_eslint(root: Path) -> Path | None:
    for candidate in _ESLINT_CANDIDATES:
        p = root / candidate
        if p.exists() and p.is_file():
            return p
    return None


def _run_eslint(eslint: Path, root: Path) -> list[dict]:
    try:
        result = subprocess.run(
            [str(eslint), ".", "--format", "json"],
            cwd=str(root),
            capture_output=True,
            timeout=300,
            text=True,
        )
        if result.stdout.strip():
            return json.loads(result.stdout)
    except (FileNotFoundError, subprocess.TimeoutExpired, json.JSONDecodeError):
        pass
    return []


def _map_eslint_messages(eslint_data: list[dict], root: Path) -> list[Issue]:
    issues: list[Issue] = []
    for file_result in eslint_data:
        file_path = file_result.get("filePath", "")
        try:
            rel = str(Path(file_path).relative_to(root))
        except ValueError:
            rel = file_path
        for msg in file_result.get("messages", []):
            sev = _ESLINT_SEVERITY_MAP.get(msg.get("severity", 1), "low")
            rule = msg.get("ruleId") or "syntax"
            issues.append(Issue(
                severity=sev,
                category="quality",
                title=f"eslint: {rule}",
                description=msg.get("message", ""),
                file=rel,
                line=msg.get("line", 0),
                code_snippet=(msg.get("source") or "").strip()[:120],
                suggestion="依 eslint 規則建議修正",
            ))
    return issues


def _strip_noise(content: str) -> str:
    """Blank out comments/string/template literals, preserving line breaks
    and length so downstream regex + brace matching operate on code only."""
    out: list[str] = []
    i, n = 0, len(content)
    while i < n:
        c = content[i]
        if c == '/' and i + 1 < n and content[i + 1] == '/':
            j = content.find('\n', i)
            j = n if j == -1 else j
            out.append(' ' * (j - i))
            i = j
            continue
        if c == '/' and i + 1 < n and content[i + 1] == '*':
            j = content.find('*/', i + 2)
            j = n if j == -1 else j + 2
            out.append(''.join(ch if ch == '\n' else ' ' for ch in content[i:j]))
            i = j
            continue
        if c in ('"', "'", '`'):
            quote = c
            j = i + 1
            while j < n:
                if content[j] == '\\':
                    j += 2
                    continue
                if content[j] == quote:
                    j += 1
                    break
                j += 1
            out.append(''.join(ch if ch == '\n' else ' ' for ch in content[i:j]))
            i = j
            continue
        out.append(c)
        i += 1
    return ''.join(out)


_FUNC_PATTERNS: tuple[re.Pattern, ...] = (
    re.compile(r'\bfunction\s*\*?\s*(\w+)?\s*(?=\()'),
    re.compile(r'\b(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?function\b\s*\w*\s*(?=\()'),
)
_ARROW_PATTERN = re.compile(
    r'\b(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s*)?(?:\([^()]*\)|\w+)\s*'
    r'(?::\s*[^=]+?)?\s*=>'
)
_METHOD_PATTERN = re.compile(
    r'^[ \t]*(?:public\s+|private\s+|protected\s+|static\s+|async\s+|readonly\s+'
    r'|abstract\s+|override\s+|get\s+|set\s+)*\*?\s*(\w+)\s*\([^)]*\)\s*'
    r'(?::\s*[^{=;]+)?\{',
    re.MULTILINE,
)
_JS_RESERVED = frozenset({
    "if", "for", "while", "switch", "catch", "function", "return",
    "do", "else", "try", "finally", "with", "new",
})

_COND_KEYWORD_RE = re.compile(r'\b(if|for|while|switch)\s*\((?:[^()]|\([^()]*\))*\)\s*$')
_CATCH_RE = re.compile(r'\bcatch\s*(?:\([^()]*\))?\s*$')
_BARE_KEYWORD_RE = re.compile(r'\b(else|try|finally)\s*$')
_ARROW_OR_FUNC_RE = re.compile(r'(=>|\bfunction\b[^{]*|\bclass\b[^{]*)\s*$')


def _match_braces(text: str) -> dict[int, int]:
    stack: list[int] = []
    pairs: dict[int, int] = {}
    for i, ch in enumerate(text):
        if ch == '{':
            stack.append(i)
        elif ch == '}' and stack:
            pairs[stack.pop()] = i
    return pairs


def _find_body_brace(text: str, pos: int, limit: int = 2000) -> int | None:
    depth = 0
    end = min(len(text), pos + limit)
    i = pos
    while i < end:
        c = text[i]
        if c in '([':
            depth += 1
        elif c in ')]':
            depth -= 1
        elif c == '{' and depth <= 0:
            return i
        elif c == ';' and depth <= 0:
            return None
        i += 1
    return None


def _line_of(text: str, pos: int) -> int:
    return text.count('\n', 0, pos) + 1


def _extract_functions(
    text: str, rel: str, pairs: dict[int, int],
) -> list[tuple[FunctionInfo, int, int]]:
    claimed_lines: set[int] = set()
    results: list[tuple[FunctionInfo, int, int]] = []

    for pattern in (*_FUNC_PATTERNS, _ARROW_PATTERN):
        for m in pattern.finditer(text):
            line = _line_of(text, m.start())
            if line in claimed_lines:
                continue
            name = m.group(1) or "(anonymous)"
            if name in _JS_RESERVED:
                continue
            brace = _find_body_brace(text, m.end())
            if brace is None:
                continue
            close = pairs.get(brace)
            if close is None:
                continue
            claimed_lines.add(line)
            end_line = _line_of(text, close)
            results.append((FunctionInfo(
                name=name,
                signature=f"function {name}(...)",
                file=rel,
                line=line,
                line_count=end_line - line + 1,
                has_doc=False,
            ), brace, close))

    for m in _METHOD_PATTERN.finditer(text):
        line = _line_of(text, m.start())
        if line in claimed_lines:
            continue
        name = m.group(1)
        if name in _JS_RESERVED:
            continue
        brace = m.end() - 1
        close = pairs.get(brace)
        if close is None:
            continue
        claimed_lines.add(line)
        end_line = _line_of(text, close)
        results.append((FunctionInfo(
            name=name,
            signature=f"{name}(...)",
            file=rel,
            line=line,
            line_count=end_line - line + 1,
            has_doc=False,
        ), brace, close))

    return results


def _brace_kind(text: str, pos: int) -> str:
    window = text[max(0, pos - 300):pos]
    if _ARROW_OR_FUNC_RE.search(window):
        return "skip"
    if (_COND_KEYWORD_RE.search(window)
            or _CATCH_RE.search(window)
            or _BARE_KEYWORD_RE.search(window)):
        return "control"
    return "transparent"


def _scan_nesting(
    text: str, start: int, end: int, pairs: dict[int, int],
    current: int, best: list[int],
) -> None:
    i = start
    while i < end:
        if text[i] != '{':
            i += 1
            continue
        close = pairs.get(i)
        if close is None or close > end:
            i += 1
            continue
        kind = _brace_kind(text, i)
        if kind == "control":
            depth = current + 1
            if depth > best[0]:
                best[0] = depth
            _scan_nesting(text, i + 1, close, pairs, depth, best)
        elif kind == "transparent":
            _scan_nesting(text, i + 1, close, pairs, current, best)
        i = close + 1


def _check_function_length(info: FunctionInfo) -> Issue | None:
    if info.line_count <= LONG_FUNCTION_THRESHOLD:
        return None
    return Issue(
        severity="medium",
        category="quality",
        title="過長的函式",
        description=f"函式 '{info.name}' 有 {info.line_count} 行",
        file=info.file,
        line=info.line,
        suggestion="拆分為多個小函式，遵循單一職責原則",
    )


def _check_function_nesting(info: FunctionInfo, depth: int) -> Issue | None:
    if depth <= DEEP_NESTING_THRESHOLD:
        return None
    return Issue(
        severity="medium",
        category="quality",
        title="過深的巢狀結構",
        description=f"函式 '{info.name}' 巢狀深度 {depth} 層",
        file=info.file,
        line=info.line,
        suggestion="使用 early return 或抽出子函式降低巢狀深度",
    )


def _analyze_functions(text: str, rel: str, analysis: ProjectAnalysis) -> None:
    pairs = _match_braces(text)
    for info, brace, close in _extract_functions(text, rel, pairs):
        analysis.functions.append(info)
        if issue := _check_function_length(info):
            analysis.issues.append(issue)

        best = [0]
        _scan_nesting(text, brace + 1, close, pairs, 0, best)
        depth = best[0]
        if depth > analysis.metrics.max_nesting_depth:
            analysis.metrics.max_nesting_depth = depth
        if issue := _check_function_nesting(info, depth):
            analysis.issues.append(issue)


def _detect_language(root: Path) -> str:
    if (root / "tsconfig.json").exists():
        return "typescript"
    return "javascript"


def _parse_package_json(root: Path) -> tuple[str, list[str]]:
    pkg_json = root / "package.json"
    if not pkg_json.exists():
        return "", []
    try:
        data = json.loads(pkg_json.read_text())
    except (OSError, json.JSONDecodeError):
        return "", []
    name = data.get("name", "") or ""
    deps = list((data.get("dependencies") or {}).keys())
    return name, deps


def _iter_source_files(root: Path, lang: str):
    exts = (".ts", ".tsx") if lang == "typescript" else (".js", ".jsx", ".mjs", ".cjs")
    for ext in exts:
        for f in root.rglob(f"*{ext}"):
            if any(p in f.parts for p in IGNORE_DIRS):
                continue
            if f.name.endswith(".d.ts"):
                continue
            if any(suffix in f.name for suffix in (".spec.", ".test.")):
                continue
            yield f


def analyze(root: Path) -> ProjectAnalysis:
    lang = _detect_language(root)
    analysis = ProjectAnalysis(language=lang, name=root.name)

    name, deps = _parse_package_json(root)
    if name:
        analysis.name = name
    analysis.dependencies = deps

    total_lines = 0
    for src_file in _iter_source_files(root, lang):
        rel = str(src_file.relative_to(root))
        analysis.files.append(rel)

        try:
            content = src_file.read_text(encoding="utf-8")
        except (OSError, UnicodeDecodeError):
            continue

        total_lines += content.count('\n') + 1
        analysis.issues.extend(detect_hardcoded_credentials(content, rel))
        analysis.issues.extend(detect_sql_injection(content, rel))
        analysis.issues.extend(detect_command_injection(content, rel))
        analysis.issues.extend(detect_commented_code(content, rel, lang))
        _analyze_functions(_strip_noise(content), rel, analysis)

    analysis.metrics.total_lines = total_lines
    lengths = [f.line_count for f in analysis.functions]
    if lengths:
        analysis.metrics.avg_function_length = sum(lengths) / len(lengths)
        analysis.metrics.max_function_length = max(lengths)

    eslint = _find_eslint(root)
    if eslint is not None:
        eslint_data = _run_eslint(eslint, root)
        analysis.issues.extend(_map_eslint_messages(eslint_data, root))
    else:
        analysis.issues.append(Issue(
            severity="low",
            category="quality",
            title="eslint 不可用",
            description="未找到 node_modules/.bin/eslint；僅執行字串模式掃描",
            file="(system)",
            suggestion="於專案執行 `npm install eslint --save-dev` 並設定 eslint config 以啟用完整分析",
        ))

    return analysis
