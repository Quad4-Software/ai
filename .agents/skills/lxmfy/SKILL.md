---
name: lxmfy
description: >
  Use when working with LXMFy, the Python LXMF bot framework, or the
  lxmfy-mcp server: LXMFBot configuration, commands, cogs, external script
  cogs, NLP intents, RNS links, RRC hub clients, storage, permissions,
  signatures, Landlock sandboxing, templates, CLI tools, and the
  lxmfy.quad4.io documentation layout.
---

# LXMFy and lxmfy-mcp

LXMFy is a Python framework for building LXMF bots on the Reticulum
network. Docs live at https://lxmfy.quad4.io/ (English and Russian),
source at https://git.quad4.io/LXMFy/LXMFy. Author: Ivan. Docs built
with Sphinx and Furo.

## Install and requirements

- Python 3.11+, `pip install rns` (1.3.8+), `pip install lxmf` (1.0.1+,
  auto-installed), `cbor2` (auto-installed, required for RRC).
- `pip install lxmfy` or install from source.
- Dev commands upstream: `make typecheck` (pyright), `make ci` (lint,
  typecheck, security check, tests, build).

## The three doc pages

The site is small and fixed. lxmfy-mcp mirrors it as three topics:

- `quick-start` (quick-start.html): prerequisites, `lxmfy create`
  walkthrough, first commands, advanced feature overview.
- `creating-bots` (creating-bots.html): basic structure, templates,
  BotConfig, FIELD_COMMANDS, cogs, external script cogs, sovereign NLP,
  RNS links, message and event handlers, storage, permissions,
  signature verification, delivery retries and propagation, RRC.
- `api-reference` (api-reference.html): LXMFBot key methods, storage
  backends, commands, help system, events, testing, permissions,
  middleware, attachments, icon appearance field, scheduler,
  signatures, message delivery, message handlers, RRC API, templates,
  CLI tools, error handling.

## Minimal bot shape

```python
from lxmfy import LXMFBot

bot = LXMFBot(name="SimpleBot", command_prefix="!", storage_path="data")

@bot.command(name="ping", description="Responds with pong")
def ping_command(ctx):
    ctx.reply("Pong!")

if __name__ == "__main__":
    print(f"Bot LXMF Address: {bot.local.hash}")
    bot.run()
```

- `ctx.sender`, `ctx.content`, `ctx.args`, `ctx.reply(...)`.
- `ctx.reply` accepts `title=` and `lxmf_fields=`.
- Type-hinted command args auto-convert: `def add(ctx, a: int, b: int)`.
- `threaded=True` runs the callback in a thread. Threaded commands must
  not touch RNS or `lxmfy.transport` directly; use `ctx.reply()`.

## Short bot example

```python
from lxmfy import LXMFBot

bot = LXMFBot(
    name="NotesBot",
    command_prefix="!",
    storage_type="json",
    storage_path="data",
    first_message_enabled=True,
)

@bot.on_first_message()
def welcome(ctx):
    ctx.reply("Hi! Try !note save <text> or !note get.")

@bot.command(name="add", description="Add two numbers")
def add(ctx, a: int, b: int):
    ctx.reply(f"{a} + {b} = {a + b}")

@bot.command(name="note", description="Save or read a note")
def note(ctx, action: str, text: str = ""):
    if action == "save":
        bot.storage.set(f"note:{ctx.sender}", text)
        ctx.reply("Saved.")
    else:
        ctx.reply(bot.storage.get(f"note:{ctx.sender}", "No note yet."))

if __name__ == "__main__":
    bot.run()
```

## BotConfig highlights

Constructor kwargs seen in docs: `name`, `announce` (seconds),
`announce_immediately`, `announce_enabled`, `admins` (set of LXMF
hashes), `hot_reloading`, `rate_limit`, `cooldown`, `max_warnings`,
`warning_timeout`, `command_prefix`, `cogs_dir`, `cogs_enabled`,
`permissions_enabled`, `storage_type` ("json" | "sqlite" | "memory"),
`storage_path`, `first_message_enabled`, `event_logging_enabled`,
`max_logged_events`, `event_middleware_enabled`,
`signature_verification_enabled`, `require_message_signatures`,
`identity_pinning_enabled`, `message_persistence_enabled`,
`message_queue_size` (default 50, oldest dropped when full),
`dynamic_cogs_enabled`, `external_cogs_enabled`,
`external_cogs_sandbox_enabled`,
`external_cogs_sandbox_type` ("auto" | "landlock" | "bwrap" |
"firejail" | "none"), `external_cogs_timeout` (default 30s),
`landlock_enabled` (default True), `nlp_enabled`, `nlp_threshold`
(default 0.5), `link_support_enabled`, `lxmf_commands_enabled`,
`direct_delivery_retries` (default 3), `propagation_fallback_enabled`,
`reticulum_config_dir` (or `LXMFY_RETICULUM_CONFIG_DIR`), and the RRC
block: `rrc_enabled`, `rrc_hubs`, `rrc_rooms`, `rrc_nick`,
`rrc_dest_name` (default "rrc.hub"), `rrc_auto_reconnect`,
`rrc_persist_sessions`.

