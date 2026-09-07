---
name: rngit
description: >
  Use when working with Git over Reticulum: rngit nodes, rns:// remotes,
  git-remote-rns, repository creation, forking, mirroring, releases,
  work documents and Reticulum commit signing.
---

# Git over Reticulum

The rngit system hosts Git repositories over Reticulum. It has two parts: the
rngit node that serves repositories, and the git-remote-rns helper that lets Git
use the rns:// URL scheme. Once RNS is installed, ordinary Git commands work with
Reticulum-hosted repositories the same way they work with any other remote.

This feature was introduced in RNS 1.2.0. It has not been tested extensively in
public or semi-public deployments, so be careful when hosting repositories.

## URLs

A Reticulum Git remote has this form:

    rns://DESTINATION_HASH/group/repo

DESTINATION_HASH is the repository node's destination hash. The path is
`group_name/repo_name`, where the group maps to a directory of bare repositories
on the node.

## The rngit utility

Run `rngit` to start a repository node. On the first run it creates a default
configuration file. Edit that file to point to repository locations, set access
permissions, and enable optional features such as the Nomad Network page node.

Command-line options:

    -h, --help            show help and exit
    --config CONFIG       path to alternative config directory
    --rnsconfig RNSCONFIG path to alternative Reticulum config directory
    -p, --print-identity  print identity and destination hashes and exit
    -s, --service         run as a service and log to file
    -i, --interactive     drop into an interactive shell after initialisation
    -v, --verbose         increase verbosity
    -q, --quiet           decrease verbosity
    --version             show program version and exit

`rngit --print-identity` prints the Git peer identity, repository node identity,
repositories destination, and the Nomad Network destination when the page node is
enabled.

`rngit -s` runs the node in service mode and writes logs to a file.

`rngit --config` changes the config directory. The default user install uses
`~/.rngit/config`. System installs use `/etc/rngit/config`.

`rngit --rnsconfig` changes the Reticulum config directory, normally
`~/.reticulum/config`.

## Client workflow

The git-remote-rns helper is invoked automatically by Git when it sees an rns://
URL. It is not normally run by hand. It reads these environment variables:

    RNGIT_CONFIG  path to alternative client config directory
    RNS_CONFIG    path to alternative Reticulum config directory

The client config lives at `~/.rngit/client_config` and can tune parameters such
as reference batch size for transfers.

Everyday Git commands work unchanged:

    git clone rns://<hash>/group/repo
    git remote add origin rns://<hash>/group/repo
    git push origin <branch>
    git pull origin <branch>

Set a branch to track a Reticulum remote as its default upstream, then use Git as
usual.

## Repository management

### Creating repositories

`rngit create rns://<hash>/group/repo` creates a new bare repository on the
remote node. The creator receives `adm` (admin) permissions automatically through
an auto-generated `.allowed` file. You must have create permission for the
target group.

Command-line options for `rngit create`:

    -h, --help            show help and exit
    --config CONFIG       path to alternative config directory
    --rnsconfig RNSCONFIG path to alternative Reticulum config directory
    -i, --identity PATH   path to identity
    -v, --verbose         increase verbosity
    -q, --quiet           decrease verbosity
    --version             show program version and exit

### Forking repositories

`rngit fork <source> <target>` copies an existing repository to your rngit node.
The source can be any valid Git URL, including https://, ssh:// or rns://.

Forks are created as bare repositories and store this metadata:

    repository.rngit.type = fork
    repository.rngit.upstream.source = <source_url>

Command-line options for `rngit fork`:

    -h, --help            show help and exit
    --config CONFIG       path to alternative config directory
    --rnsconfig RNSCONFIG path to alternative Reticulum config directory
    -i, --identity PATH   path to identity
    -v, --verbose         increase verbosity
    -q, --quiet           decrease verbosity
    --version             show program version and exit

### Mirroring repositories

`rngit mirror <source> <target>` creates a mirror that stays synchronised with an
upstream repository. Mirrors use the same source URL types as forks, but set
`repository.rngit.type` to `mirror` and add
`repository.rngit.upstream.sync`, a Unix timestamp of the last successful
synchronisation.

Command-line options for `rngit mirror` mirror those of `rngit fork`.

### Manual synchronisation

`rngit sync rns://<hash>/group/repo` fetches all refs from the configured
upstream. You must have read and write permissions for the repository. For
mirrors, a successful manual sync resets the automatic sync timer.

### Automatic mirror synchronisation

