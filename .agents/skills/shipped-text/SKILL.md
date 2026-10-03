---
name: shipped-text
description: >
  Writes READMEs, docs, comments, log lines, and debug messages as
  product text only. Use whenever creating or editing README, docs,
  comments, log output, or debug messages. Forbids leaking chat,
  design debate, rejected ideas, or "we decided" rationale into
  shipped files.
---

# Shipped text

dont mention what we talk about in READMEs or docs or debug messages or comments

Those files describe the software as it is. They are not a record of the
chat, the prompt, rejected designs, or why an alternative was dropped.

## Applies to

- README and other docs
- code comments and godoc/docstrings
- log lines, errors, and debug messages
- commit messages only when they start explaining the conversation

## Do

- State behavior, flags, paths, formats, and how to run it
- Keep comments as go doc, python doc, or that language's doc form
- Keep debug output as facts the operator can act on

## Do not

- Mention the chat, the user, the agent, or "we discussed"
- Narrate design debate ("not X because anyone can extract Y")
- Contrast with a dropped idea the reader never sees
- Write "for extra security", "as an extra auth measure", or similar sales
- Restate the prompt in the file

## Examples

Wrong:

    A shared secret in the client is not used. Anyone can extract it.
    Access control is the pairing code instead.

Right:

    Clients must offer the lyra-sync-v1 subprotocol. The first binary
    frame is LYR1 plus an 8-byte group. Frames go only to that room.

Wrong:

    // We decided rooms are better than a hardcoded key

Right:

    // Room key is the 8-byte group after magic LYR1.