Key methods: `run(delay=10)`, `send(dest, msg, title=, lxmf_fields=,
stamp_cost=, opportunistic=)`, `send_with_attachment`, `command`,
`intent`, `request_link`, `on_link`, `load_extension`,
`reload_extension`, `add_cog`, `remove_cog`, `on_first_message`,
`on_message`, `validate`, `set_propagation_node`,
`get_landlock_status`, `connect_rrc`, `disconnect_rrc`, `on_rrc`,
`bot.rrc` (RRCManager), `bot.storage`, `bot.scheduler`,
`bot.middleware`, `bot.events`, `bot.nlp`, `bot.signature_manager`.

## Cogs

- Python cogs: file in `cogs_dir`, class (optionally `lxmfy.Cog`),
  `@Command(name=..., description=...)` methods taking `(self, ctx)`,
  and a required `setup(bot)` that calls `bot.add_cog(...)`.
- External script cogs: any executable non-.py file with a shebang in
  `cogs_dir` becomes a command. Args: `$1` sender hash, `$2` full
  content, `$3+` args. Env: `LXMFY_SENDER`, `LXMFY_CONTENT`,
  `LXMFY_HAS_ADMIN`. Stdout becomes the reply. Default 30s timeout,
  separate threads, sandboxed per `external_cogs_sandbox_type`
  ("auto" prefers Landlock, then bubblewrap, then firejail).

## Message handling order

1. First-message handler (`@bot.on_first_message()`, needs
   `first_message_enabled=True`).
2. General handlers (`@bot.on_message()`).
3. Command processing (prefix match, or `FIELD_COMMANDS`).

Handlers return True to stop processing, False to continue.

## Structured commands via LXMF fields

- `FIELD_COMMANDS` (0x09) in an incoming message routes
  `{"command": ..., "args": [...], "request_id": ...}` through the same
  command registry as text commands; replies automatically carry
  `FIELD_RESULTS` (0x0A) with the response and request_id.
- `ctx.fields` holds raw LXMF fields, `ctx.request_id` is set from the
  incoming field command.
- Helpers: `FIELD_COMMANDS`, `FIELD_RESULTS`, `pack_result`,
  `unpack_commands`. Disable with `lxmf_commands_enabled=False`.

## NLP intents

Local TF-IDF + cosine similarity intent matching, fully offline.
Enable with `nlp_enabled=True`, register with
`@bot.intent("name", examples=[...])`. Persist the model with
`bot.nlp.export_model()` / `bot.nlp.import_model(data)`.

## Security model

- LXMF signs all messages itself; LXMFy only enforces policy via
  `message.signature_validated` and `message.unverified_reason`.
  `signature_verification_enabled=True` logs failures;
  `require_message_signatures=True` rejects unsigned or invalid.
  `BYPASS_SPAM` permission bypasses verification.
- `identity_pinning_enabled=True` pins an LXMF address to its
  first-seen public key.
- Landlock LSM (Linux 5.13+): `landlock_enabled=True` sandboxes the bot
  process after startup, limiting writable paths to storage, config,
  cogs, Reticulum config, and temp. `LXMFY_LANDLOCK=0` disables,
  `LXMFY_LANDLOCK=1` forces. Inspect with `bot.get_landlock_status()`.
- External cogs get their own sandbox via `external_cogs_sandbox_type`.
- CLI: `lxmfy signatures test|enable|disable`.

## Delivery

- `direct_delivery_retries` (default 3) retries failed direct sends,
  counter resets on success.
- `bot.set_propagation_node("<hash>")` routes through an LXMF
  propagation node for store-and-forward.
- Outgoing queue persists by default (`message_persistence_enabled`),
  bounded by `message_queue_size`. Invalid destination hashes are not
  restored.

## RRC (Reticulum Relay Chat)