In the node config, set `mirror_interval` to a number of hours:

    [rngit]
    mirror_interval = 24

The default is 24 hours. The node checks for mirrors that need sync every 15
minutes and fetches updates when the configured interval has elapsed since the
last sync. The sync timestamp updates only on success. Failures are logged and
retried later.

## Repository structure

A node stores repositories under group directories. A repository at
`/var/git/public/myrepo` is served as `public/myrepo` via the URL
`rns://DESTINATION_HASH/public/myrepo`.

The node config at `~/.rngit/config` or `/etc/rngit/config` contains:

    [rngit]
    node_name = My Git Node
    announce_interval = 360
    record_stats = yes

    [repositories]
    public = /var/git/public
    internal = /var/git/internal

    [access]
    public = r:all, w:9710b86ba12c42d1d8f30f74fe509286
    internal = rw:9710b86ba12c42d1d8f30f74fe509286

    [pages]
    serve_nomadnet = yes
    unicode_icons = no

`[repositories]` maps group names to filesystem paths. `[access]` grants group
permissions. `[pages]` enables the optional Nomad Network page node.

## Permissions

By default, no permissions are granted for anything. Enable only the permissions
you need.

### Permission types

    r    read           clone, fetch, view repositories and work documents
    w    write          push changes and manage work documents
    rw   read/write     both read and write
    c    create         create, fork or mirror new repositories in a group
    s    stats          view repository activity statistics
    rel  release        create and manage releases
    i    interact       comment on and interact with work documents
    p    propose        propose new work documents without full write
    adm  admin          full access

### Targets

Permissions apply to one of these targets:

    all    everyone
    a      short form for all
    none   nobody
    n      short form for none
    HASH   a 32-character hexadecimal Reticulum identity hash

### Permission hierarchy

Resolution happens in this order:

1. Repository-level permissions, if present
2. Group-level permissions as fallback
3. Admin rights as final override

For work documents, document-specific permissions are checked first.

### Configuration methods

Group permissions can live in the main config file under `[access]`, or in a
`group_name.allowed` file placed next to the `group_name` directory.

Repository permissions are set in `.allowed` files placed next to the repository
directory, for example `myrepo.allowed` for `myrepo`.

A permission file looks like this:

    r:all
    w:9710b86ba12c42d1d8f30f74fe509286
    rel:9710b86ba12c42d1d8f30f74fe509286

### Dynamic permissions

Make a `.allowed` file executable, and the node runs it and parses stdout as
permission rules. This lets external authentication systems drive access
control.

### Work document permissions

Work documents use `.allowed` files in the work directory, for example
`42.allowed` for document 42. They support the permissions r, w, i, p and adm.
Document permissions override repository permissions for that document.

### Creator permissions

Creating a repository via create, fork or mirror grants the creator admin
permissions automatically. Creating a work document grants the creator interact
and write on that document.

### Permission examples

Public read and restricted write:

    r:all
    w:9710b86ba12c42d1d8f30f74fe509286

Collaborative development:

    r:all
    i:all
    p:all
    w:9710b86ba12c42d1d8f30f74fe509286
    rel:9710b86ba12c42d1d8f30f74fe509286

Private repository:

    rw:9710b86ba12c42d1d8f30f74fe509286
    rw:a1b2c3d4e5f686ba12c42d1ba12ef1aa

Mirror with stats:

    r:all
    s:all
    w:none

## Remote permission management

`rngit perms rns://<hash>/group` or `rngit perms rns://<hash>/group/repo`
retrieves the current `.allowed` file, opens it in the editor configured as
`$EDITOR`, and sends the changes back to the node. You must have `adm`
permissions on the target group or repository.

The remote editor validates each line for correct `permission:target` syntax,
valid permission types, valid target forms, and 32-character hexadecimal identity
hashes. Invalid rules reopen the editor with an error.

Granting `adm` to remote identities delegates full control, including the ability
to revoke your own access. Use it with care.

Command-line options for `rngit perms`:

    -h, --help            show help and exit
    --config CONFIG       path to alternative config directory
    --rnsconfig RNSCONFIG path to alternative Reticulum config directory
    -i, --identity PATH   path to identity
    -v, --verbose         increase verbosity
    -q, --quiet           decrease verbosity
    --version             show program version and exit

## Identity and destination aliases

Define aliases for commonly used identity and destination hashes to make
permissions and remotes easier to read.

