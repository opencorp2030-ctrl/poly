// Poly's MCP server, exposed as a Supabase Edge Function so AI assistants
// (Claude, ChatGPT, or anything else that speaks MCP) can publish and
// browse community packages "as" a signed-in Poly user, without ever
// handling that user's email/password.
//
// Two ways in:
//  1. A raw `Authorization: Bearer <poly_pat_...>` header -- works with
//     Claude Desktop's manually-edited config file (which lets you set
//     arbitrary headers on a remote MCP server entry).
//  2. OAuth 2.1 + Dynamic Client Registration -- required by Claude.ai's
//     and ChatGPT's polished "Add custom connector" UI, which only
//     knows how to talk to remote MCP servers via OAuth, not a pasted
//     static token. The "authorization" step here is just the user
//     pasting their existing Poly personal access token into a small
//     form; the OAuth access_token these clients end up holding on the
//     other side of the dance IS that same Poly PAT, single-use code in
//     between. No separate trust model, no separate revocation path --
//     revoking the token on account still kills it everywhere.
//
// Auth resolution for tool calls happens entirely inside Postgres, via
// the mcp_* SECURITY DEFINER functions (see migrations
// add_api_tokens_and_mcp_functions / add_mcp_oauth_codes) -- this
// function never sees a Supabase session JWT, so verify_jwt is disabled
// and everything above is this file's own custom auth.
//
// Transport: MCP "Streamable HTTP", stateless mode -- every JSON-RPC
// request is a self-contained POST, no server-side session state.

const SUPABASE_URL = Deno.env.get("SUPABASE_URL")!;
const ANON_KEY = Deno.env.get("SUPABASE_ANON_KEY")!;
const SERVICE_ROLE_KEY = Deno.env.get("SUPABASE_SERVICE_ROLE_KEY")!;

const CORS_HEADERS = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Headers": "authorization, content-type, mcp-session-id, mcp-protocol-version",
  "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
};

// ---------------------------------------------------------------------
// MCP tool definitions + JSON-RPC handling (unchanged from the
// bearer-token-only version)
// ---------------------------------------------------------------------