Bots join RRC hubs as normal clients over RNS Links with CBOR
envelopes. Compatible with NomadNet and rrcd hubs including MeshChatX.
Package `lxmfy.rrc`: `RRCClient`, `RRCManager` (`bot.rrc`),
`RRCMessage`, `RRC_VERSION`.

- Critical: the bot must use the same Reticulum config as the hub or
  hub announces never arrive (`Hub identity unknown`). Set
  `reticulum_config_dir="~/.reticulum"` or export
  `LXMFY_RETICULUM_CONFIG_DIR`, and keep rnsd or MeshChatX running.
- Quick start: `lxmfy run rrc`. Default hub
  `664fc0e8d2e448658e37bb3f34e6c88f`, room `#general`.
- Events to `@bot.on_rrc` handlers: `status`, `welcome`, `joined`,
  `parted`, `msg`, `notice`, `action`, `motd`, `error`, `rtt`.
  Handler signature `(event, client, payload)`; `payload` is an
  `RRCMessage` for `msg` events with `room`, `text`, `nick`, `src`,
  `mention`.
- Runtime API: `bot.connect_rrc(hash, rooms=[...])`,
  `bot.rrc.send_message(room, text)`, `send_notice`, `send_action`,
  `join`, `part`, `status`, `bot.disconnect_rrc()`.
- Envelope flow: HELLO/WELCOME, JOIN/PART, MSG/NOTICE/ACTION,
  PING/PONG, ERROR, RESOURCE_ENVELOPE. Auto-reconnect re-joins rooms
  after WELCOME. Sessions persist across restarts by default.

## RNS links

`link_support_enabled=True`, then `bot.request_link(dest_hash,
callback, "app_name", "aspect")` outbound and `bot.on_link(handler)`
for inbound. For stateful or high-bandwidth exchanges beyond message
packets.

## Templates and CLI

- `lxmfy create <name>` scaffolds `name.py`, `cogs/` (with
  `__init__.py` and a `basic.py` example), `data/`, `config/`.
- `lxmfy create --template echo|note|reminder|rrc|cogtest <name>`.
- `lxmfy run <template>` runs a template directly.
- Template classes in `lxmfy.templates`: `EchoBot`, `NoteBot` (JSON
  storage), `ReminderBot` (SQLite), `RRCBot` (hubs, rooms, nick,
  reticulum_config_dir kwargs; defaults to the public hub and
  `~/.reticulum`), `CogTestBot`.

## Icon and attachments

- `IconAppearance(icon_name=..., fg_color=3bytes, bg_color=3bytes)` +
  `pack_icon_appearance_field(icon)` produce an `lxmf_fields` dict
  using `LXMF.FIELD_ICON_APPEARANCE`; icon names come from Material
  Symbols. Combine fields with `{**icon_field, **other}`.
- `Attachment(type=AttachmentType.IMAGE, name=, data=, format=)` sent
  via `bot.send_with_attachment`.

## Scheduler, events, middleware, permissions

- `@bot.scheduler.schedule(name=..., cron_expr="0 0 * * *")`.
- `@bot.events.on("message_received", priority=EventPriority.X)`;
  `event.data`, `event.cancel()`; dispatch custom events with
  `bot.events.dispatch(Event(name, data={...}))`.
- `@bot.middleware.register(MiddlewareType.PRE_COMMAND)`.
- `permissions_enabled=True` plus `DefaultPerms` roles and flags such
  as `USE_COMMANDS`, `MANAGE_USERS`, `BYPASS_SPAM`; `admin_only=True`
  on commands and `ctx.is_admin`.

## Storage

`bot.storage` exposes `set`, `get(key, default)`, `exists`, `delete`,
`scan(prefix)`. Backends: `JSONStorage(dir)`, `SQLiteStorage(dbfile)`,
`MemoryStorage()` (volatile).

## Testing and reliability suite

Upstream test suite covers manifold testing of the NLP intent vector
space, chaos engineering (bit-rot, SD card failure, storage
corruption), temporal drift (clock jumps of a year), and leak
detection (memory, fds, threads). Run with the repo test runner.

## Tool reference

Tool details are in [references/tools.md](references/tools.md). This section is optional if lxmfy-mcp is installed.

## Conventions for lxmfy-related work

- This is a Python framework; the Go server only documents and
  scaffolds it. Keep generated Python consistent with upstream docs.
- Follow the Zen of Reticulum (see reticulum skill): bots are
  peers, keep outputs small, never fabricate mesh state.
- LXMFy is the rare write-capable piece of the stack (bots send
  messages, manage storage, run script cogs). Keep lxmfy-mcp itself
  read-only; scaffolding returns file contents, it does not write.