Identity aliases used during permission resolution go in the `[aliases]` section
of `~/.rngit/config`:

    [aliases]
    alice = d09285e660cfe27cee6d9a0beb58b7e0
    bob = ffcffb4e255e156e77f79b82c13086a6

Destination aliases used by the client go in the `[aliases]` section of
`~/.rngit/client_config`:

    [aliases]
    bobs_node = 8a37cdd16938ce79861561adbd59023a
    my_node = 50824b711717f97c2fb1166ceddd5ea9

Aliases are resolved locally. A fork still records the full destination hash of
the upstream source for later synchronisation.

## Serving pages over Nomad Network

Enable the page node in `~/.rngit/config`:

    [pages]
    serve_nomadnet = yes

When enabled, `rngit --print-identity` also prints the Nomad Network
destination. Clients can connect to that destination and browse repository
information, file contents, commit history, refs and statistics.

Page node features:

- front page listing accessible groups
- group, repository, release, file, commit, refs and statistics pages
- automatic Markdown to Micron conversion
- syntax highlighting when pygments is installed
- customisable Micron templates in `~/.rngit/templates/`
- Nerd Font icons by default. Set `unicode_icons = yes` for plain Unicode
- a "Thanks" counter on each repository page

Install pygments for code syntax highlighting:

    pip install pygments

Micron documents and rawmu code blocks render as Micron inside Markdown files.

### Templates

The page node reads Micron templates from `~/.rngit/templates/`:

    base.mu    base template wrapping all pages
    front.mu   front page listing groups
    group.mu   group page listing repositories
    repo.mu    repository overview
    releases.mu and release.mu  release list and detail pages
    tree.mu    file browser
    blob.mu    file content display
    commits.mu and commit.mu  commit history and detail
    refs.mu    branches and tags
    stats.mu   repository statistics

Templates can include these variables:

    {PAGE_CONTENT}  main content (required)
    {NODE_NAME}     configured node name
    {NAVIGATION}    breadcrumb links
    {VERSION}       rngit version
    {GEN_TIME}      page generation time

Executable templates have their stdout used as the template content.

### Repository statistics

When `record_stats = yes` is set in the `[rngit]` section, the node tracks
views, fetches and pushes. Users need the `s` permission to see statistics.
Excluded identities go in `stats_ignore_identities`.

## Verified releases

The rngit release system signs every artifact with Ed25519 and embeds the
signatures in a signed release manifest (`.rsm`). Anyone who has the manifest can
verify artifacts offline and fetch future updates securely.

### Fetching releases

`rngit release rns://<hash>/group/repo fetch <version>:<artifact>` retrieves a
release. Use `latest:all` for the latest release and all artifacts. Use shell
wildcards for patterns:

    rngit release rns://<hash>/group/repo fetch "1.2.0:*-py3-*.whl"

Patterns use `*`, `?`, `[seq]` and `[!seq]`. Quote the argument so the shell does
not expand it.

You can also fetch from a local manifest:

    rngit release some_program_1.5.2.rsm fetch latest:all

Use `--signer <hash>` to require a specific signing identity. If a downloaded
artifact fails signature verification, the fetch aborts.

### Offline verification

Verify artifacts against a local `.rsm` manifest:

    rngit release myapp-1.2.0.rsm verify
    rngit release myapp-1.2.0.rsm verify "latest:*.whl"

The `verify` operation is equivalent to `fetch --offline`.

To verify a single file, ensure the `.rsg` signature is next to the artifact and
run:

    rnid -V myapp-1.2.0.tar.gz

### Creating releases

`rngit release rns://<hash>/group/repo create <version>:<artifacts_dir>` builds
and publishes a release:

1. Verifies the Git tag exists
2. Opens `$EDITOR` for release notes
3. Signs each artifact with the identity's Ed25519 key
4. Creates `.rsg` signature files next to each artifact
5. Builds a signed `manifest.rsm`
6. Uploads artifacts, signatures and manifest to the origin node

Use `--local` to build the release locally without uploading:

    rngit release rns://<hash>/group/repo create 1.2.0:./dist --local

Release manifests embed the signing identity's public key and a detached
signature. The manifest is named `manifest.rsm` in the artifacts directory.

### Release storage

Releases are stored in a directory named `repo_name.releases` next to the bare
repository. Each release is a subdirectory containing:

    META            release metadata in ConfigObj format
    RELEASE.md or RELEASE.mu  release notes
    artifacts/      uploaded files
    THANKS          appreciation count

