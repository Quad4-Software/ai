# Web and protocol checklist

## JWT and tokens

- `jwt.decode` without an explicit `algorithms=` allowlist: alg
  confusion, RS256-public-key-as-HS256-secret is the classic bypass.
- `alg: none` accepted anywhere (crit).
- `kid` header used to pick a key file without sanitization: path
  traversal to a known-key file, or SQLi in key lookup.
- `jku`/`x5u` headers fetching keys from token-controlled URLs: SSRF
  plus key substitution.
- Sensitive data in payload treated as confidential: JWTs are signed,
  not encrypted.
- Expiry and `nbf` unchecked; `iss`/`aud` unvalidated.

## HTTP and request handling

- Request smuggling: hand-rolled HTTP parsing, proxies disagreeing on
  Content-Length vs Transfer-Encoding (Go net/http CVE-2025-22871,
  ASP.NET CVE-2025-55315). Do not write HTTP parsers; keep runtimes
  patched.
- Open redirects: Location headers built from request fields.
- Host header trust: `X-Forwarded-Host`, `Host` used in password-reset
  links or absolute redirects.
- CORS: `Access-Control-Allow-Origin` reflecting the request Origin with
  credentials allowed.
- Cache poisoning: unkeyed inputs reflected in cacheable responses.
- Response splitting: CR/LF in user data reaching headers.

## Prototype pollution and attribute injection

- JS: `__proto__`, `constructor`, `prototype` keys surviving JSON merge,
  `deep-set`, `Object.assign` recursion.
- Python: `getattr`/`setattr`/`__class__`/`__dict__` reached by user
  keys (docarray CVE-2025-5150 pattern).
- Fix shape: reject magic keys at the parse boundary, use Maps or
  dataclasses, never merge recursively into objects used for authz.

## Secrets in git

- `git_secrets` scans recent history; rotate anything it finds, deletion
  commits do not revoke.
- High-yield spots: `.env`, `config/*.yml`, `*.pem`, `id_rsa`,
  CI variables echoed in workflow logs, test fixtures.
- Redact values in reports; show enough to identify the credential.

## Rate and resource limits

- Unbounded body reads, unbounded `io.ReadAll(r.Body)`, no upload cap:
  memory DoS.
- No timeout on outbound calls: a slow upstream stalls your handler
  pool.
- Pagination without a max page size; `limit` parameters trusted
  uncapped.
- Regex over user input without length caps: ReDoS. Check nested
  quantifiers and alternation overlap.
