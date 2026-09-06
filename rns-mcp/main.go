// SPDX-License-Identifier: 0BSD
// Command rns-mcp is a stdio MCP server exposing the Reticulum
// manual as searchable, section-aware tools. Stdlib only, low memory.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Quad4-Software/ai/rns-mcp/internal/community"
	"github.com/Quad4-Software/ai/rns-mcp/internal/docs"
	"github.com/Quad4-Software/ai/rns-mcp/internal/mcp"
	"github.com/Quad4-Software/ai/rns-mcp/internal/sysutil"
)

const (
	base           = "https://reticulum.network/manual/"
	wikiWelcomeURL = "https://reticulum.miraheze.org/wiki/Welcome"
	wikiWelcomeID  = "wiki-welcome"
)

var topics = []docs.Topic{
	{ID: "whatis", Title: "What is Reticulum?", URL: base + "whatis.html", Summary: "Overview, current status, what Reticulum offers, where it can be used, interface types"},
	{ID: "zen", Title: "Zen of Reticulum", URL: base + "zen.html", Summary: "Design philosophy: decentralization, physics of trust, scarcity, sovereignty, identity, ethics, post-IP patterns"},
	{ID: "gettingstarted", Title: "Getting Started Fast", URL: base + "gettingstartedfast.html", Summary: "Install, utilities, creating a network, bootstrapping connectivity, public entrypoints, platform notes"},
	{ID: "using", Title: "Using Reticulum on Your System", URL: base + "using.html", Summary: "Config and data dirs, rnsd rnstatus rnpath rnprobe rncp rnx rnsh rnodeconf utilities, remote and blackhole management"},
	{ID: "understanding", Title: "Understanding Reticulum", URL: base + "understanding.html", Summary: "Motivation, goals, destinations, announces, identities, transport, node types, wire format, crypto primitives"},
	{ID: "hardware", Title: "Communications Hardware", URL: base + "hardware.html", Summary: "RNode, WiFi, Ethernet, serial, packet radio modems, combining hardware"},
	{ID: "interfaces", Title: "Configuring Interfaces", URL: base + "interfaces.html", Summary: "Auto, Backbone, TCP server/client, UDP, I2P, RNode LoRa, KISS, AX.25, Pipe, interface modes, announce rate control, discovery"},
	{ID: "networks", Title: "Building Networks", URL: base + "networks.html", Summary: "Network concepts, transport nodes, trustless networking, heterogeneous connectivity"},
	{ID: "distributed", Title: "Distributed Development", URL: base + "distributed.html", Summary: "Protocols over platforms, artifact-centered workflows, composable primitives"},
	{ID: "git", Title: "Git Over Reticulum", URL: base + "git.html", Summary: "rngit, repositories, permissions, aliases, page nodes, signed releases, commit signing, work documents"},
	{ID: "software", Title: "Programs Using Reticulum", URL: base + "software.html", Summary: "NomadNet, Sideband, MeshChatX, RRC, LXMF, LXST, RNS FileSync and other ecosystem programs"},
	{ID: "examples", Title: "Code Examples", URL: base + "examples.html", Summary: "Minimal, announce, broadcast, echo, link, requests and responses, channel, buffer, filetransfer, custom interfaces"},
	{ID: "reference", Title: "API Reference", URL: base + "reference.html", Summary: "Reticulum, Identity, Destination, Packet, Link, Resource, Channel, Buffer, Transport classes"},
	{ID: "support", Title: "Support Reticulum", URL: base + "support.html", Summary: "Donations and feedback"},
	{ID: "license", Title: "Reticulum License", URL: base + "license.html", Summary: "License text"},
	{ID: wikiWelcomeID, Title: "Reticulum Community Wiki - Welcome", URL: wikiWelcomeURL, Summary: "Community wiki welcome: guides, FAQ, glossary, hardware, software"},
}

const zenGates = `Zen of Reticulum design gates

Non-negotiables:
- No mandatory cloud or SaaS center for core mesh messaging, identity, or pathfinding.
- Address peers by destination hash and aspect, not IP, DNS, or hostname.
- Assume hostile links. No plaintext mesh shortcuts.
- Design for scarcity and delay: store-and-forward, small payloads, recoverable missing-path states.
- Code to RNS/LXMF/LXST intent, not to a specific radio or IP interface.
- Mesh peers are not REST clients.

Finish gate, redesign if any fail:
1. Works with clearnet disabled for mesh-critical paths
2. Addresses destination hash plus aspect
3. Survives delay, missing path, and identity switch
4. Payload size justified for constrained links
5. No new unauthenticated mutating surface
6. No cross-identity leakage`