const TOOLS = [
  {
    name: "whoami",
    description: "Return the signed-in Poly account this token belongs to: username, email, plan, badges, and whether it's the official account.",
    inputSchema: { type: "object", properties: {}, additionalProperties: false },
  },
  {
    name: "search_packages",
    description: "Search Poly's community package registry by (partial) name. Use this before publishing to check whether a name is already taken.",
    inputSchema: {
      type: "object",
      properties: { query: { type: "string", description: "Package name or partial name to search for" } },
      required: ["query"],
      additionalProperties: false,
    },
  },
  {
    name: "search_dependencies",
    description:
      "Search across every ecosystem poly can install from at once (exact name): pip, npm, crates.io, Homebrew, poly's built-in tap catalog, plus a partial-name match against the community registry. " +
      "Mirrors poly.candygate.eu/dependencies -- use this to check whether something already exists anywhere before deciding to publish a new community package.",
    inputSchema: {
      type: "object",
      properties: { query: { type: "string", description: "Exact package name (pip/npm/cargo/brew/tap) -- community search is partial-match" } },
      required: ["query"],
      additionalProperties: false,
    },
  },
  {
    name: "get_package",
    description: "Get full detail on one community package: version, description, size, download count, official status, and publisher. Mirrors package.",
    inputSchema: {
      type: "object",
      properties: { name: { type: "string", description: "Exact community package name" } },
      required: ["name"],
      additionalProperties: false,
    },
  },
  {
    name: "list_own_packages",
    description: "List every community package published by this token's account.",
    inputSchema: { type: "object", properties: {}, additionalProperties: false },
  },
  {
    name: "publish_package",
    description:
      "Publish a single-file package to Poly's community registry under this token's account. " +
      "The file becomes installable anywhere via `poly install community:<name>`. " +
      "Package names are first-come-first-served: publishing an existing name you don't own fails. " +
      "Publishing an existing name you do own updates it in place (new version).\n\n" +
      "IMPORTANT: Poly is a CLI package manager, not a web host. A published package is a single " +
      "command-line script or small binary that a user runs from their terminal after " +
      "`poly install community:<name>` -- e.g. a shell script that prints something, generates a " +
      "password, rolls a die, converts a file, etc. Do NOT build an HTML/CSS/JS web app, a styled " +
      "page, or anything meant to be opened in a browser -- that isn't installable or runnable via " +
      "poly and isn't what this registry is for. Keep it simple: a short script is the norm.\n\n" +
      "If this is fulfilling a request submitted on poly.candygate.eu/integrations, set " +
      "category to \"integration\" so it's listed in the site's Integrations library.",
    inputSchema: {
      type: "object",
      properties: {
        name: { type: "string", description: "Package name, must match ^[a-zA-Z0-9_.-]{1,60}$" },
        version: { type: "string", description: "Version string, e.g. 1.0.0" },
        description: { type: "string", description: "Short description shown in search results" },
        file_base64: { type: "string", description: "Base64-encoded file contents (max ~37MB base64, 50MB decoded) -- a script or small binary, not an HTML app" },
        file_name: { type: "string", description: "Original file name, used only to infer an extension" },
        category: { type: "string", description: "Optional. Set to \"integration\" if this fulfills a poly.candygate.eu/integrations request, so it's listed there." },
      },
      required: ["name", "version", "file_base64"],
      additionalProperties: false,
    },
  },
  {
    name: "search_members",
    description: "Search the Poly community directory by (partial) username. Mirrors community.",
    inputSchema: {
      type: "object",
      properties: { query: { type: "string", description: "Username or partial username to search for" } },
      required: ["query"],
      additionalProperties: false,
    },
  },
  {
    name: "get_profile",
    description: "Get a Poly user's public profile by exact username: bio, avatar, plan, official status, badges, member since, and their published packages. Mirrors profile.",
    inputSchema: {
      type: "object",
      properties: { username: { type: "string", description: "Exact username" } },
      required: ["username"],
      additionalProperties: false,
    },
  },
  {
    name: "update_profile",
    description: "Update this token's own account: username and/or bio. Mirrors the profile form on account. Only fields you provide are changed.",
    inputSchema: {
      type: "object",
      properties: {
        username: { type: "string", description: "New username, 1-30 chars, letters/digits/._- only" },
        bio: { type: "string", description: "New bio, max 280 characters" },
      },
      additionalProperties: false,
    },
  },
  {
    name: "update_avatar",
    description: "Update this token's own account avatar image. Mirrors the photo upload on account.",
    inputSchema: {
      type: "object",
      properties: {
        file_base64: { type: "string", description: "Base64-encoded image bytes (max ~2.6MB base64, 2MB decoded)" },
        file_name: { type: "string", description: "Original file name, used to infer the image extension (png/jpg/webp/gif)" },
      },
      required: ["file_base64"],
      additionalProperties: false,
    },
  },
  {
    name: "delete_package",
    description: "Permanently delete a community package this token's account published -- removes both the database entry and the stored file. Cannot be undone, and cannot delete another account's package.",
    inputSchema: {
      type: "object",
      properties: { name: { type: "string", description: "Exact name of the package to delete" } },
      required: ["name"],
      additionalProperties: false,
    },
  },
  {
    name: "get_stats",
    description: "Get Poly's live public stats: total members and pro members. Mirrors the numbers shown on community.",
    inputSchema: { type: "object", properties: {}, additionalProperties: false },
  },

  // --- Social ---
  {
    name: "follow_user",
    description: "Follow a Poly user with this token's account.",
    inputSchema: { type: "object", properties: { username: { type: "string" } }, required: ["username"], additionalProperties: false },
  },
  {
    name: "unfollow_user",
    description: "Unfollow a Poly user.",
    inputSchema: { type: "object", properties: { username: { type: "string" } }, required: ["username"], additionalProperties: false },
  },
  {
    name: "list_followers",
    description: "List a user's followers.",
    inputSchema: { type: "object", properties: { username: { type: "string" } }, required: ["username"], additionalProperties: false },
  },
  {
    name: "list_following",
    description: "List who a user follows.",
    inputSchema: { type: "object", properties: { username: { type: "string" } }, required: ["username"], additionalProperties: false },
  },
  {
    name: "get_follow_status",
    description: "Check this token's relationship with another user: whether you follow them, they follow you, and any friend request status.",
    inputSchema: { type: "object", properties: { username: { type: "string" } }, required: ["username"], additionalProperties: false },
  },
  {
    name: "send_friend_request",
    description: "Send a friend request to a Poly user.",
    inputSchema: { type: "object", properties: { username: { type: "string" } }, required: ["username"], additionalProperties: false },
  },
  {
    name: "accept_friend_request",
    description: "Accept a pending incoming friend request from a user.",
    inputSchema: { type: "object", properties: { username: { type: "string" } }, required: ["username"], additionalProperties: false },
  },
  {
    name: "decline_friend_request",
    description: "Decline (or cancel your own outgoing) a pending friend request with a user.",
    inputSchema: { type: "object", properties: { username: { type: "string" } }, required: ["username"], additionalProperties: false },
  },
  {
    name: "remove_friend",
    description: "Remove an existing friend.",
    inputSchema: { type: "object", properties: { username: { type: "string" } }, required: ["username"], additionalProperties: false },
  },
  {
    name: "list_friend_requests",
    description: "List this token's pending friend requests, both incoming and outgoing.",
    inputSchema: { type: "object", properties: {}, additionalProperties: false },
  },
  {
    name: "list_friends",
    description: "List this token's confirmed friends.",
    inputSchema: { type: "object", properties: {}, additionalProperties: false },
  },

  // --- Notifications ---
  {
    name: "list_notifications",
    description: "List this token's 30 most recent notifications (follows, friend requests, downloads, staff/admin announcements).",
    inputSchema: { type: "object", properties: {}, additionalProperties: false },
  },
  {
    name: "mark_notification_read",
    description: "Mark one notification as read.",
    inputSchema: { type: "object", properties: { notification_id: { type: "string", description: "UUID from list_notifications" } }, required: ["notification_id"], additionalProperties: false },
  },
  {
    name: "mark_all_notifications_read",
    description: "Mark all of this token's notifications as read.",
    inputSchema: { type: "object", properties: {}, additionalProperties: false },
  },
  {
    name: "delete_notification",
    description: "Delete one notification.",
    inputSchema: { type: "object", properties: { notification_id: { type: "string", description: "UUID from list_notifications" } }, required: ["notification_id"], additionalProperties: false },
  },
  {
    name: "get_notification_prefs",
    description: "Get which notification types this account currently receives (service/staff announcements are always delivered regardless).",
    inputSchema: { type: "object", properties: {}, additionalProperties: false },
  },
  {
    name: "update_notification_prefs",
    description: "Change which optional notification types this account receives. Only fields you provide are changed.",
    inputSchema: {
      type: "object",
      properties: {
        follow: { type: "boolean" },
        download: { type: "boolean" },
        friend_request: { type: "boolean" },
        friend_accept: { type: "boolean" },
      },
      additionalProperties: false,
    },
  },

  // --- Profile extras ---
  {
    name: "update_bio_rich",
    description: "Set this token's rich bio (up to 3500 characters of text; light HTML formatting like <b>/<i>/<u>/<a> is allowed).",
    inputSchema: { type: "object", properties: { bio_html: { type: "string" } }, required: ["bio_html"], additionalProperties: false },
  },
  {
    name: "update_bio_photo",
    description: "Set one of this token's 2 bio photo slots.",
    inputSchema: {
      type: "object",
      properties: {
        slot: { type: "integer", description: "1 or 2" },
        file_base64: { type: "string", description: "Base64-encoded image bytes (max ~2.6MB base64, 2MB decoded)" },
        file_name: { type: "string", description: "Original file name, used to infer the image extension" },
      },
      required: ["slot", "file_base64"],
      additionalProperties: false,
    },
  },
  {
    name: "set_dev_mode",
    description: "Enable or disable developer mode on this account (shows advanced account details on account).",
    inputSchema: { type: "object", properties: { enabled: { type: "boolean" } }, required: ["enabled"], additionalProperties: false },
  },
  {
    name: "check_banned",
    description: "Check whether this token's account is currently banned or suspended.",
    inputSchema: { type: "object", properties: {}, additionalProperties: false },
  },

  // --- Admin / staff only (server-side authorization, same rules as
  // the website's hidden /admin page) ---
  {
    name: "admin_ban_user",
    description: "[Admin/staff only] Ban or temporarily suspend a user.",
    inputSchema: {
      type: "object",
      properties: {
        username: { type: "string" },
        reason: { type: "string", description: "Shown to the banned user" },
        days: { type: "integer", description: "Temporary ban length in days; omit for a permanent ban" },
      },
      required: ["username"],
      additionalProperties: false,
    },
  },
  {
    name: "admin_unban_user",
    description: "[Admin/staff only] Lift a user's ban.",
    inputSchema: { type: "object", properties: { username: { type: "string" } }, required: ["username"], additionalProperties: false },
  },
  {
    name: "admin_block_package",
    description: "[Admin/staff only] Block a community package from search/install.",
    inputSchema: { type: "object", properties: { name: { type: "string" }, reason: { type: "string" } }, required: ["name"], additionalProperties: false },
  },
  {
    name: "admin_unblock_package",
    description: "[Admin/staff only] Unblock a community package.",
    inputSchema: { type: "object", properties: { name: { type: "string" } }, required: ["name"], additionalProperties: false },
  },
  {
    name: "admin_search_users",
    description: "[Admin/staff only] Search all users by username or email, including private fields (email, ban status).",
    inputSchema: { type: "object", properties: { query: { type: "string" } }, additionalProperties: false },
  },
  {
    name: "admin_search_packages",
    description: "[Admin/staff only] Search all packages including blocked ones.",
    inputSchema: { type: "object", properties: { query: { type: "string" } }, additionalProperties: false },
  },
  {
    name: "admin_send_notification",
    description: "[Admin/staff only] Send a notification. Staff can only broadcast to everyone; only the admin account can target specific usernames.",
    inputSchema: {
      type: "object",
      properties: {
        title: { type: "string" },
        body_html: { type: "string", description: "Optional rich HTML body" },
        target_usernames: { type: "array", items: { type: "string" }, description: "Omit to broadcast to everyone (admin or staff). Provide to target specific users (admin only)." },
      },
      required: ["title"],
      additionalProperties: false,
    },
  },
  {
    name: "admin_list_banned_users",
    description: "[Admin/staff only] List all currently banned users.",
    inputSchema: { type: "object", properties: {}, additionalProperties: false },
  },
  {
    name: "admin_list_blocked_packages",
    description: "[Admin/staff only] List all currently blocked packages.",
    inputSchema: { type: "object", properties: {}, additionalProperties: false },
  },
];

