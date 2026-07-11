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
//     revoking the token on account.html still kills it everywhere.
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
    description: "Return the signed-in Poly account this token belongs to: username, email, plan, and whether it's the official account.",
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
      "Publishing an existing name you do own updates it in place (new version).",
    inputSchema: {
      type: "object",
      properties: {
        name: { type: "string", description: "Package name, must match ^[a-zA-Z0-9_.-]{1,60}$" },
        version: { type: "string", description: "Version string, e.g. 1.0.0" },
        description: { type: "string", description: "Short description shown in search results" },
        file_base64: { type: "string", description: "Base64-encoded file contents (max ~37MB base64, 50MB decoded)" },
        file_name: { type: "string", description: "Original file name, used only to infer an extension" },
      },
      required: ["name", "version", "file_base64"],
      additionalProperties: false,
    },
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
      if (!token) return toolText("No token provided. Configure a Poly personal access token (from account.html) as this connector's Authorization header.", true);
      const rows = await callRpc("mcp_whoami", { p_token: token });
      if (!rows?.length) return toolText("Token did not resolve to an account.", true);
      const p = rows[0];
      return toolText(`Signed in as @${p.username} (${p.email}), plan: ${p.plan}${p.is_official ? ", official ✓" : ""}.`);
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
        });
      } catch (e) {
        return toolText(`Publish failed: ${(e as Error).message}`, true);
      }

      return toolText(`Published ${name}@${version} (${bytes.length} bytes). Anyone can now install it with: poly install community:${name}`);
    }

    default:
      return toolText(`Unknown tool: ${name}`, true);
  }
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
function handleRegister(): Response {
  const body = {
    client_id: "poly-mcp-" + randomId(8),
    client_id_issued_at: Math.floor(Date.now() / 1000),
    token_endpoint_auth_method: "none",
    grant_types: ["authorization_code"],
    response_types: ["code"],
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
const CONNECT_PAGE_URL = "https://poly.candygate.eu/mcp-connect.html";

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
  if (path.endsWith("/.well-known/oauth-protected-resource")) return handleResourceMetadata(baseUrl);
  if (path.endsWith("/register") && req.method === "POST") return handleRegister();
  if (path.endsWith("/authorize") && req.method === "GET") return handleAuthorizeGet(req);
  if (path.endsWith("/authorize") && req.method === "POST") return handleAuthorizePost(req);
  if (path.endsWith("/token") && req.method === "POST") return handleToken(req);

  if (req.method !== "POST") {
    return new Response("method not allowed", { status: 405, headers: CORS_HEADERS });
  }
  return handleJsonRpc(req);
});
