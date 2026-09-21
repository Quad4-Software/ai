# SurrealDB security model

Docs root: https://surrealdb.com/docs/learn/security

## Four authentication methods

| Method | For | Defined with | Credential |
| --- | --- | --- | --- |
| System users | Operators and services administering the instance | DEFINE USER at root, namespace or database level | username + password |
| Record users | End users of your app, one record each | DEFINE ACCESS ... TYPE RECORD with SIGNUP/SIGNIN queries you write | whatever your SIGNIN query reads |
| JWT access | Clients an external IdP already authenticated | DEFINE ACCESS ... TYPE JWT holding a public key, JWKS URL or HMAC secret | token from the provider |
| Bearer access | Other systems, machine clients | DEFINE ACCESS ... TYPE BEARER plus ACCESS ... GRANT per client | the grant key |

- A record user is scoped to one database and bound by table and field
 permissions. A root system user bypasses permissions entirely and must
 never be embedded in an application.
- JWT HMAC algorithms (HS256 default, HS384, HS512) use one symmetric
 secret that both signs and verifies: anyone holding it can mint tokens
 with any claims. Prefer a public key or JWKS URL, which can only
 verify.
- Bearer grants can be audited (ACCESS ... SHOW) and revoked
 (ACCESS ... REVOKE) without touching the user they act as.

## Authorisation

- **Levels:** root > namespace > database. DEFINE USER/ACCESS at a level
 scopes the credential to everything beneath it.
- **Roles:** predefined roles on system users, plus custom roles.
- **Permissions:** PERMISSIONS clauses on tables and fields are
 SurrealQL WHERE-like expressions evaluated per request, giving
 row-level and field-level security. This is what makes direct
 frontend-to-database connections viable.
- **Tokens:** sign-in returns a token. Sessions and tokens expire
 independently. Token claims populate $session/$auth parameters.

## Server hardening flags

On `surreal start`, capability flags gate dangerous features:

- `--deny-funcs` / `--allow-funcs` - control which SurrealQL functions
 run (keep `http::*` off for agent or multi-tenant exposure)
- `--allow-net` / `--deny-net` - network egress allowlist for functions
- `--allow-origin` - CORS. Set explicitly, avoid `*` in production
- `--allow-guests` - unauthenticated access is off by default
- Auth itself: `--user`/`--pass` seed the root user on a new datastore.
 Prefer SURREAL_USER/SURREAL_PASS env vars over literals in commands or
 committed config.

## Signing in over HTTP

```bash
curl -X POST -H "Accept: application/json" \
  -d '{"user":"root","pass":"secret"}' \
  http://localhost:8000/signin
```

`POST /signup` creates a record user. Both return a bearer token for
later requests.

## Managed service posture

SurrealDB Cloud adds encryption in transit and at rest, audit logging,
and SOC 2 Type 2, ISO 27001, Cyber Essentials Plus and GDPR compliance.
