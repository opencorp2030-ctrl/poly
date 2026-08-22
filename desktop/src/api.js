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
    .select("email,username,bio,plan,created_at")
    .eq("id", uid)
    .single();
  if (error) throw error;
  return data;
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
  listOwnApps,
  getApp,
  upsertApp,
  publishRelease,
  deleteApp,
  uploadAppMedia,
  uploadAppBuild,
};