function jsonRpcResult(id: unknown, result: unknown) {
  return { jsonrpc: "2.0", id, result };
}
function jsonRpcError(id: unknown, code: number, message: string) {
  return { jsonrpc: "2.0", id, error: { code, message } };
}
function toolText(text: string, isError = false) {
  return { content: [{ type: "text", text }], isError };
}

function bearerToken(req: Request): string | null {
  const h = req.headers.get("authorization") || req.headers.get("Authorization");
  if (!h) return null;
  const m = /^Bearer\s+(.+)$/i.exec(h.trim());
  return m ? m[1] : null;
}

async function callRpc(fn: string, body: Record<string, unknown>, key = ANON_KEY) {
  const resp = await fetch(`${SUPABASE_URL}/rest/v1/rpc/${fn}`, {
    method: "POST",
    headers: {
      apikey: key,
      Authorization: `Bearer ${key}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });
  const data = await resp.json().catch(() => null);
  if (!resp.ok) {
    const msg = (data && (data.message || data.error_description)) || resp.statusText;
    throw new Error(String(msg));
  }
  return data;
}

async function handleToolCall(name: string, args: Record<string, unknown>, token: string | null) {
  switch (name) {
    case "whoami": {
      if (!token) return toolText("No token provided. Configure a Poly personal access token (from account) as this connector's Authorization header.", true);
      const rows = await callRpc("mcp_whoami", { p_token: token });
      if (!rows?.length) return toolText("Token did not resolve to an account.", true);
      const p = rows[0];
      const badges = p.badges?.length ? `, badges: ${p.badges.join(", ")}` : "";
      return toolText(`Signed in as @${p.username} (${p.email}), plan: ${p.plan}${p.is_official ? ", official ✓" : ""}${badges}.`);
    }

    case "search_packages": {
      const q = String(args.query ?? "");
      const url = `${SUPABASE_URL}/rest/v1/community_packages_public?name=ilike.*${encodeURIComponent(q)}*&select=name,version,description,download_count,is_official,uploader_username&order=download_count.desc&limit=20`;
      const resp = await fetch(url, { headers: { apikey: ANON_KEY, Authorization: `Bearer ${ANON_KEY}` } });
      const rows = await resp.json();
      if (!resp.ok) return toolText(`Search failed: ${JSON.stringify(rows)}`, true);
      if (!rows.length) return toolText(`No community packages match "${q}".`);
      const lines = rows.map((r: any) =>
        `- ${r.name}@${r.version} by @${r.uploader_username}${r.is_official ? " (official ✓)" : ""} — ${r.download_count} downloads${r.description ? " — " + r.description : ""}`
      );
      return toolText(lines.join("\n"));
    }

    case "search_dependencies": {
      const q = String(args.query ?? "");
      if (!q) return toolText("query is required.", true);

      const BUILTIN_TAPS: Record<string, { version: string; desc: string }> = {
        ripgrep: { version: "15.1.0", desc: "Line-oriented search tool that recursively searches directories for a regex pattern" },
        fd: { version: "10.4.2", desc: "A simple, fast and user-friendly alternative to find" },
        jq: { version: "1.8.2", desc: "Lightweight and flexible command-line JSON processor" },
      };

      async function tryFetch(url: string): Promise<any | null> {
        try {
          const resp = await fetch(url);
          if (resp.status === 404) return null;
          if (!resp.ok) return null;
          return await resp.json();
        } catch {
          return null;
        }
      }

      const [pip, npm, cargo, brew, community] = await Promise.all([
        tryFetch(`https://pypi.org/pypi/${encodeURIComponent(q)}/json`),
        tryFetch(`https://registry.npmjs.org/${encodeURIComponent(q)}/latest`),
        tryFetch(`https://crates.io/api/v1/crates/${encodeURIComponent(q)}`),
        tryFetch(`https://formulae.brew.sh/api/formula/${encodeURIComponent(q)}.json`),
        fetch(
          `${SUPABASE_URL}/rest/v1/community_packages_public?name=ilike.*${encodeURIComponent(q)}*&select=name,version,description,uploader_username&order=download_count.desc&limit=5`,
          { headers: { apikey: ANON_KEY, Authorization: `Bearer ${ANON_KEY}` } }
        ).then((r) => (r.ok ? r.json() : [])).catch(() => []),
      ]);

      const lines: string[] = [];
      const tap = BUILTIN_TAPS[q];
      if (tap) lines.push(`- tap:${q}@${tap.version} — ${tap.desc}`);
      if (pip) lines.push(`- pip:${q}@${pip.info.version} — ${pip.info.summary || ""}`);
      if (npm) lines.push(`- npm:${q}@${npm.version} — ${npm.description || ""}`);
      if (cargo) lines.push(`- cargo:${q}@${cargo.crate.newest_version} — ${cargo.crate.description || ""}`);
      if (brew) lines.push(`- brew:${q}@${brew.versions?.stable || "?"} — ${brew.desc || ""}`);
      for (const c of community) {
        lines.push(`- community:${c.name}@${c.version} by @${c.uploader_username} — ${c.description || ""}`);
      }

      if (!lines.length) return toolText(`No matches for "${q}" in tap, pip, npm, cargo, brew, or community.`);
      return toolText(lines.join("\n"));
    }

    case "get_package": {
      const name = String(args.name ?? "");
      if (!name) return toolText("name is required.", true);
      const url = `${SUPABASE_URL}/rest/v1/community_packages_public?name=eq.${encodeURIComponent(name)}&select=name,version,description,size_bytes,download_count,is_official,updated_at,uploader_username&limit=1`;
      const resp = await fetch(url, { headers: { apikey: ANON_KEY, Authorization: `Bearer ${ANON_KEY}` } });
      const rows = await resp.json();
      if (!resp.ok) return toolText(`Lookup failed: ${JSON.stringify(rows)}`, true);
      if (!rows?.length) return toolText(`No community package named "${name}".`, true);
      const p = rows[0];
      return toolText(
        `${p.name}@${p.version}${p.is_official ? " (official ✓)" : ""}\n` +
        `${p.description || "(no description)"}\n` +
        `Published by @${p.uploader_username} · ${p.download_count} downloads · ${p.size_bytes} bytes · updated ${p.updated_at}\n` +
        `Install: poly install community:${p.name}`
      );
    }

    case "list_own_packages": {
      if (!token) return toolText("No token provided.", true);
      const rows = await callRpc("mcp_list_own_packages", { p_token: token });
      if (!rows?.length) return toolText("This account hasn't published anything yet.");
      const lines = rows.map((r: any) =>
        `- ${r.name}@${r.version}${r.is_official ? " (official ✓)" : ""} — ${r.download_count} downloads`
      );
      return toolText(lines.join("\n"));
    }

    case "publish_package": {
      if (!token) return toolText("No token provided. Configure a Poly personal access token as this connector's Authorization header.", true);
      const name = String(args.name ?? "");
      const version = String(args.version ?? "");
      const description = args.description ? String(args.description) : "";
      const fileBase64 = String(args.file_base64 ?? "");
      const fileName = args.file_name ? String(args.file_name) : name;
      if (!name || !version || !fileBase64) return toolText("name, version, and file_base64 are required.", true);

      const who = await callRpc("mcp_whoami", { p_token: token });
      if (!who?.length) return toolText("Invalid or revoked token.", true);
      const userId = who[0].user_id;

      let bytes: Uint8Array;
      try {
        bytes = Uint8Array.from(atob(fileBase64), (c) => c.charCodeAt(0));
      } catch {
        return toolText("file_base64 is not valid base64.", true);
      }
      if (bytes.length > 50 * 1024 * 1024) return toolText("File exceeds the 50MB limit.", true);

      const digest = await crypto.subtle.digest("SHA-256", bytes);
      const sha256Hex = Array.from(new Uint8Array(digest)).map((b) => b.toString(16).padStart(2, "0")).join("");

      const dotIdx = fileName.lastIndexOf(".");
      const ext = dotIdx > 0 ? fileName.slice(dotIdx) : ".bin";
      const storagePath = `${userId}/${name}/${version}${ext}`;

      const uploadResp = await fetch(`${SUPABASE_URL}/storage/v1/object/community-packages/${storagePath}`, {
        method: "POST",
        headers: {
          apikey: SERVICE_ROLE_KEY,
          Authorization: `Bearer ${SERVICE_ROLE_KEY}`,
          "Content-Type": "application/octet-stream",
          "x-upsert": "true",
        },
        body: bytes,
      });
      if (!uploadResp.ok) {
        const body = await uploadResp.text();
        return toolText(`Upload failed: ${uploadResp.status} ${body}`, true);
      }

      try {
        await callRpc("mcp_publish_package", {
          p_token: token,
          p_name: name,
          p_version: version,
          p_storage_path: storagePath,
          p_sha256: sha256Hex,
          p_size_bytes: bytes.length,
          p_description: description,
          p_category: args.category ? String(args.category) : null,
        });
      } catch (e) {
        return toolText(`Publish failed: ${(e as Error).message}`, true);
      }

      return toolText(`Published ${name}@${version} (${bytes.length} bytes). Anyone can now install it with: poly install community:${name}`);
    }

    case "search_members": {
      if (!token) return toolText("No token provided.", true);
      const q = String(args.query ?? "");
      const rows = await callRpc("mcp_search_members", { p_token: token, p_query: q });
      if (!rows?.length) return toolText(`No members match "${q}".`);
      const lines = rows.map((m: any) => {
        const badges = m.badges?.length ? ` [${m.badges.join(", ")}]` : "";
        return `- @${m.username}${m.is_official ? " (official ✓)" : ""}${badges} — ${m.plan}${m.bio ? " — " + m.bio : ""}`;
      });
      return toolText(lines.join("\n"));
    }

    case "get_profile": {
      if (!token) return toolText("No token provided.", true);
      const username = String(args.username ?? "");
      if (!username) return toolText("username is required.", true);
      const rows = await callRpc("mcp_get_profile", { p_token: token, p_username: username });
      if (!rows?.length) return toolText(`No user named "${username}".`, true);
      const p = rows[0];

      const pkgResp = await fetch(
        `${SUPABASE_URL}/rest/v1/community_packages_public?uploader_username=eq.${encodeURIComponent(username)}&select=name,version,download_count,is_official`,
        { headers: { apikey: ANON_KEY, Authorization: `Bearer ${ANON_KEY}` } }
      );
      const pkgs = pkgResp.ok ? await pkgResp.json() : [];

      const badges = p.badges?.length ? ` — badges: ${p.badges.join(", ")}` : "";
      let text = `@${p.username}${p.is_official ? " (official ✓)" : ""} — ${p.plan} — member since ${p.created_at}${badges}\n${p.bio || "(no bio)"}`;
      if (pkgs.length) {
        text += "\nPublished packages:\n" + pkgs.map((pk: any) => `- ${pk.name}@${pk.version}${pk.is_official ? " (official ✓)" : ""} — ${pk.download_count} downloads`).join("\n");
      } else {
        text += "\nNothing published yet.";
      }
      return toolText(text);
    }

    case "update_profile": {
      if (!token) return toolText("No token provided.", true);
      const username = args.username ? String(args.username) : null;
      const bio = args.bio ? String(args.bio) : null;
      if (!username && !bio) return toolText("Provide at least one of username or bio.", true);
      try {
        const rows = await callRpc("mcp_update_profile", { p_token: token, p_username: username, p_bio: bio });
        const p = rows?.[0];
        return toolText(`Updated. Now @${p?.username}, bio: ${p?.bio || "(empty)"}`);
      } catch (e) {
        return toolText(`Update failed: ${(e as Error).message}`, true);
      }
    }

    case "update_avatar": {
      if (!token) return toolText("No token provided.", true);
      const fileBase64 = String(args.file_base64 ?? "");
      const fileName = args.file_name ? String(args.file_name) : "avatar.png";
      if (!fileBase64) return toolText("file_base64 is required.", true);

      const who = await callRpc("mcp_whoami", { p_token: token });
      if (!who?.length) return toolText("Invalid or revoked token.", true);
      const userId = who[0].user_id;

      let bytes: Uint8Array;
      try {
        bytes = Uint8Array.from(atob(fileBase64), (c) => c.charCodeAt(0));
      } catch {
        return toolText("file_base64 is not valid base64.", true);
      }
      if (bytes.length > 2 * 1024 * 1024) return toolText("Image exceeds the 2MB limit.", true);

      const dotIdx = fileName.lastIndexOf(".");
      const ext = (dotIdx > 0 ? fileName.slice(dotIdx + 1) : "png").toLowerCase();
      const storagePath = `${userId}/avatar.${ext}`;

      // The avatars bucket is restricted to real image MIME types --
      // application/octet-stream (fine for community-packages) gets a
      // 415 here.
      const MIME_BY_EXT: Record<string, string> = { png: "image/png", jpg: "image/jpeg", jpeg: "image/jpeg", webp: "image/webp", gif: "image/gif" };
      const mimeType = MIME_BY_EXT[ext] || "image/png";

      const uploadResp = await fetch(`${SUPABASE_URL}/storage/v1/object/avatars/${storagePath}`, {
        method: "POST",
        headers: {
          apikey: SERVICE_ROLE_KEY,
          Authorization: `Bearer ${SERVICE_ROLE_KEY}`,
          "Content-Type": mimeType,
          "x-upsert": "true",
        },
        body: bytes,
      });
      if (!uploadResp.ok) {
        const body = await uploadResp.text();
        return toolText(`Upload failed: ${uploadResp.status} ${body}`, true);
      }

      const avatarUrl = `${SUPABASE_URL}/storage/v1/object/public/avatars/${storagePath}?t=${Date.now()}`;
      try {
        await callRpc("mcp_update_avatar", { p_token: token, p_avatar_url: avatarUrl });
      } catch (e) {
        return toolText(`Saved image but failed to update profile: ${(e as Error).message}`, true);
      }
      return toolText(`Avatar updated: ${avatarUrl}`);
    }

    case "delete_package": {
      if (!token) return toolText("No token provided.", true);
      const name = String(args.name ?? "");
      if (!name) return toolText("name is required.", true);

      let storagePath: string | undefined;
      try {
        const rows = await callRpc("mcp_delete_package", { p_token: token, p_name: name });
        storagePath = rows?.[0]?.storage_path;
      } catch (e) {
        return toolText(`Delete failed: ${(e as Error).message}`, true);
      }

      if (storagePath) {
        const delResp = await fetch(`${SUPABASE_URL}/storage/v1/object/community-packages/${storagePath}`, {
          method: "DELETE",
          headers: { apikey: SERVICE_ROLE_KEY, Authorization: `Bearer ${SERVICE_ROLE_KEY}` },
        });
        if (!delResp.ok) {
          return toolText(`Deleted ${name} from the registry, but couldn't remove the stored file (${delResp.status}). Not installable anymore either way.`);
        }
      }
      return toolText(`Deleted ${name}. It's no longer installable via poly install community:${name}.`);
    }

    case "get_stats": {
      const rows = await callRpc("community_stats", {});
      const s = rows?.[0];
      if (!s) return toolText("Stats unavailable.", true);
      return toolText(`${s.total_members} members, ${s.pro_members} on Pro.`);
    }

    // --- Social ---
    case "follow_user": {
      if (!token) return toolText("No token provided.", true);
      const username = String(args.username ?? "");
      try {
        await callRpc("mcp_follow_user", { p_token: token, p_username: username });
        return toolText(`Now following @${username}.`);
      } catch (e) {
        return toolText((e as Error).message, true);
      }
    }

    case "unfollow_user": {
      if (!token) return toolText("No token provided.", true);
      const username = String(args.username ?? "");
      try {
        await callRpc("mcp_unfollow_user", { p_token: token, p_username: username });
        return toolText(`Unfollowed @${username}.`);
      } catch (e) {
        return toolText((e as Error).message, true);
      }
    }

    case "list_followers": {
      if (!token) return toolText("No token provided.", true);
      const username = String(args.username ?? "");
      try {
        const rows = await callRpc("mcp_list_followers", { p_token: token, p_username: username });
        if (!rows?.length) return toolText(`@${username} has no followers yet.`);
        return toolText(rows.map((r: any) => `- @${r.username}${r.is_official ? " (official ✓)" : ""}`).join("\n"));
      } catch (e) {
        return toolText((e as Error).message, true);
      }
    }

    case "list_following": {
      if (!token) return toolText("No token provided.", true);
      const username = String(args.username ?? "");
      try {
        const rows = await callRpc("mcp_list_following", { p_token: token, p_username: username });
        if (!rows?.length) return toolText(`@${username} isn't following anyone yet.`);
        return toolText(rows.map((r: any) => `- @${r.username}${r.is_official ? " (official ✓)" : ""}`).join("\n"));
      } catch (e) {
        return toolText((e as Error).message, true);
      }
    }

    case "get_follow_status": {
      if (!token) return toolText("No token provided.", true);
      const username = String(args.username ?? "");
      try {
        const rows = await callRpc("mcp_get_follow_status", { p_token: token, p_username: username });
        const r = rows?.[0];
        if (!r) return toolText("Status unavailable.", true);
        return toolText(`You follow @${username}: ${r.you_follow_them}. @${username} follows you: ${r.they_follow_you}. Friend status: ${r.friend_status}.`);
      } catch (e) {
        return toolText((e as Error).message, true);
      }
    }

    case "send_friend_request": {
      if (!token) return toolText("No token provided.", true);
      const username = String(args.username ?? "");
      try {
        await callRpc("mcp_send_friend_request", { p_token: token, p_username: username });
        return toolText(`Friend request sent to @${username}.`);
      } catch (e) {
        return toolText((e as Error).message, true);
      }
    }

    case "accept_friend_request": {
      if (!token) return toolText("No token provided.", true);
      const username = String(args.username ?? "");
      try {
        await callRpc("mcp_accept_friend_request", { p_token: token, p_username: username });
        return toolText(`You and @${username} are now friends.`);
      } catch (e) {
        return toolText((e as Error).message, true);
      }
    }

    case "decline_friend_request": {
      if (!token) return toolText("No token provided.", true);
      const username = String(args.username ?? "");
      try {
        await callRpc("mcp_decline_friend_request", { p_token: token, p_username: username });
        return toolText(`Friend request with @${username} cleared.`);
      } catch (e) {
        return toolText((e as Error).message, true);
      }
    }

    case "remove_friend": {
      if (!token) return toolText("No token provided.", true);
      const username = String(args.username ?? "");
      try {
        await callRpc("mcp_remove_friend", { p_token: token, p_username: username });
        return toolText(`Removed @${username} as a friend.`);
      } catch (e) {
        return toolText((e as Error).message, true);
      }
    }

    case "list_friend_requests": {
      if (!token) return toolText("No token provided.", true);
      const rows = await callRpc("mcp_list_friend_requests", { p_token: token });
      if (!rows?.length) return toolText("No pending friend requests.");
      return toolText(rows.map((r: any) => `- @${r.username} (${r.direction})`).join("\n"));
    }

    case "list_friends": {
      if (!token) return toolText("No token provided.", true);
      const rows = await callRpc("mcp_list_friends", { p_token: token });
      if (!rows?.length) return toolText("No friends yet.");
      return toolText(rows.map((r: any) => `- @${r.username}${r.is_official ? " (official ✓)" : ""}`).join("\n"));
    }

    // --- Notifications ---
    case "list_notifications": {
      if (!token) return toolText("No token provided.", true);
      const rows = await callRpc("mcp_list_notifications", { p_token: token });
      if (!rows?.length) return toolText("No notifications.");
      return toolText(rows.map((n: any) =>
        `- [${n.read_at ? "read" : "unread"}] ${n.id} · ${n.type} · ${n.title}${n.from_username ? " (from @" + n.from_username + ")" : ""} · ${n.created_at}`
      ).join("\n"));
    }

    case "mark_notification_read": {
      if (!token) return toolText("No token provided.", true);
      await callRpc("mcp_mark_notification_read", { p_token: token, p_id: String(args.notification_id ?? "") });
      return toolText("Marked as read.");
    }

    case "mark_all_notifications_read": {
      if (!token) return toolText("No token provided.", true);
      await callRpc("mcp_mark_all_notifications_read", { p_token: token });
      return toolText("All notifications marked as read.");
    }

    case "delete_notification": {
      if (!token) return toolText("No token provided.", true);
      await callRpc("mcp_delete_notification", { p_token: token, p_id: String(args.notification_id ?? "") });
      return toolText("Notification deleted.");
    }

    case "get_notification_prefs": {
      if (!token) return toolText("No token provided.", true);
      const prefs = await callRpc("mcp_get_notification_prefs", { p_token: token });
      return toolText(JSON.stringify(prefs));
    }

    case "update_notification_prefs": {
      if (!token) return toolText("No token provided.", true);
      try {
        const prefs = await callRpc("mcp_update_notification_prefs", {
          p_token: token,
          p_follow: args.follow ?? null,
          p_download: args.download ?? null,
          p_friend_request: args.friend_request ?? null,
          p_friend_accept: args.friend_accept ?? null,
        });
        return toolText(`Updated. Now: ${JSON.stringify(prefs)}`);
      } catch (e) {
        return toolText((e as Error).message, true);
      }
    }

    // --- Profile extras ---
    case "update_bio_rich": {
      if (!token) return toolText("No token provided.", true);
      const bioHtml = String(args.bio_html ?? "");
      try {
        await callRpc("mcp_update_bio_rich", { p_token: token, p_bio_html: bioHtml });
        return toolText("Bio updated.");
      } catch (e) {
        return toolText((e as Error).message, true);
      }
    }

    case "update_bio_photo": {
      if (!token) return toolText("No token provided. Configure a Poly personal access token as this connector's Authorization header.", true);
      const slot = Number(args.slot ?? 0);
      const fileBase64 = String(args.file_base64 ?? "");
      const fileName = args.file_name ? String(args.file_name) : "photo.jpg";
      if (slot !== 1 && slot !== 2) return toolText("slot must be 1 or 2.", true);
      if (!fileBase64) return toolText("file_base64 is required.", true);

      const who = await callRpc("mcp_whoami", { p_token: token });
      if (!who?.length) return toolText("Invalid or revoked token.", true);
      const userId = who[0].user_id;

      let bytes: Uint8Array;
      try {
        bytes = Uint8Array.from(atob(fileBase64), (c) => c.charCodeAt(0));
      } catch {
        return toolText("file_base64 is not valid base64.", true);
      }
      if (bytes.length > 2 * 1024 * 1024) return toolText("Image exceeds the 2MB limit.", true);

      const dotIdx = fileName.lastIndexOf(".");
      const ext = (dotIdx > 0 ? fileName.slice(dotIdx + 1) : "jpg").toLowerCase();
      const storagePath = `${userId}/bio-${slot}.${ext}`;
      const MIME_BY_EXT: Record<string, string> = { png: "image/png", jpg: "image/jpeg", jpeg: "image/jpeg", webp: "image/webp", gif: "image/gif" };
      const mimeType = MIME_BY_EXT[ext] || "image/jpeg";

      const uploadResp = await fetch(`${SUPABASE_URL}/storage/v1/object/avatars/${storagePath}`, {
        method: "POST",
        headers: {
          apikey: SERVICE_ROLE_KEY,
          Authorization: `Bearer ${SERVICE_ROLE_KEY}`,
          "Content-Type": mimeType,
          "x-upsert": "true",
        },
        body: bytes,
      });
      if (!uploadResp.ok) {
        const body = await uploadResp.text();
        return toolText(`Upload failed: ${uploadResp.status} ${body}`, true);
      }

      const photoUrl = `${SUPABASE_URL}/storage/v1/object/public/avatars/${storagePath}?t=${Date.now()}`;
      try {
        await callRpc("mcp_set_bio_photo", { p_token: token, p_slot: slot, p_url: photoUrl });
      } catch (e) {
        return toolText(`Saved image but failed to update profile: ${(e as Error).message}`, true);
      }
      return toolText(`Bio photo ${slot} updated: ${photoUrl}`);
    }

    case "set_dev_mode": {
      if (!token) return toolText("No token provided.", true);
      const enabled = !!args.enabled;
      await callRpc("mcp_set_dev_mode", { p_token: token, p_enabled: enabled });
      return toolText(`Developer mode ${enabled ? "enabled" : "disabled"}.`);
    }

    case "check_banned": {
      if (!token) return toolText("No token provided.", true);
      const rows = await callRpc("mcp_check_banned", { p_token: token });
      const r = rows?.[0];
      if (!r) return toolText("Status unavailable.", true);
      if (!r.banned) return toolText("Not banned.");
      return toolText(`Banned${r.banned_until ? " until " + r.banned_until : " permanently"}.${r.banned_reason ? " Reason: " + r.banned_reason : ""}`, true);
    }

    // --- Admin / staff only ---
    case "admin_ban_user": {
      if (!token) return toolText("No token provided.", true);
      try {
        await callRpc("mcp_admin_ban_user", { p_token: token, p_username: String(args.username ?? ""), p_reason: args.reason ? String(args.reason) : null, p_days: args.days ?? null });
        return toolText(`@${args.username} banned.`);
      } catch (e) {
        return toolText((e as Error).message, true);
      }
    }

    case "admin_unban_user": {
      if (!token) return toolText("No token provided.", true);
      try {
        await callRpc("mcp_admin_unban_user", { p_token: token, p_username: String(args.username ?? "") });
        return toolText(`@${args.username} unbanned.`);
      } catch (e) {
        return toolText((e as Error).message, true);
      }
    }

    case "admin_block_package": {
      if (!token) return toolText("No token provided.", true);
      try {
        await callRpc("mcp_admin_block_package", { p_token: token, p_name: String(args.name ?? ""), p_reason: args.reason ? String(args.reason) : null });
        return toolText(`${args.name} blocked.`);
      } catch (e) {
        return toolText((e as Error).message, true);
      }
    }

    case "admin_unblock_package": {
      if (!token) return toolText("No token provided.", true);
      try {
        await callRpc("mcp_admin_unblock_package", { p_token: token, p_name: String(args.name ?? "") });
        return toolText(`${args.name} unblocked.`);
      } catch (e) {
        return toolText((e as Error).message, true);
      }
    }

    case "admin_search_users": {
      if (!token) return toolText("No token provided.", true);
      try {
        const rows = await callRpc("mcp_admin_search_users", { p_token: token, p_query: args.query ? String(args.query) : "" });
        if (!rows?.length) return toolText("No matches.");
        return toolText(rows.map((u: any) =>
          `- @${u.username} (${u.email}) · ${u.plan}${u.badge_staff ? " · staff" : ""}${u.banned ? " · BANNED" + (u.banned_reason ? ": " + u.banned_reason : "") : ""}`
        ).join("\n"));
      } catch (e) {
        return toolText((e as Error).message, true);
      }
    }

    case "admin_search_packages": {
      if (!token) return toolText("No token provided.", true);
      try {
        const rows = await callRpc("mcp_admin_search_packages", { p_token: token, p_query: args.query ? String(args.query) : "" });
        if (!rows?.length) return toolText("No matches.");
        return toolText(rows.map((p: any) =>
          `- ${p.name}@${p.version} by @${p.uploader_username} · ${p.download_count} downloads${p.blocked ? " · BLOCKED" + (p.blocked_reason ? ": " + p.blocked_reason : "") : ""}`
        ).join("\n"));
      } catch (e) {
        return toolText((e as Error).message, true);
      }
    }

    case "admin_send_notification": {
      if (!token) return toolText("No token provided.", true);
      const title = String(args.title ?? "");
      if (!title) return toolText("title is required.", true);
      try {
        const n = await callRpc("mcp_admin_send_notification", {
          p_token: token,
          p_title: title,
          p_body_html: args.body_html ? String(args.body_html) : null,
          p_target_usernames: Array.isArray(args.target_usernames) && args.target_usernames.length ? args.target_usernames : null,
        });
        return toolText(`Sent to ${n} recipient(s).`);
      } catch (e) {
        return toolText((e as Error).message, true);
      }
    }

    case "admin_list_banned_users": {
      if (!token) return toolText("No token provided.", true);
      try {
        const rows = await callRpc("mcp_admin_list_banned_users", { p_token: token });
        if (!rows?.length) return toolText("No banned users.");
        return toolText(rows.map((u: any) => `- @${u.username}${u.banned_until ? " until " + u.banned_until : " (permanent)"}${u.banned_reason ? ": " + u.banned_reason : ""}`).join("\n"));
      } catch (e) {
        return toolText((e as Error).message, true);
      }
    }

    case "admin_list_blocked_packages": {
      if (!token) return toolText("No token provided.", true);
      try {
        const rows = await callRpc("mcp_admin_list_blocked_packages", { p_token: token });
        if (!rows?.length) return toolText("No blocked packages.");
        return toolText(rows.map((p: any) => `- ${p.name} by @${p.uploader_username}${p.blocked_reason ? ": " + p.blocked_reason : ""}`).join("\n"));
      } catch (e) {
        return toolText((e as Error).message, true);
      }
    }

    default:
      return toolText(`Unknown tool: ${name}`, true);
  }
}

