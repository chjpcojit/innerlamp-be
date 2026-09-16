#!/usr/bin/env python3
"""Conformance checker for OKF v0.2 bundles holding BDD documentation.

Usage:
    python3 okf_lint.py <bundle-dir> [--quiet] [--json]

Exits 1 if any ERROR is found, 0 otherwise. Warnings never fail the run:
OKF requires consumers to tolerate missing optional fields, unknown types,
unknown keys, and broken links, so those are advisory by design.

Dependency-free. Uses PyYAML for frontmatter parsing when it is installed,
otherwise falls back to a parser covering the YAML subset OKF frontmatter
actually uses (nested maps, block and flow sequences, flow mappings).
"""

import argparse
import json
import os
import re
import sys
from datetime import date

RESERVED = {"index.md", "log.md"}
VALID_STATUS = {"draft", "stable", "deprecated"}
ACTOR_RE = re.compile(r"^(human:.+|process:.+|[^/\s:]+/[^/\s]+)$")
ISO_DATE_RE = re.compile(r"^\d{4}-\d{2}-\d{2}$")
DATE_HEADING_RE = re.compile(r"^##\s+(.+?)\s*$")
FOOTNOTE_REF_RE = re.compile(r"\[\^([^\]]+)\]")
FOOTNOTE_DEF_RE = re.compile(r"^\[\^([^\]]+)\]:", re.M)
MD_LINK_RE = re.compile(r"\[[^\]]*\]\(([^)]+)\)")


# --------------------------------------------------------------------------
# Minimal YAML-subset parser (used when PyYAML is unavailable)
# --------------------------------------------------------------------------

def _strip_comment(line):
    out, quote = [], None
    for ch in line:
        if quote:
            out.append(ch)
            if ch == quote:
                quote = None
        elif ch in "\"'":
            quote = ch
            out.append(ch)
        elif ch == "#":
            break
        else:
            out.append(ch)
    return "".join(out).rstrip()


def _scalar(tok):
    tok = tok.strip()
    if not tok:
        return None
    if tok[0] == tok[-1] and tok[0] in "\"'" and len(tok) > 1:
        return tok[1:-1]
    low = tok.lower()
    if low in ("true", "false"):
        return low == "true"
    if low in ("null", "~"):
        return None
    try:
        return int(tok)
    except ValueError:
        pass
    try:
        return float(tok)
    except ValueError:
        pass
    return tok


def _split_flow(body):
    """Split a flow collection body on top-level commas."""
    parts, depth, quote, cur = [], 0, None, []
    for ch in body:
        if quote:
            cur.append(ch)
            if ch == quote:
                quote = None
            continue
        if ch in "\"'":
            quote = ch
            cur.append(ch)
        elif ch in "{[":
            depth += 1
            cur.append(ch)
        elif ch in "}]":
            depth -= 1
            cur.append(ch)
        elif ch == "," and depth == 0:
            parts.append("".join(cur))
            cur = []
        else:
            cur.append(ch)
    if "".join(cur).strip():
        parts.append("".join(cur))
    return parts


def _parse_flow(tok):
    tok = tok.strip()
    if tok.startswith("{") and tok.endswith("}"):
        result = {}
        for part in _split_flow(tok[1:-1]):
            if ":" not in part:
                continue
            k, v = part.split(":", 1)
            result[k.strip()] = _parse_value(v.strip())
        return result
    if tok.startswith("[") and tok.endswith("]"):
        return [_parse_value(p.strip()) for p in _split_flow(tok[1:-1])]
    return _scalar(tok)


def _parse_value(tok):
    tok = tok.strip()
    if tok.startswith("{") or tok.startswith("["):
        return _parse_flow(tok)
    return _scalar(tok)


def _indent(line):
    return len(line) - len(line.lstrip(" "))


def _parse_block(lines, i, indent):
    """Parse a block at the given indent. Returns (value, next_index)."""
    # Sequence?
    if i < len(lines) and lines[i].lstrip().startswith("- ") or (
        i < len(lines) and lines[i].strip() == "-"
    ):
        items = []
        while i < len(lines):
            line = lines[i]
            if not line.strip():
                i += 1
                continue
            if _indent(line) < indent or not line.lstrip().startswith("-"):
                break
            rest = line.lstrip()[1:].strip()
            item_indent = _indent(line) + 2
            if not rest:
                i += 1
                val, i = _parse_block(lines, i, item_indent)
                items.append(val)
            elif rest.startswith("{") or rest.startswith("["):
                items.append(_parse_flow(rest))
                i += 1
            elif ":" in rest and not rest.split(":", 1)[1].strip().startswith("//"):
                # Inline first key of a block mapping item.
                k, v = rest.split(":", 1)
                sub = {}
                if v.strip():
                    sub[k.strip()] = _parse_value(v)
                    i += 1
                else:
                    i += 1
                    nested, i = _parse_block(lines, i, item_indent + 1)
                    sub[k.strip()] = nested
                more, i = _parse_block(lines, i, item_indent)
                if isinstance(more, dict):
                    sub.update(more)
                items.append(sub)
            else:
                items.append(_scalar(rest))
                i += 1
        return items, i

    # Mapping
    result = {}
    while i < len(lines):
        line = lines[i]
        if not line.strip():
            i += 1
            continue
        cur = _indent(line)
        if cur < indent:
            break
        if line.lstrip().startswith("-"):
            break
        if ":" not in line:
            i += 1
            continue
        k, v = line.split(":", 1)
        key = k.strip()
        v = v.strip()
        if v:
            result[key] = _parse_value(v)
            i += 1
        else:
            i += 1
            child, i = _parse_block(lines, i, cur + 1)
            result[key] = child
    return result, i