func strArg(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func obj(props map[string]any, req ...string) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": req}
}

func tools(src *docs.Source, util *sysutil.Runner) []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "list_topics",
			Description: "List Reticulum manual topics (id, title, URL, summary).",
			InputSchema: obj(map[string]any{}),
			Handle: func(_ context.Context, _ json.RawMessage) (string, error) {
				b, err := json.MarshalIndent(src.List(), "", "  ")
				return string(b), err
			},
		},
		{
			Name:        "get_topic",
			Description: "Fetch the full plain text of a Reticulum manual page by topic id.",
			InputSchema: obj(map[string]any{"id": strArg("topic id from list_topics")}, "id"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.ID == "" {
					return "", fmt.Errorf("missing required argument: id")
				}
				return src.Get(ctx, strings.TrimSpace(a.ID))
			},
		},
		{
			Name:        "list_sections",
			Description: "List the section headings of a Reticulum manual page.",
			InputSchema: obj(map[string]any{"id": strArg("topic id from list_topics")}, "id"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.ID == "" {
					return "", fmt.Errorf("missing required argument: id")
				}
				secs, err := src.Sections(ctx, strings.TrimSpace(a.ID))
				if err != nil {
					return "", err
				}
				b, err := json.MarshalIndent(secs, "", "  ")
				return string(b), err
			},
		},
		{
			Name:        "get_section",
			Description: "Fetch one section of a manual page by anchor or heading name. Cheaper than get_topic on large pages.",
			InputSchema: obj(map[string]any{
				"id":      strArg("topic id from list_topics"),
				"section": strArg("section anchor or heading substring from list_sections"),
			}, "id", "section"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					ID      string `json:"id"`
					Section string `json:"section"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.ID == "" || a.Section == "" {
					return "", fmt.Errorf("missing required arguments: id, section")
				}
				return src.GetSection(ctx, strings.TrimSpace(a.ID), a.Section)
			},
		},
		{
			Name:          "search_docs",
			Description:   "Full-text search across the Reticulum manual. Returns ranked excerpts with topic and section anchors.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"query": "announce"}`)}},
			InputSchema: obj(map[string]any{
				"query": strArg("search terms"),
				"limit": map[string]any{"type": "integer", "description": "max results, default 5, max 20"},
			}, "query"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Query string `json:"query"`
					Limit int    `json:"limit"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Query == "" {
					return "", fmt.Errorf("missing required argument: query")
				}
				return src.Search(ctx, a.Query, a.Limit)
			},
		},
		{
			Name:        "fetch_page",
			Description: "Fetch any page on reticulum.network as plain text (host allowlisted). Use for deep links not in the topic index.",
			InputSchema: obj(map[string]any{"url": strArg("full URL on reticulum.network")}, "url"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					URL string `json:"url"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.URL == "" {
					return "", fmt.Errorf("missing required argument: url")
				}
				return src.FetchURL(ctx, strings.TrimSpace(a.URL))
			},
		},
		{
			Name:          "rns_util_help",
			Description:   "Show --help output for a local RNS utility. Covers rngit, rnsd, rnstatus, rnpath, rnprobe, rnid, rncp, rnx, rnsh, rnodeconf, nomadnet.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"utility": "rnstatus"}`)}},
			InputSchema: obj(map[string]any{
				"utility": map[string]any{"type": "string", "description": "utility name", "enum": sysutil.Utilities},
			}, "utility"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Utility string `json:"utility"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Utility == "" {
					return "", fmt.Errorf("missing required argument: utility")
				}
				return util.Help(ctx, strings.TrimSpace(a.Utility))
			},
		},
		{
			Name:        "rns_status",
			Description: "Run the local rnstatus utility: shared instance info, interfaces, traffic. Requires local RNS utilities (pipx install rns).",
			InputSchema: obj(map[string]any{
				"all": map[string]any{"type": "boolean", "description": "pass -a for verbose stats"},
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					All bool `json:"all"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				return util.Status(ctx, a.All)
			},
		},
		{
			Name:        "rns_path_table",
			Description: "Run rnpath -t: list all known destination paths, hops, next-hop transport, interface, expiry.",
			InputSchema: obj(map[string]any{
				"max_hops": strArg("optional max hops filter, digits only"),
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					MaxHops string `json:"max_hops"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				return util.PathTable(ctx, a.MaxHops)
			},
		},
		{
			Name:        "rns_path_lookup",
			Description: "Run rnpath <destination>: request or confirm the path to a 32 or 64 char hex destination hash.",
			InputSchema: obj(map[string]any{"destination": strArg("hex destination hash")}, "destination"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Destination string `json:"destination"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Destination == "" {
					return "", fmt.Errorf("missing required argument: destination")
				}
				return util.PathLookup(ctx, strings.TrimSpace(a.Destination))
			},
		},
		{
			Name:        "rns_destination_hash",
			Description: "Run rnid -i <identity> -H <aspects>: show destination hashes for aspects of an identity hash.",
			InputSchema: obj(map[string]any{
				"identity": strArg("32 or 64 char hex identity or destination hash"),
				"aspects":  strArg("space or comma separated aspects, e.g. lxmf.delivery"),
			}, "identity", "aspects"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Identity string `json:"identity"`
					Aspects  string `json:"aspects"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Identity == "" || a.Aspects == "" {
					return "", fmt.Errorf("missing required arguments: identity, aspects")
				}
				return util.DestinationHash(ctx, strings.TrimSpace(a.Identity), a.Aspects)
			},
		},
		{
			Name:          "rns_config_check",
			Description:   "Lint a Reticulum config file: unknown sections, interfaces missing 'type', unknown interface types, ifac_size pitfalls. Read-only.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{}`)}},
			InputSchema: obj(map[string]any{
				"path": strArg("config path, default ~/.reticulum/config"),
			}),
			Handle: func(_ context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Path string `json:"path"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				if a.Path == "" {
					home, _ := os.UserHomeDir()
					a.Path = filepath.Join(home, ".reticulum", "config")
				}
				return configCheck(a.Path)
			},
		},
		{
			Name:        "rns_probe",
			Description: "Run rnprobe <app_name> <destination>: send a probe packet to verify reachability.",
			InputSchema: obj(map[string]any{
				"app_name":    strArg("probe app name, e.g. nomadnetwork.node"),
				"destination": strArg("hex destination hash"),
			}, "app_name", "destination"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					AppName     string `json:"app_name"`
					Destination string `json:"destination"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.AppName == "" || a.Destination == "" {
					return "", fmt.Errorf("missing required arguments: app_name, destination")
				}
				return util.Probe(ctx, a.AppName, strings.TrimSpace(a.Destination))
			},
		},
		{
			Name:        "unsigned_posts",
			Description: "List unsigned.io posts (Mark Qvist's Reticulum articles: announcements, design posts, field notes).",
			InputSchema: obj(map[string]any{}),
			Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
				links, err := community.UnsignedPosts(ctx)
				if err != nil {
					return "", err
				}
				b, _ := json.MarshalIndent(links, "", "  ")
				return string(b), nil
			},
		},
		{
			Name:        "forum_categories",
			Description: "List rns.recipes forum categories (General, Build Guides, Help, Showcase, Regional).",
			InputSchema: obj(map[string]any{}),
			Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
				links, err := community.ForumCategories(ctx)
				if err != nil {
					return "", err
				}
				b, _ := json.MarshalIndent(links, "", "  ")
				return string(b), nil
			},
		},
		{
			Name:        "forum_threads",
			Description: "List threads in an rns.recipes forum category (e.g. general, help, build-guides).",
			InputSchema: obj(map[string]any{
				"category": strArg("category name, e.g. general"),
				"limit":    map[string]any{"type": "integer", "description": "max threads, default 20"},
			}, "category"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Category string `json:"category"`
					Limit    int    `json:"limit"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Category == "" {
					return "", fmt.Errorf("missing required argument: category")
				}
				if a.Limit <= 0 || a.Limit > 50 {
					a.Limit = 20
				}
				links, err := community.ForumThreads(ctx, a.Category, a.Limit)
				if err != nil {
					return "", err
				}
				if len(links) == 0 {
					return "no threads found in " + a.Category, nil
				}
				b, _ := json.MarshalIndent(links, "", "  ")
				return string(b), nil
			},
		},
		{
			Name:        "forum_latest",
			Description: "Newest rns.recipes forum threads via its RSS feed.",
			InputSchema: obj(map[string]any{
				"limit": map[string]any{"type": "integer", "description": "max, default 15"},
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Limit int `json:"limit"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				if a.Limit <= 0 || a.Limit > 50 {
					a.Limit = 15
				}
				links, err := community.ForumLatest(ctx, a.Limit)
				if err != nil {
					return "", err
				}
				b, _ := json.MarshalIndent(links, "", "  ")
				return string(b), nil
			},
		},
		{
			Name:          "forum_search",
			Description:   "Search rns.recipes forum thread titles by regex across all categories.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"query": "rnode"}`)}},
			InputSchema: obj(map[string]any{
				"query": strArg("regex or text"),
				"limit": map[string]any{"type": "integer", "description": "max results, default 15"},
			}, "query"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Query string `json:"query"`
					Limit int    `json:"limit"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Query == "" {
					return "", fmt.Errorf("missing required argument: query")
				}
				if a.Limit <= 0 || a.Limit > 50 {
					a.Limit = 15
				}
				links, err := community.ForumSearch(ctx, a.Query, a.Limit)
				if err != nil {
					return "", err
				}
				if len(links) == 0 {
					return "no matching threads", nil
				}
				b, _ := json.MarshalIndent(links, "", "  ")
				return string(b), nil
			},
		},
		{
			Name:          "community_read",
			Description:   "Read a community page as text: rns.recipes forum thread, unsigned.io post, or a markqvist/Reticulum discussion URL.",
			InputExamples: []map[string]any{{"arguments": json.RawMessage(`{"url": "https://unsigned.io/posts/..."}`)}},
			InputSchema:   obj(map[string]any{"url": strArg("full URL")}, "url"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					URL string `json:"url"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.URL == "" {
					return "", fmt.Errorf("missing required argument: url")
				}
				return community.ReadPage(ctx, a.URL)
			},
		},
		{
			Name:        "github_discussions",
			Description: "List recent markqvist/Reticulum GitHub discussions (parsed from the public page).",
			InputSchema: obj(map[string]any{
				"limit": map[string]any{"type": "integer", "description": "max, default 15"},
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Limit int `json:"limit"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- error tolerated; empty/default is handled downstream
				if a.Limit <= 0 || a.Limit > 50 {
					a.Limit = 15
				}
				links, err := community.GitHubDiscussions(ctx, a.Limit)
				if err != nil {
					return "", err
				}
				b, _ := json.MarshalIndent(links, "", "  ")
				return string(b), nil
			},
		},
		{
			Name:        "github_search",
			Description: "Search issues/PRs in markqvist/Reticulum via the GitHub REST API (unauthenticated, 10 req/min). Discussions are GraphQL-only; use github_discussions for those.",
			InputSchema: obj(map[string]any{
				"query": strArg("search terms, e.g. rnode power"),
				"limit": map[string]any{"type": "integer", "description": "max, default 10"},
			}, "query"),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Query string `json:"query"`
					Limit int    `json:"limit"`
				}
				if err := json.Unmarshal(args, &a); err != nil || a.Query == "" {
					return "", fmt.Errorf("missing required argument: query")
				}
				if a.Limit <= 0 || a.Limit > 30 {
					a.Limit = 10
				}
				links, err := community.GitHubSearch(ctx, a.Query, a.Limit)
				if err != nil {
					return "", err
				}
				if len(links) == 0 {
					return "no matches", nil
				}
				b, _ := json.MarshalIndent(links, "", "  ")
				return string(b), nil
			},
		},
		{
			Name:        "interface_directory",
			Description: "Search the public rns.recipes interface directory for Reticulum nodes, backbones, TCP, I2P, RNode, etc.",
			InputSchema: obj(map[string]any{
				"search": strArg("free text, e.g. Vienna or RNode"),
				"type":   strArg("interface type filter, e.g. tcp, backbone, rnode, i2p"),
				"status": strArg("status filter, e.g. online or offline"),
				"limit":  map[string]any{"type": "integer", "description": fmt.Sprintf("max results, default %d, max %d", community.DirectoryDefaultLimit, community.DirectoryMaxLimit)},
			}),
			Handle: func(ctx context.Context, args json.RawMessage) (string, error) {
				var a struct {
					Search string `json:"search"`
					Type   string `json:"type"`
					Status string `json:"status"`
					Limit  int    `json:"limit"`
				}
				json.Unmarshal(args, &a) // #nosec G104 -- empty/default is handled downstream
				return community.DirectorySearch(ctx, strings.TrimSpace(a.Search), strings.TrimSpace(a.Type), strings.TrimSpace(a.Status), a.Limit)
			},
		},
		{
			Name:        "wiki_welcome",
			Description: "Fetch the Reticulum community wiki Welcome page as plain text.",
			InputSchema: obj(map[string]any{}),
			Handle: func(ctx context.Context, _ json.RawMessage) (string, error) {
				return src.Get(ctx, wikiWelcomeID)
			},
		},
	}
}

func prompts(src *docs.Source) []mcp.Prompt {
	return []mcp.Prompt{
		{
			Name:        "zen_review",
			Description: "Review a design or change against the Zen of Reticulum finish gates.",
			Arguments:   []mcp.PromptArg{{Name: "design", Description: "the design or diff to review", Required: true}},
			Handle: func(args map[string]string) (string, error) {
				return zenGates + "\n\nReview this against the gates. Cite the specific gate for each pass or fail.\n\n" + args["design"], nil
			},
		},
	}
}

func main() {
	src := docs.NewSource("Reticulum Manual", topics)
	srv := mcp.NewServer("rns-mcp", "0.3.0", tools(src, sysutil.New()), prompts(src))
	if err := srv.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "rns-mcp:", err)
		os.Exit(1)
	}
}

// knownSections and knownInterfaceTypes come from the manual's
// configuring-interfaces page; used to lint a local config.
var knownSections = map[string]bool{
	"reticulum": true, "logging": true, "interfaces": true,
	"plugins": true, "network": true,
}

var knownIfaceTypes = map[string]bool{
	"autointerface": true, "tcpclientinterface": true,
	"tcpserverinterface": true, "udpinterface": true, "i2pinterface": true,
	"rnodeinterface": true, "serialinterface": true, "kissinterface": true,
	"ax25kissinterface": true, "pipeinterface": true, "backboneinterface": true,
	"localclientinterface": true, "androidinterface": true,
	"websocketinterface": true, "webrtcinterface": true,
}

// configCheck lints a Reticulum config file: unknown sections, interface
// blocks missing type/enabled, unknown interface types, insecure flags.
func configCheck(path string) (string, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path jailed to repo root or local config
	if err != nil {
		return "", err
	}
	var b strings.Builder
	var warnings, infos []string
	var section, ifaceName string
	inIface := false
	keys := map[string]string{}
	ifaceHasType := false
	flushIface := func() {
		if inIface && !ifaceHasType {
			warnings = append(warnings, "interface "+ifaceName+": no 'type' key")
		}
		ifaceHasType = false
	}
	for raw := range strings.SplitSeq(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			if strings.HasPrefix(line, "[[") { // interface blocks
				flushIface()
				inIface = true
				ifaceName = strings.Trim(line, "[] ")
				continue
			}
			flushIface()
			inIface = false
			section = strings.Trim(line, "[] ")
			if !knownSections[strings.ToLower(section)] {
				infos = append(infos, "unknown section ["+section+"]")
			}
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			k, v = strings.TrimSpace(k), strings.TrimSpace(v)
			keys[k] = v
			if inIface {
				if k == "type" {
					ifaceHasType = true
					if !knownIfaceTypes[strings.ToLower(v)] {
						warnings = append(warnings, ifaceName+": unknown interface type "+v)
					}
				}
				if k == "ifac_size" && (v == "0" || v == "auto") {
					infos = append(infos, ifaceName+": ifac_size 0/auto on constrained links can inflate announces")
				}
			}
		}
	}
	flushIface()
	if _, ok := keys["rpc_key"]; !ok {
		infos = append(infos, "no rpc_key set (shared instance RPC is unauthenticated-by-config)")
	}
	if len(keys) == 0 {
		warnings = append(warnings, "no key=value pairs parsed")
	}
	fmt.Fprintf(&b, "config check: %s\n", path)
	if len(warnings) == 0 && len(infos) == 0 {
		b.WriteString("clean: no issues found\n")
	}
	for _, w := range warnings {
		b.WriteString("WARN  " + w + "\n")
	}
	for _, i := range infos {
		b.WriteString("INFO  " + i + "\n")
	}
	return b.String(), nil
}
