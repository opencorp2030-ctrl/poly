// Poly's MCP server, exposed as a Supabase Edge Function so AI assistants
// (Claude, ChatGPT, or anything else that speaks MCP) can publish and
// browse community packages "as" a signed-in Poly user, without ever
// handling that user's email/password.
//
// Auth model: the connecting client sends a Poly personal access token
// (generated on account.html, shown once) as `Authorization: Bearer
// <token>`. That token is resolved to a user_id inside Postgres, by the
// mcp_* SECURITY DEFINER functions (see migration
// add_api_tokens_and_mcp_functions) -- this function never sees a
// Supabase session JWT, so verify_jwt is disabled for this endpoint and
// custom auth happens entirely via those RPCs.
//
// Transport: MCP "Streamable HTTP", stateless mode -- every request is a
// self-contained JSON-RPC 2.0 call over a single POST, no server-side
// session state, which fits a serverless function cleanly.

const SUPABASE_URL = Deno.env.get("SUPABASE_URL")!;
const ANON_KEY = Deno.env.get("SUPABASE_ANON_KEY")!;
const SERVICE_ROLE_KEY = Deno.env.get("SUPABASE_SERVICE_ROLE_KEY")!;

const CORS_HEADERS = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Headers": "authorization, content-type, mcp-session-id, mcp-protocol-version",
  "Access-Control-Allow-Methods": "POST, OPTIONS",
};

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

Deno.serve(async (req: Request) => {
  if (req.method === "OPTIONS") {
    return new Response(null, { headers: CORS_HEADERS });
  }
  if (req.method !== "POST") {
    return new Response("method not allowed", { status: 405, headers: CORS_HEADERS });
  }

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
          serverInfo: { name: "poly-mcp", version: "1.0.0" },
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
});