def parse_yaml(text):
    """Parse the OKF frontmatter subset. Raises ValueError on malformed input."""
    try:
        import yaml  # type: ignore
        return yaml.safe_load(text) or {}
    except ImportError:
        pass
    lines = [_strip_comment(l) for l in text.split("\n")]
    lines = [l for l in lines if l.strip() != ""]
    if not lines:
        return {}
    value, _ = _parse_block(lines, 0, 0)
    if not isinstance(value, dict):
        raise ValueError("frontmatter is not a mapping")
    return value


# --------------------------------------------------------------------------
# Frontmatter extraction
# --------------------------------------------------------------------------

def split_frontmatter(text):
    """Return (frontmatter_text or None, body)."""
    if not text.startswith("---"):
        return None, text
    lines = text.split("\n")
    if lines[0].strip() != "---":
        return None, text
    for idx in range(1, len(lines)):
        if lines[idx].strip() == "---":
            return "\n".join(lines[1:idx]), "\n".join(lines[idx + 1:])
    return None, text


# --------------------------------------------------------------------------
# Checks
# --------------------------------------------------------------------------

class Report:
    def __init__(self):
        self.items = []

    def add(self, level, path, msg):
        self.items.append({"level": level, "path": path, "message": msg})

    def error(self, path, msg):
        self.add("ERROR", path, msg)

    def warn(self, path, msg):
        self.add("WARN", path, msg)

    @property
    def errors(self):
        return [i for i in self.items if i["level"] == "ERROR"]

    @property
    def warnings(self):
        return [i for i in self.items if i["level"] == "WARN"]


def check_actor(rep, rel, field, value):
    if not isinstance(value, str) or not ACTOR_RE.match(value.strip()):
        rep.warn(rel, f"{field}: {value!r} does not match the actor convention "
                      f"(human:<id>, process:<id>, or <producer>/<version>)")


def check_index(rep, rel, text, is_root):
    fm, _ = split_frontmatter(text)
    if fm is None:
        return
    if not is_root:
        rep.error(rel, "index.md must not contain frontmatter "
                       "(only a bundle-root index.md may, carrying okf_version)")
        return
    try:
        data = parse_yaml(fm)
    except ValueError as exc:
        rep.error(rel, f"unparseable frontmatter: {exc}")
        return
    extra = set(data) - {"okf_version"}
    if extra:
        rep.error(rel, "root index.md frontmatter may contain only okf_version; "
                       f"found {sorted(extra)}")


def check_log(rep, rel, text):
    fm, body = split_frontmatter(text)
    if fm is not None:
        rep.error(rel, "log.md must not contain frontmatter")
    dates = []
    for line in body.split("\n"):
        m = DATE_HEADING_RE.match(line)
        if not m:
            continue
        heading = m.group(1)
        if not ISO_DATE_RE.match(heading):
            rep.warn(rel, f"date heading '## {heading}' is not ISO YYYY-MM-DD")
        else:
            dates.append(heading)
    if dates and dates != sorted(dates, reverse=True):
        rep.warn(rel, "date headings are not newest-first")


