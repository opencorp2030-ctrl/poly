// App-management calls, mirroring exactly what site/publish-app.html does
// against the same RPCs/tables/storage buckets (upsert_app,
// publish_app_release, delete_app, apps, app-media, app-builds).
const session = require("./session");

function requireUser() {
  const uid = session.currentUserId();
  if (!uid) throw new Error("not signed in");
  return uid;
}

async function getProfile() {
  const uid = requireUser();
  const { data, error } = await session.supabase
    .from("profiles")
    .select("email,username,bio,avatar_url,plan,is_official,notification_prefs,created_at")
    .eq("id", uid)
    .single();
  if (error) throw error;
  return data;
}

async function updateProfile({ username, bio }) {
  const uid = requireUser();
  const patch = {};
  if (username !== undefined) patch.username = username;
  if (bio !== undefined) patch.bio = bio;
  const { error } = await session.supabase.from("profiles").update(patch).eq("id", uid);
  if (error) throw error;
}

async function updateNotificationPrefs(prefs) {
  const uid = requireUser();
  const { error } = await session.supabase.from("profiles").update({ notification_prefs: prefs }).eq("id", uid);
  if (error) throw error;
}

async function uploadAvatar(buffer, fileName) {
  const uid = requireUser();
  const dotIdx = fileName.lastIndexOf(".");
  const ext = (dotIdx > 0 ? fileName.slice(dotIdx + 1) : "png").toLowerCase();
  const path = `${uid}/avatar.${ext}`;
  const { error: uploadError } = await session.supabase.storage.from("avatars").upload(path, buffer, {
    upsert: true,
    contentType: `image/${ext === "jpg" ? "jpeg" : ext}`,
  });
  if (uploadError) throw uploadError;
  const { data: urlData } = session.supabase.storage.from("avatars").getPublicUrl(path);
  const avatarUrl = `${urlData.publicUrl}?t=${Date.now()}`;
  const { error: updateError } = await session.supabase.from("profiles").update({ avatar_url: avatarUrl }).eq("id", uid);
  if (updateError) throw updateError;
  return avatarUrl;
}

// --- Community (search, follow) ---

async function searchMembers(query, page = 1, pageSize = 20) {
  const uid = requireUser();
  const from = (page - 1) * pageSize;
  const to = from + pageSize - 1;
  let q = session.supabase.from("profiles_public").select("id,username,bio,avatar_url,plan,is_official", { count: "exact" });
  if (query) q = q.ilike("username", `%${query}%`).neq("id", uid);
  const { data, error, count } = await q
    .order("plan", { ascending: false })
    .order("username", { ascending: true })
    .range(from, to);
  if (error) throw error;
  return { rows: data, count: count || 0 };
}

async function getFollowStatus(targetId) {
  const uid = requireUser();
  const { data, error } = await session.supabase
    .from("follows")
    .select("follower_id")
    .eq("follower_id", uid)
    .eq("followed_id", targetId)
    .maybeSingle();
  if (error) throw error;
  return { following: !!data };
}

async function setFollow(targetId, follow) {
  const uid = requireUser();
  if (follow) {
    const { error } = await session.supabase.from("follows").insert({ follower_id: uid, followed_id: targetId });
    if (error) throw error;
  } else {
    const { error } = await session.supabase.from("follows").delete().eq("follower_id", uid).eq("followed_id", targetId);
    if (error) throw error;
  }
}

// --- Apps store (public browse) ---

async function searchApps(query, sort = "new", page = 1, pageSize = 20) {
  const from = (page - 1) * pageSize;
  const to = from + pageSize - 1;
  let q = session.supabase.from("apps_public").select("*", { count: "exact" });
  if (query) q = q.or(`name.ilike.%${query}%,tagline.ilike.%${query}%`);
  if (sort === "rating") q = q.order("avg_rating", { ascending: false }).order("rating_count", { ascending: false });
  else if (sort === "downloads") q = q.order("download_count", { ascending: false });
  else q = q.order("created_at", { ascending: false });
  const { data, error, count } = await q.range(from, to);
  if (error) throw error;
  return { rows: data, count: count || 0 };
}

async function listOwnApps() {
  const uid = requireUser();
  const { data, error } = await session.supabase
    .from("apps")
    .select("*")
    .eq("publisher_id", uid)
    .order("created_at", { ascending: false });
  if (error) throw error;
  return data;
}

async function getApp(id) {
  const uid = requireUser();
  const { data, error } = await session.supabase
    .from("apps")
    .select("*")
    .eq("id", id)
    .eq("publisher_id", uid)
    .maybeSingle();
  if (error) throw error;
  return data;
}

async function upsertApp(fields) {
  requireUser();
  const { data, error } = await session.supabase.rpc("upsert_app", fields);
  if (error) throw error;
  return data;
}

async function publishRelease(fields) {
  requireUser();
  const { data, error } = await session.supabase.rpc("publish_app_release", fields);
  if (error) throw error;
  return data;
}

async function deleteApp(appId) {
  requireUser();
  const { data, error } = await session.supabase.rpc("delete_app", { p_app_id: appId });
  if (error) throw error;
  return data;
}

function detectMime(buffer) {
  if (buffer[0] === 0x89 && buffer[1] === 0x50) return "image/png";
  if (buffer[0] === 0xff && buffer[1] === 0xd8) return "image/jpeg";
  if (buffer[0] === 0x47 && buffer[1] === 0x49) return "image/gif";
  if (buffer[0] === 0x52 && buffer[1] === 0x49) return "image/webp";
  return "application/octet-stream";
}

async function uploadAppMedia(appId, buffer, label) {
  const uid = requireUser();
  const mime = detectMime(buffer);
  const ext = mime.split("/")[1] || "png";
  const path = `${uid}/${appId}/${label}-${Date.now()}.${ext}`;
  const { error } = await session.supabase.storage.from("app-media").upload(path, buffer, {
    upsert: true,
    contentType: mime,
  });
  if (error) throw error;
  const { data } = session.supabase.storage.from("app-media").getPublicUrl(path);
  return data.publicUrl;
}

async function uploadAppBuild(appId, buffer, fileName) {
  const uid = requireUser();
  const dotIdx = fileName.lastIndexOf(".");
  const ext = dotIdx > 0 ? fileName.slice(dotIdx) : ".bin";
  const path = `${uid}/${appId}/build-${Date.now()}${ext}`;
  const { error } = await session.supabase.storage.from("app-builds").upload(path, buffer, {
    upsert: true,
    contentType: "application/octet-stream",
  });
  if (error) throw error;
  return { storage_path: path, size_bytes: buffer.length };
}

module.exports = {
  getProfile,
  updateProfile,
  updateNotificationPrefs,
  uploadAvatar,
  listOwnApps,
  getApp,
  upsertApp,
  publishRelease,
  deleteApp,
  uploadAppMedia,
  uploadAppBuild,
  searchMembers,
  getFollowStatus,
  setFollow,
  searchApps,
};