### Release command-line options

    -h, --help            show help and exit
    --config CONFIG       path to alternative config directory
    --rnsconfig RNSCONFIG path to alternative Reticulum config directory
    -i, --identity PATH   path to release identity
    -s, --signer PATH     path to signing identity, if different from release identity
    -n, --name name       package name if different from repo name
    -L, --local           generate release locally, do not upload
    -o, --offline         verify manifest locally, do not fetch updates
    -v, --verbose         increase verbosity
    -q, --quiet           decrease verbosity
    --version             show program version and exit

### Release operations

    list    list releases
    view    show release details
    fetch   fetch and verify artifacts
    create  create a new release
    delete  remove a release
    latest  show latest release details

Creating a release requires the `rel` permission. The artifacts directory must
exist and contain at least one file, and release notes cannot be empty.

## Work documents

Work documents track tasks, investigations, issues and progress for a
repository. They are stored as structured msgpack data and support threaded
updates and comments.

### Operations

    list            list work documents
    view            view a document and its comments
    create          create a new work document
    propose         create a proposed document
    edit            edit an existing document
    update          add a comment
    complete        mark active as completed
    activate        mark completed as active
    delete          delete a document and its comments
    perms           manage document permissions

Use `--scope active`, `--scope completed`, `--scope proposed` or `--scope all`
with `list`.

### Creating and proposing

`rngit work rns://<hash>/group/repo create --title "Title"` opens `$EDITOR` for
the document body. Save an empty file to cancel.

`rngit work ... propose` creates a document in the proposed scope. Proposing
requires the propose permission. Activating a proposal requires write and interact permissions.

### State and permissions

The complete and activate commands move a document between the active and
completed states. The delete command removes a document and its comments.

Document permissions are managed with `rngit work ... perms -d <id>`. Document
permissions override repository permissions for that document.

### Cryptographic attribution

Every work document is signed with its creator's Ed25519 Reticulum identity. The
signature is stored with the content and verified when the document is viewed.
This gives attribution and integrity without a central authority.

### Work document command-line options

    -h, --help            show help and exit
    --config CONFIG       path to alternative config directory
    --rnsconfig RNSCONFIG path to alternative Reticulum config directory
    -i, --identity PATH   path to identity
    --scope SCOPE         active, completed, proposed or all
    -t, --title TITLE     document title for create or propose
    -d, --id ID           document ID
    -v, --verbose         increase verbosity
    -q, --quiet           decrease verbosity
    --version             show program version and exit

## Commit signing

`rngcs` is a Git commit signing and validation shim that uses Reticulum
identities. It hooks into Git's SSH-format signing support.

### Setup

Generate a Reticulum identity with `rnid`:

    rnid -g ~/.rngit/client_identity

Configure Git to use `rngcs` for SSH-format signatures:

    git config --global gpg.format ssh
    git config --global gpg.ssh.program rngcs
    git config --global gpg.ssh.allowedsignersfile none
    git config --global user.signingKey ~/.rngit/client_identity

`gpg.ssh.allowedsignersfile` must be set, but `rngcs` does not use it. Set it to
`none` or any value. All validation uses the embedded RSG data.

To enable signing only for one repository, use `--local` instead of `--global`.

### Author binding

The Git author email must match the Reticulum identity hash of the signing key:

    git config --global user.email "1a54d64db7e8beca6f2c6cd17b0cb479"

`rngcs` compares the Git author field with the signer identity from the RSG
signature. If they differ, verification fails.

### Sign and verify

Sign a commit as usual:

    git commit -S -m "Refactored module"

This produces an RSG signature wrapped in an SSH-style ASCII armor, stored in the
commit's signature header. It contains:

    SHA256 hash of the commit content
    signer's Reticulum identity hash
    signer's public key
    signature of the complete envelope

Verify with standard Git commands:

    git log --show-signature
    git show --show-signature

`rngcs` handles all verification. A `.mailmap` file can resolve identity hashes to
LXMF addresses for author display.

## Notes

- `rns://` URLs only resolve inside a running Reticulum network. Do not fabricate
them or try to fetch them over HTTP.
- `RNGIT_CONFIG` and `RNS_CONFIG` override default client and Reticulum config
paths.
- The client config at `~/.rngit/client_config` can adjust reference batch size
and destination aliases.
- The page node requires `serve_nomadnet = yes` in the `[pages]` section.
- Release manifests (`.rsm`) keep embedded signatures so artifacts can be
verified offline and updated from any source.