def check_concept(rep, rel, text, bundle, all_paths):
    fm, body = split_frontmatter(text)
    if fm is None:
        rep.error(rel, "missing YAML frontmatter block delimited by --- lines")
        return
    try:
        data = parse_yaml(fm)
    except ValueError as exc:
        rep.error(rel, f"unparseable frontmatter: {exc}")
        return
    if not isinstance(data, dict):
        rep.error(rel, "frontmatter is not a mapping")
        return

    ctype = data.get("type")
    if not ctype or not str(ctype).strip():
        rep.error(rel, "missing required non-empty 'type'")

    for rec in ("title", "description"):
        if not data.get(rec):
            rep.warn(rel, f"no '{rec}' (recommended)")

    status = data.get("status")
    if status is not None and status not in VALID_STATUS:
        rep.warn(rel, f"status '{status}' is not one of {sorted(VALID_STATUS)}")

    stale = data.get("stale_after")
    if stale is not None:
        s = str(stale)
        if not ISO_DATE_RE.match(s):
            rep.warn(rel, f"stale_after '{s}' is not an absolute YYYY-MM-DD date")
        elif s < date.today().isoformat():
            rep.warn(rel, f"stale_after {s} has passed — content is stale")

    gen = data.get("generated")
    if isinstance(gen, dict):
        if not gen.get("by"):
            rep.warn(rel, "generated is present but has no 'by'")
        else:
            check_actor(rep, rel, "generated.by", gen["by"])

    ver = data.get("verified")
    if ver is not None:
        entries = [ver] if isinstance(ver, dict) else ver
        if not isinstance(entries, list):
            rep.warn(rel, "verified should be a mapping or a list of mappings")
        else:
            for e in entries:
                if isinstance(e, dict) and e.get("by"):
                    check_actor(rep, rel, "verified[].by", e["by"])
                else:
                    rep.warn(rel, f"verified entry {e!r} has no 'by'")

    source_ids = set()
    srcs = data.get("sources")
    if srcs is not None:
        if not isinstance(srcs, list):
            rep.warn(rel, "sources should be a list")
        else:
            for s in srcs:
                if not isinstance(s, dict):
                    rep.warn(rel, f"sources entry {s!r} is not a mapping")
                    continue
                if not s.get("resource"):
                    rep.warn(rel, f"sources entry {s.get('id', s)!r} has no "
                                  "'resource' (required within an entry)")
                if s.get("id"):
                    source_ids.add(str(s["id"]))
                if s.get("author"):
                    check_actor(rep, rel, "sources[].author", s["author"])
                lm = s.get("last_modified")
                if lm is not None and not ISO_DATE_RE.match(str(lm)):
                    rep.warn(rel, f"sources[].last_modified '{lm}' is not YYYY-MM-DD")

    # Footnote labels must resolve to a sources id.
    defined = set(FOOTNOTE_DEF_RE.findall(body))
    referenced = set(FOOTNOTE_REF_RE.findall(body)) - defined
    for label in sorted(referenced | defined):
        if label not in source_ids:
            rep.warn(rel, f"footnote [^{label}] does not match any sources[].id")
    for label in sorted(referenced - defined):
        rep.warn(rel, f"footnote [^{label}] is referenced but never defined")

    # BDD-specific: a Feature ought to contain at least one Gherkin fence.
    if str(ctype).strip().lower() == "feature":
        if "```gherkin" not in body:
            rep.warn(rel, "type: Feature with no ```gherkin fence — "
                          "a feature without scenarios is not yet a spec")
        if not re.search(r"^#\s+Narrative\s*$", body, re.M):
            rep.warn(rel, "type: Feature with no '# Narrative' heading")

    # Bundle-relative links.
    for target in MD_LINK_RE.findall(body):
        if not target.startswith("/"):
            continue
        t = target.split("#", 1)[0]
        if not t or t.endswith("/"):
            continue
        if t.lstrip("/") not in all_paths:
            rep.warn(rel, f"link {target} has no target in the bundle "
                          "(tolerated by OKF; write it or fix the path)")


def lint(bundle):
    rep = Report()
    bundle = os.path.abspath(bundle)
    if not os.path.isdir(bundle):
        rep.error(bundle, "not a directory")
        return rep

    md_files = []
    for root, dirs, files in os.walk(bundle):
        dirs[:] = [d for d in dirs if not d.startswith(".")]
        for f in sorted(files):
            if f.endswith(".md"):
                md_files.append(os.path.join(root, f))

    if not md_files:
        rep.error(bundle, "no .md files found")
        return rep

    all_paths = {os.path.relpath(p, bundle) for p in md_files}

    for path in md_files:
        rel = os.path.relpath(path, bundle)
        try:
            text = open(path, encoding="utf-8").read()
        except (OSError, UnicodeDecodeError) as exc:
            rep.error(rel, f"cannot read: {exc}")
            continue
        name = os.path.basename(path)
        if name == "index.md":
            check_index(rep, rel, text, is_root=(os.path.dirname(rel) == ""))
        elif name == "log.md":
            check_log(rep, rel, text)
        else:
            check_concept(rep, rel, text, bundle, all_paths)

    return rep


def main():
    ap = argparse.ArgumentParser(description="Lint an OKF v0.2 bundle.")
    ap.add_argument("bundle", help="path to the bundle directory")
    ap.add_argument("--quiet", action="store_true", help="show errors only")
    ap.add_argument("--json", action="store_true", dest="as_json",
                    help="emit machine-readable JSON")
    args = ap.parse_args()

    rep = lint(args.bundle)

    if args.as_json:
        print(json.dumps({
            "errors": rep.errors,
            "warnings": rep.warnings,
            "conformant": not rep.errors,
        }, indent=2))
        return 1 if rep.errors else 0

    shown = rep.errors if args.quiet else rep.items
    for item in shown:
        print(f"{item['level']:5} {item['path']}: {item['message']}")

    n_files = len({i["path"] for i in rep.items})
    if rep.errors:
        print(f"\nNOT CONFORMANT — {len(rep.errors)} error(s), "
              f"{len(rep.warnings)} warning(s) across {n_files} file(s).")
        return 1
    print(f"\nCONFORMANT with OKF v0.2 — 0 errors, {len(rep.warnings)} warning(s).")
    return 0


if __name__ == "__main__":
    sys.exit(main())