// Per the MCP Authorization spec, a protected resource server must
// answer an unauthenticated (or invalidly authenticated) request with
// HTTP 401 and a WWW-Authenticate header pointing at its protected-
// resource metadata -- that 401 is the ONLY signal most clients
// (Claude.ai, ChatGPT) use to realize OAuth is required at all. Letting
// `initialize`/`tools/list` quietly succeed without a token (as this
// server used to) means the client sees a working connection and never
// starts the OAuth dance, so the user is never prompted for a token.
const PROTECTED_RESOURCE_METADATA_URL = "https://iuymslcbbrbahxbfuzrr.supabase.co/functions/v1/mcp/.well-known/oauth-protected-resource";

function unauthorizedResponse(id: unknown): Response {
  return new Response(JSON.stringify(jsonRpcError(id ?? null, -32001, "Authentication required")), {
    status: 401,
    headers: {
      ...CORS_HEADERS,
      "Content-Type": "application/json",
      "WWW-Authenticate": `Bearer resource_metadata="${PROTECTED_RESOURCE_METADATA_URL}"`,
    },
  });
}

async function handleJsonRpc(req: Request): Promise<Response> {
  let body: any;
  try {
    body = await req.json();
  } catch {
    return new Response(JSON.stringify(jsonRpcError(null, -32700, "Parse error")), {
      status: 400,
      headers: { ...CORS_HEADERS, "Content-Type": "application/json" },
    });
  }

  const { id, method, params } = body ?? {};
  const isNotification = id === undefined;
  const token = bearerToken(req);

  // notifications/initialized carries no meaningful auth outcome (the
  // client doesn't process a response body for it) -- let it through
  // regardless so a client that's already past the challenge doesn't
  // get stuck on a stray notification.
  if (method !== "notifications/initialized") {
    let authed = false;
    if (token) {
      try {
        const who = await callRpc("mcp_whoami", { p_token: token });
        authed = !!who?.length;
      } catch {
        authed = false;
      }
    }
    if (!authed) return unauthorizedResponse(id);
  }

  try {
    let result: unknown;

    switch (method) {
      case "initialize":
        result = {
          protocolVersion: "2025-03-26",
          capabilities: { tools: {} },
          serverInfo: { name: "poly-mcp", version: "1.1.0" },
        };
        break;

      case "notifications/initialized":
        return new Response(null, { status: 202, headers: CORS_HEADERS });

      case "ping":
        result = {};
        break;

      case "tools/list":
        result = { tools: TOOLS };
        break;

      case "tools/call": {
        const toolName = params?.name;
        const args = params?.arguments ?? {};
        result = await handleToolCall(toolName, args, token);
        break;
      }

      default:
        if (isNotification) return new Response(null, { status: 202, headers: CORS_HEADERS });
        return new Response(JSON.stringify(jsonRpcError(id, -32601, `Method not found: ${method}`)), {
          status: 404,
          headers: { ...CORS_HEADERS, "Content-Type": "application/json" },
        });
    }

    if (isNotification) return new Response(null, { status: 202, headers: CORS_HEADERS });
    return new Response(JSON.stringify(jsonRpcResult(id, result)), {
      headers: { ...CORS_HEADERS, "Content-Type": "application/json" },
    });
  } catch (e) {
    if (isNotification) return new Response(null, { status: 202, headers: CORS_HEADERS });
    return new Response(JSON.stringify(jsonRpcError(id, -32603, (e as Error).message)), {
      status: 500,
      headers: { ...CORS_HEADERS, "Content-Type": "application/json" },
    });
  }
}

