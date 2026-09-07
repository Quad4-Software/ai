# Python bug and security checklist

Each entry: pattern, severity, confirmation oracle. Run `bandit`,
Semgrep `p/python`, and `pip-audit` alongside the scanners.

## Deserialization and eval

- `pickle.loads` / `pickle.load` / `dill` / `shelve` on anything that
  crossed a trust boundary (bandit B301, crit). Oracle: `__reduce__`
  gadget runs `os.system`.
- `yaml.load` without `Loader=yaml.SafeLoader` (B506, high).
- `eval`, `exec`, `compile` on input-derived strings (B307). Fix:
  `ast.literal_eval` for data, a real parser for anything else.
- `marshal`, `jsonpickle`, `cbor2.loads` on untrusted bytes.
- `input()` in Python 2 semantics, or `eval(input())` patterns.
- `getattr(obj, user_string)` / `setattr` with user-controlled attribute
  names: attribute injection, Python analog of prototype pollution.

## Command and code execution

- `subprocess.*` with `shell=True` (B602/B605, high), `os.system`,
  `os.popen`, `commands.getoutput`. Oracle: `;id` executes.
- `os.execl`/`os.spawnl` with formatted strings.
- `pty.spawn`, `code.InteractiveConsole` reachable from a handler.

## Injection

- SQL via f-strings, `%` formatting or `.format()` into
  `execute`/`executemany`/`raw()` (B608, high). Fix: parameterized
  queries; verify the driver uses real parameters, not client-side
  quoting.
- SSTI: `jinja2.Template(user_input)`, `render_template_string` with
  concatenation, `|safe` or `Markup()` on user data (B701). Oracle:
  `{{7*7}}` renders 49.
- LDAP, XPath, OS command strings built by concatenation.
- `open(os.path.join(base, user))` or `open(base + user)`: traversal.
  Oracle: `../../etc/passwd` resolves outside base.
- `tarfile`/`zipfile` `extractall` on untrusted archives: member names
  with `..` or absolute paths escape (CVE-2007-4559; 3.12+ `filter=`
  bypasses CVE-2025-4517, CVE-2025-4330). Check `filter="data"` AND a
  patched interpreter.
- `shutil.unpack_archive`/`unpack` on untrusted input.

## Crypto and TLS

- `hashlib.md5`/`sha1` for passwords or integrity (B303). Fix:
  `hashlib.sha256`, `scrypt`, `argon2`, `bcrypt`.
- `random.*` for tokens, OTPs, nonces, session IDs (B311). Fix:
  `secrets` module.
- `requests.*(verify=False)`, `ssl.CERT_NONE`, `check_hostname=False`
  (B501).
- `==` on secrets, tokens, signatures: timing oracle. Fix:
  `secrets.compare_digest` / `hmac.compare_digest`.
- JWT `decode` without `algorithms=`, or allowing `alg=none`.
- Hardcoded credentials: `password|api_key|secret|token = "..."` (B105).

## Logic and correctness

- Mutable default arguments: `def f(x=[])` / `x={}`. Oracle: call twice,
  state persists.
- Late-binding closures: `lambdas`/`defs` in loops capturing the loop
  variable. Fix: default-arg binding `lambda x=x: ...`.
- `assert` used for authorization or input validation (B101): stripped
  under `-O`.
- Bare `except:` or `except Exception: pass`: swallows the failure the
  test or guard was for.
- `==` vs `is` confusion for `None`/sentinels; `is` with ints/strings.
- `time.time()` or naive `datetime.now()` for expiry: clock skew, no
  monotonicity. Fix: `time.monotonic` for durations, timezone-aware
  `datetime.now(tz=timezone.utc)` for timestamps.
- `dict`/`set` iteration order assumptions in security decisions.
- `float` for money or permission arithmetic.
- `tempfile.mktemp`: predictable name, TOCTOU. Fix: `NamedTemporaryFile`.
- `os.environ` mutation leaking into child processes.
- XML: `xml.etree`, `lxml.etree` without `defusedxml` or
  `resolve_entities=False`: XXE (B320/B410).

## SSRF and HTTP

- `requests`/`urllib`/`httpx` on user-supplied URLs without scheme/host
  allowlists, redirect limits, and IP pinning (B310). Oracle: hit
  `169.254.169.254` or an internal host.
- `urllib.request.urlopen` on `file://` URLs.
