#!/usr/bin/env python3
"""Convert a `quarry toc file` batch-mode JSON result into the compact TOC format:

    # <path>  (<package>)
    <optional one-line header sentence>
      <start>-<sigend>-<end>  [<Owner>.]<Name>  [<kind-abbrev>]

kind is abbreviated: function -> fun, method -> met, type -> typ.

Usage:
    quarry toc file --doc-sentences 0 --target-dir <repo> $(find <repo> -name '*.go') > raw.json
    gen_compact_toc.py raw.json [--header-mode none|first-sentence|full] > compact.txt
"""
import json
import re
import sys

KIND_ABBREV = {"function": "fun", "method": "met", "type": "typ"}


def first_sentence(text):
    if not text:
        return ""
    text = " ".join(text.split())
    m = re.match(r"(.{0,160}?[.!?])(\s|$)", text)
    return m.group(1) if m else text[:160]


def render(data, header_mode):
    out = []
    for r in data["results"]:
        if r.get("status") != "found":
            out.append(f"# {r.get('path')}  [error: {r.get('error', 'unknown')}]")
            continue
        pkg = r.get("package") or ""
        out.append(f"# {r['path']}  ({pkg})")
        if header_mode == "full" and r.get("header"):
            out.append(r["header"].strip())
        elif header_mode == "first-sentence" and r.get("header"):
            out.append(first_sentence(r["header"]))
        for s in r.get("symbols", []):
            kind = KIND_ABBREV.get(s["kind"], s["kind"])
            name = f"{s['owner']}.{s['name']}" if s.get("owner") else s["name"]
            sigend = s.get("sigend", s["end"])
            out.append(f"  {s['start']}-{sigend}-{s['end']}  {name}  [{kind}]")
    return "\n".join(out) + "\n"


if __name__ == "__main__":
    path = sys.argv[1]
    header_mode = "first-sentence"
    if "--header-mode" in sys.argv:
        header_mode = sys.argv[sys.argv.index("--header-mode") + 1]
    with open(path) as f:
        data = json.load(f)
    sys.stdout.write(render(data, header_mode))