// ---------------------------------------------------------------------
// OAuth 2.1 + Dynamic Client Registration shim
// ---------------------------------------------------------------------

function randomId(bytes = 24): string {
  const arr = new Uint8Array(bytes);
  crypto.getRandomValues(arr);
  return Array.from(arr, (b) => b.toString(16).padStart(2, "0")).join("");
}

function base64url(bytes: ArrayBuffer): string {
  const b64 = btoa(String.fromCharCode(...new Uint8Array(bytes)));
  return b64.replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

// Every request self-registers as a client; there is no per-client
// state worth tracking (public client, PKCE handles verification), so
// this just mints an opaque id rather than persisting anything.
async function handleRegister(req: Request): Promise<Response> {
  // RFC 7591 clients commonly expect their submitted metadata (redirect_uris
  // above all) echoed back in the response, since that's what tells them
  // which values the server considers registered for this client_id.
  const submitted = await req.json().catch(() => ({} as Record<string, unknown>));
  const body = {
    client_id: "poly-mcp-" + randomId(8),
    client_id_issued_at: Math.floor(Date.now() / 1000),
    token_endpoint_auth_method: "none",
    grant_types: ["authorization_code"],
    response_types: ["code"],
    redirect_uris: submitted.redirect_uris ?? [],
    client_name: submitted.client_name ?? "Poly",
  };
  return new Response(JSON.stringify(body), { status: 201, headers: { ...CORS_HEADERS, "Content-Type": "application/json" } });
}

// The authorization_endpoint is deliberately NOT this edge function --
// Supabase's default *.supabase.co domain forcibly downgrades any
// text/html response from an Edge Function to text/plain with a
// locked-down CSP (documented, intentional, to stop the shared domain
// being used for phishing pages). A custom domain lifts that, but costs
// extra; simplest fix is hosting the one interactive page (the "paste
// your token" form) as a static page on poly.candygate.eu instead,
// which has no such restriction. This function only ever returns JSON
// or redirects, never HTML.
const CONNECT_PAGE_URL = "https://poly.candygate.eu/mcp-connect";

function oauthMetadata(baseUrl: string) {
  return {
    issuer: baseUrl,
    authorization_endpoint: CONNECT_PAGE_URL,
    token_endpoint: `${baseUrl}/token`,
    registration_endpoint: `${baseUrl}/register`,
    response_types_supported: ["code"],
    grant_types_supported: ["authorization_code"],
    code_challenge_methods_supported: ["S256", "plain"],
    token_endpoint_auth_methods_supported: ["none"],
  };
}

function handleAuthMetadata(baseUrl: string): Response {
  return new Response(JSON.stringify(oauthMetadata(baseUrl)), { headers: { ...CORS_HEADERS, "Content-Type": "application/json" } });
}

function handleResourceMetadata(baseUrl: string): Response {
  const body = { resource: baseUrl, authorization_servers: [baseUrl] };
  return new Response(JSON.stringify(body), { headers: { ...CORS_HEADERS, "Content-Type": "application/json" } });
}

// GET /authorize is only ever hit by a client that ignored the
// authorization_endpoint in the metadata and guessed this URL --
// bounce it to the real (static, HTML-capable) connect page with the
// same query string rather than trying to serve HTML here.
async function handleAuthorizeGet(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const target = `${CONNECT_PAGE_URL}?${url.searchParams.toString()}`;
  return new Response(null, { status: 302, headers: { ...CORS_HEADERS, Location: target } });
}

async function handleAuthorizePost(req: Request): Promise<Response> {
  const form = await req.formData();
  const polyToken = String(form.get("poly_token") || "").trim();
  const redirectUri = String(form.get("redirect_uri") || "");
  const state = String(form.get("state") || "");
  const codeChallenge = String(form.get("code_challenge") || "");
  const clientId = String(form.get("client_id") || "");
  const codeChallengeMethod = String(form.get("code_challenge_method") || "");
  const responseType = String(form.get("response_type") || "");

  if (!redirectUri) {
    return new Response("missing redirect_uri", { status: 400, headers: CORS_HEADERS });
  }

  // Validate the token actually resolves before minting a code for it.
  // On failure, bounce back to the static connect page (not re-render
  // HTML here) so the user sees the error and can retry.
  try {
    const who = await callRpc("mcp_whoami", { p_token: polyToken });
    if (!who?.length) throw new Error("empty");
  } catch {
    const back = new URL(CONNECT_PAGE_URL);
    back.searchParams.set("client_id", clientId);
    back.searchParams.set("redirect_uri", redirectUri);
    back.searchParams.set("state", state);
    back.searchParams.set("code_challenge", codeChallenge);
    back.searchParams.set("code_challenge_method", codeChallengeMethod);
    back.searchParams.set("response_type", responseType);
    back.searchParams.set("error", "invalid_token");
    back.searchParams.set("error_description", "That token isn't valid or has been revoked. Generate a new one and try again.");
    return new Response(null, { status: 302, headers: { ...CORS_HEADERS, Location: back.toString() } });
  }

  const code = randomId(24);
  await fetch(`${SUPABASE_URL}/rest/v1/mcp_oauth_codes`, {
    method: "POST",
    headers: {
      apikey: SERVICE_ROLE_KEY,
      Authorization: `Bearer ${SERVICE_ROLE_KEY}`,
      "Content-Type": "application/json",
      Prefer: "return=minimal",
    },
    body: JSON.stringify({ code, poly_token: polyToken, redirect_uri: redirectUri, code_challenge: codeChallenge || null }),
  });

  const redirect = new URL(redirectUri);
  redirect.searchParams.set("code", code);
  if (state) redirect.searchParams.set("state", state);
  return new Response(null, { status: 302, headers: { ...CORS_HEADERS, Location: redirect.toString() } });
}

async function handleToken(req: Request): Promise<Response> {
  const contentType = req.headers.get("content-type") || "";
  let params: URLSearchParams;
  if (contentType.includes("application/json")) {
    const body = await req.json().catch(() => ({}));
    params = new URLSearchParams(body as Record<string, string>);
  } else {
    params = new URLSearchParams(await req.text());
  }

  const grantType = params.get("grant_type");
  if (grantType !== "authorization_code") {
    return new Response(JSON.stringify({ error: "unsupported_grant_type" }), { status: 400, headers: { ...CORS_HEADERS, "Content-Type": "application/json" } });
  }

  const code = params.get("code") || "";
  const codeVerifier = params.get("code_verifier") || "";

  const lookupResp = await fetch(`${SUPABASE_URL}/rest/v1/mcp_oauth_codes?code=eq.${encodeURIComponent(code)}&select=poly_token,code_challenge,created_at`, {
    headers: { apikey: SERVICE_ROLE_KEY, Authorization: `Bearer ${SERVICE_ROLE_KEY}` },
  });
  const rows = await lookupResp.json();
  if (!rows?.length) {
    return new Response(JSON.stringify({ error: "invalid_grant" }), { status: 400, headers: { ...CORS_HEADERS, "Content-Type": "application/json" } });
  }
  const row = rows[0];

  // Single-use: delete immediately, regardless of outcome below.
  await fetch(`${SUPABASE_URL}/rest/v1/mcp_oauth_codes?code=eq.${encodeURIComponent(code)}`, {
    method: "DELETE",
    headers: { apikey: SERVICE_ROLE_KEY, Authorization: `Bearer ${SERVICE_ROLE_KEY}` },
  });

  const ageMs = Date.now() - new Date(row.created_at).getTime();
  if (ageMs > 10 * 60 * 1000) {
    return new Response(JSON.stringify({ error: "invalid_grant", error_description: "code expired" }), { status: 400, headers: { ...CORS_HEADERS, "Content-Type": "application/json" } });
  }

  if (row.code_challenge) {
    if (!codeVerifier) {
      return new Response(JSON.stringify({ error: "invalid_grant", error_description: "code_verifier required" }), { status: 400, headers: { ...CORS_HEADERS, "Content-Type": "application/json" } });
    }
    const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(codeVerifier));
    const computed = base64url(digest);
    if (computed !== row.code_challenge) {
      return new Response(JSON.stringify({ error: "invalid_grant", error_description: "PKCE verification failed" }), { status: 400, headers: { ...CORS_HEADERS, "Content-Type": "application/json" } });
    }
  }

  const body = {
    access_token: row.poly_token,
    token_type: "Bearer",
    expires_in: 315360000, // Poly tokens don't expire on their own; only explicit revocation ends them.
  };
  return new Response(JSON.stringify(body), { headers: { ...CORS_HEADERS, "Content-Type": "application/json" } });
}

