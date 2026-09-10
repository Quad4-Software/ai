#!/usr/bin/env python3
# SPDX-License-Identifier: 0BSD
"""Check markdown links in docs and agent skills.

Scans every *.md file outside vendored/ignored trees, then verifies
relative file targets exist and HTTP(S) URLs respond. Fenced and inline
code spans are skipped, as are placeholders, loopback hosts, non-HTTP
schemes, and anchor-only links. Bot-blocked responses (401/403/405/429)
and gated hosts warn but do not fail.
"""

import concurrent.futures
import http.client
import re
import ssl
import sys
import urllib.parse
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
TIMEOUT = 20
MAX_REDIRECTS = 6
WORKERS = 8
USER_AGENT = "ai-link-check/1.0"

# Vendored trees are upstream-owned and never scanned.
SKIP_DIRS = {".git", ".github", "dist", "third_party", "node_modules"}

SKIP_HOSTS = {
    "localhost",
    "127.0.0.1",
    "0.0.0.0",
    "::1",
    "example.com",
    "example.org",
    "example.net",
}

# Hosts behind browser challenges: any 4xx warns instead of failing.
GATED_HOSTS = {"git.quad4.io"}

# Reachable-but-bot-blocked statuses warn instead of failing CI.
SOFT_STATUSES = {401, 403, 405, 406, 429, 451}

MD_LINK_RE = re.compile(r"!?\[[^\]]*\]\(\s*<?([^<>\s)]+)>?(?:\s[^)]*)?\)")
BARE_URL_RE = re.compile(r"https?://[^\s<>\"')\\]+")
INLINE_CODE_RE = re.compile(r"`[^`]*`")
FENCE_RE = re.compile(r"^\s*```")


def strip_trailing(url):
    while url and url[-1] in ".,;:!?\"'`":
        url = url[:-1]
    # Drop a closing paren or bracket that has no opener inside the URL.
    if url.endswith(")") and "(" not in url:
        url = url[:-1]
    if url.endswith("]") and "[" not in url:
        url = url[:-1]
    return url


def is_skippable(url):
    if "..." in url or "<" in url or ">" in url:
        return True
    parts = urllib.parse.urlparse(url)
    if parts.scheme not in ("http", "https"):
        return True
    host = parts.hostname
    if not host:
        return True
    if host in SKIP_HOSTS or host.endswith((".test", ".invalid", ".localhost")):
        return True
    return False


def request_once(url):
    """Single HTTP request, no redirect handling. Returns (status, location, err)."""
    parts = urllib.parse.urlparse(url)
    conn_cls = http.client.HTTPSConnection if parts.scheme == "https" else http.client.HTTPConnection
    port = parts.port or (443 if parts.scheme == "https" else 80)
    path = parts.path or "/"
    if parts.query:
        path += "?" + parts.query
    conn = conn_cls(parts.hostname, port, timeout=TIMEOUT)
    try:
        conn.request(
            "GET",
            path,
            headers={
                "User-Agent": USER_AGENT,
                "Accept": "text/html,application/xhtml+xml,*/*;q=0.8",
            },
        )
        resp = conn.getresponse()
        resp.read()
        return resp.status, resp.getheader("Location"), None
    except (OSError, http.client.HTTPException, ssl.SSLError) as e:
        return None, None, str(e)
    finally:
        conn.close()


def check_url(url):
    """Follow redirects manually. Returns (outcome, detail)."""
    gated = urllib.parse.urlparse(url).hostname in GATED_HOSTS
    for _ in range(MAX_REDIRECTS + 1):
        status, location, err = request_once(url)
        if err is not None:
            return "fail", err
        if 300 <= status < 400:
            if not location:
                return "ok", None
            url = urllib.parse.urljoin(url, location)
            continue
        if 200 <= status < 300:
            return "ok", None
        if gated and 400 <= status < 500:
            return "warn", f"HTTP {status} (gated host)"
        if status in SOFT_STATUSES:
            return "warn", f"HTTP {status}"
        return "fail", f"HTTP {status}"
    return "fail", "too many redirects"


def extract_links(path):
    """Yield (line_no, target, kind) for md links, images, and bare URLs."""
    links = []
    in_fence = False
    for line_no, raw_line in enumerate(
        path.read_text(encoding="utf-8").splitlines(), start=1
    ):
        if FENCE_RE.match(raw_line):
            in_fence = not in_fence
            continue
        if in_fence:
            continue
        line = INLINE_CODE_RE.sub(" ", raw_line)
        for m in MD_LINK_RE.finditer(line):
            raw = m.group(1)
            if "..." in raw:
                continue
            links.append((line_no, strip_trailing(raw), "md"))
        for m in BARE_URL_RE.finditer(line):
            raw = m.group(0)
            if "..." in raw:
                continue
            # A URL immediately followed by <...> is a template prefix.
            if m.end() < len(line) and line[m.end()] == "<":
                continue
            links.append((line_no, strip_trailing(raw), "url"))
    return links


def check_relative(base, target):
    """Resolve a markdown-relative link against the file's directory."""
    path = urllib.parse.unquote(urllib.parse.urlparse(target).path)
    if not path:
        return "skip", None
    if path.startswith("/"):
        dest = (REPO_ROOT / path.lstrip("/")).resolve()
    else:
        dest = (base.parent / path).resolve()
    try:
        dest.relative_to(REPO_ROOT)
    except ValueError:
        return "skip", None
    if dest.exists():
        return "ok", None
    return "fail", f"missing {dest.relative_to(REPO_ROOT)}"


def main():
    md_files = sorted(
        p
        for p in REPO_ROOT.rglob("*.md")
        if not any(part in SKIP_DIRS for part in p.relative_to(REPO_ROOT).parts)
    )
    failures = []
    warnings = []
    checked = skipped = 0
    http_targets = {}

    for md in md_files:
        rel = md.relative_to(REPO_ROOT)
        for line_no, target, kind in extract_links(md):
            if not target or target.startswith("#"):
                skipped += 1
                continue
            if kind == "md" and not re.match(r"^[a-zA-Z][a-zA-Z0-9+.-]*://", target):
                outcome, detail = check_relative(md, target)
                if outcome == "skip":
                    skipped += 1
                elif outcome == "ok":
                    checked += 1
                else:
                    failures.append(f"{rel}:{line_no}: {target} -> {detail}")
                continue
            if is_skippable(target):
                skipped += 1
                continue
            http_targets.setdefault(target, []).append((rel, line_no))

    with concurrent.futures.ThreadPoolExecutor(max_workers=WORKERS) as pool:
        future_to_url = {pool.submit(check_url, url): url for url in http_targets}
        for fut in concurrent.futures.as_completed(future_to_url):
            url = future_to_url[fut]
            outcome, detail = fut.result()
            checked += len(http_targets[url])
            if outcome == "ok":
                continue
            for rel, line_no in http_targets[url]:
                msg = f"{rel}:{line_no}: {url} -> {detail or outcome}"
                if outcome == "warn":
                    warnings.append(msg)
                else:
                    failures.append(msg)

    for w in sorted(warnings):
        print(f"WARN {w}")
    for f in sorted(failures):
        print(f"FAIL {f}")
    print(
        f"{checked} links checked, {skipped} skipped, "
        f"{len(warnings)} warnings, {len(failures)} failures "
        f"across {len(md_files)} markdown files"
    )
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
