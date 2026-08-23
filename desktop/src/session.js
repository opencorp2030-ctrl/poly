// Auth session management, mirroring internal/account/account.go: same
// Supabase project, same public/RLS-scoped anon key (safe to embed --
// already shipped inside the Go CLI binary), same password grant, same
// ~/.poly/credentials.json file so the CLI and Desktop share one session.
const { createClient } = require("@supabase/supabase-js");
const WebSocket = require("ws");
const creds = require("./credentials");

const SUPABASE_URL = "https://iuymslcbbrbahxbfuzrr.supabase.co";
const SUPABASE_ANON_KEY = "sb_publishable_ocTnDk4vyMMGcyGBeB3bOg_0CMGkdaD";

// Electron's bundled Node doesn't expose a native global WebSocket
// (that lands in Node 22+); supabase-js's realtime client requires one
// to even construct, though this app never opens a realtime channel --
// hand it the `ws` package instead of shipping a newer Node.
const supabase = createClient(SUPABASE_URL, SUPABASE_ANON_KEY, {
  auth: { persistSession: false, autoRefreshToken: false },
  realtime: { transport: WebSocket },
});

let currentUser = null; // { id, email }

function currentUserId() {
  return currentUser ? currentUser.id : null;
}

async function applySession(accessToken, refreshToken) {
  const { data, error } = await supabase.auth.setSession({
    access_token: accessToken,
    refresh_token: refreshToken,
  });
  if (error) throw error;
  currentUser = { id: data.user.id, email: data.user.email };
  return currentUser;
}

// Applies a session handed back by the desktop-connect popup window
// (already authenticated against poly.candygate.eu) and persists it the
// same way the CLI's `poly login` does.
async function loginWithTokens(accessToken, refreshToken) {
  const { data, error } = await supabase.auth.setSession({
    access_token: accessToken,
    refresh_token: refreshToken,
  });
  if (error) throw error;
  currentUser = { id: data.user.id, email: data.user.email };
  creds.save({
    user_id: data.user.id,
    email: data.user.email,
    access_token: data.session.access_token,
    refresh_token: data.session.refresh_token,
  });
  return currentUser;
}

// Mirrors account.LoadFresh(): try the stored session, transparently
// refresh if the access token is expired or about to expire, fail open
// (return null) if nothing usable is on disk.
async function resume() {
  const stored = creds.load();
  if (!stored || !stored.access_token) return null;

  const expiry = creds.jwtExpiry(stored.access_token);
  const freshEnough = expiry && expiry.getTime() - Date.now() > 30_000;

  if (freshEnough) {
    try {
      return await applySession(stored.access_token, stored.refresh_token);
    } catch {
      // fall through to a refresh attempt
    }
  }

  try {
    const { data, error } = await supabase.auth.refreshSession({ refresh_token: stored.refresh_token });
    if (error) throw error;
    currentUser = { id: data.user.id, email: data.user.email };
    creds.save({
      user_id: data.user.id,
      email: data.user.email,
      access_token: data.session.access_token,
      refresh_token: data.session.refresh_token,
    });
    return currentUser;
  } catch {
    return null;
  }
}

function logout() {
  currentUser = null;
  creds.clear();
  supabase.auth.signOut().catch(() => {});
}

module.exports = { supabase, loginWithTokens, resume, logout, currentUserId, get currentUser() { return currentUser; } };