// ---------------------------------------------------------------------
// Router
// ---------------------------------------------------------------------

Deno.serve(async (req: Request) => {
  if (req.method === "OPTIONS") {
    return new Response(null, { headers: CORS_HEADERS });
  }

  const url = new URL(req.url);
  const path = url.pathname;
  // Not derived from req.url/url.origin: Supabase's edge runtime hands
  // this function an internal http:// URL even though the public
  // request was always https:// -- OAuth clients would then try to
  // hit http endpoints and fail (or worse). This is always reachable
  // at this exact public https URL, so it's simplest to hardcode it.
  const baseUrl = "https://iuymslcbbrbahxbfuzrr.supabase.co/functions/v1/mcp";

  if (path.endsWith("/.well-known/oauth-authorization-server")) return handleAuthMetadata(baseUrl);
  // Some MCP clients (observed: Claude.ai) probe OpenID Connect
  // Discovery instead of (or before) plain OAuth AS metadata. The
  // response shape is a superset-compatible document either way, so
  // serve the same metadata here rather than leaving it 404/405 and
  // stalling the whole discovery chain before it ever reaches
  // /register or /authorize.
  if (path.endsWith("/.well-known/openid-configuration")) return handleAuthMetadata(baseUrl);
  if (path.endsWith("/.well-known/oauth-protected-resource")) return handleResourceMetadata(baseUrl);
  if (path.endsWith("/register") && req.method === "POST") return handleRegister(req);
  if (path.endsWith("/authorize") && req.method === "GET") return handleAuthorizeGet(req);
  if (path.endsWith("/authorize") && req.method === "POST") return handleAuthorizePost(req);
  if (path.endsWith("/token") && req.method === "POST") return handleToken(req);

  if (req.method !== "POST") {
    return new Response("method not allowed", { status: 405, headers: CORS_HEADERS });
  }
  return handleJsonRpc(req);
});
