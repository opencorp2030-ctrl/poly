// Shared header widget: mobile nav toggle, active-link highlighting,
// user menu, notification bell (+ polling, staff broadcast composer),
// ban gate, and theme toggle wiring. Runs on every page that includes
// the standard header markup. Page-specific business logic (account
// settings, admin actions, profile rendering, ...) stays in each page
// and is untouched by this module.
import { supabase } from './supabase-client.js';
import { currentEffectiveTheme, initThemeToggle } from './theme.js';
import { refreshIcons } from './motion.js';

const ADMIN_ID = '7aeb0cc5-7188-46f3-8e12-4caa84573087';

function escapeHtml(s) {
  const div = document.createElement('div');
  div.textContent = s == null ? '' : String(s);
  return div.innerHTML;
}

function timeAgo(iso) {
  const diff = (Date.now() - new Date(iso).getTime()) / 1000;
  if (diff < 60) return 'now';
  if (diff < 3600) return Math.floor(diff / 60) + 'm';
  if (diff < 86400) return Math.floor(diff / 3600) + 'h';
  return Math.floor(diff / 86400) + 'd';
}

function initMobileToggle() {
  const toggle = document.getElementById('nav-toggle');
  const links = document.getElementById('top-links');
  if (!toggle || !links) return;
  toggle.addEventListener('click', () => {
    const open = links.classList.toggle('open');
    toggle.setAttribute('aria-expanded', open ? 'true' : 'false');
  });
  links.querySelectorAll('a').forEach((a) => {
    a.addEventListener('click', () => {
      links.classList.remove('open');
      toggle.setAttribute('aria-expanded', 'false');
    });
  });
}

function initActiveLink() {
  const links = document.getElementById('top-links');
  if (!links) return;
  const path = window.location.pathname.replace(/^\/+|\/+$/g, '') || 'index';
  links.querySelectorAll('a[href]').forEach((a) => {
    const href = a.getAttribute('href').replace(/^\/+|\/+$/g, '');
    a.classList.toggle('current', href === path || (href === '' && path === 'index'));
  });
}

async function renderUserMenu(session) {
  const menu = document.getElementById('user-menu');
  if (!menu) return;
  if (!session) { menu.style.display = 'none'; return; }
  const { data } = await supabase.from('profiles_public').select('username, avatar_url').eq('id', session.user.id).single();
  const avatarSlot = document.getElementById('user-menu-avatar');
  if (avatarSlot) {
    if (data && data.avatar_url) {
      const img = document.createElement('img');
      img.id = 'user-menu-avatar';
      img.src = data.avatar_url;
      img.alt = '';
      avatarSlot.replaceWith(img);
    } else {
      avatarSlot.textContent = ((data && data.username) || session.user.email || '?').charAt(0).toUpperCase();
    }
  }
  const nameEl = document.getElementById('user-menu-name');
  if (nameEl) nameEl.textContent = data && data.username ? '@' + data.username : session.user.email;
  menu.style.display = 'block';
}

function initUserMenu() {
  const btn = document.getElementById('user-menu-btn');
  const dropdown = document.getElementById('user-menu-dropdown');
  const menu = document.getElementById('user-menu');
  const signout = document.getElementById('user-menu-signout');
  if (btn && dropdown) {
    btn.addEventListener('click', () => {
      const open = dropdown.classList.toggle('open');
      btn.setAttribute('aria-expanded', open ? 'true' : 'false');
    });
  }
  if (menu) {
    document.addEventListener('click', (e) => {
      if (!menu.contains(e.target)) {
        dropdown && dropdown.classList.remove('open');
        btn && btn.setAttribute('aria-expanded', 'false');
      }
    });
  }
  if (signout) {
    signout.addEventListener('click', async () => {
      await supabase.auth.signOut();
      window.location.reload();
    });
  }
}

async function checkBan(session) {
  if (!session) return;
  const { data } = await supabase.rpc('check_banned');
  if (data && data[0] && data[0].banned) window.location.href = 'banned';
}

let notifCache = [];
let notifPollInterval = null;

async function loadNotifications() {
  const list = document.getElementById('notif-list');
  const countEl = document.getElementById('notif-count');
  if (!list || !countEl) return;
  const { data } = await supabase.from('notifications').select('id, type, title, body_html, link_url, is_service, created_at, read_at').order('created_at', { ascending: false }).limit(30);
  notifCache = data || [];
  if (!notifCache.length) {
    list.innerHTML = '<div class="notif-empty">No notifications yet.</div>';
    countEl.style.display = 'none';
    return;
  }
  const unread = notifCache.filter((n) => !n.read_at).length;
  if (unread > 0) { countEl.textContent = unread > 9 ? '9+' : String(unread); countEl.style.display = 'flex'; } else { countEl.style.display = 'none'; }
  list.innerHTML = notifCache.map((n) => `
    <a class="notif-item ${n.read_at ? '' : 'unread'}" data-id="${n.id}">
      <div class="n-title">${escapeHtml(n.title)}</div>
      ${n.body_html ? `<div class="n-body">${n.body_html}</div>` : ''}
      <div class="n-time">${timeAgo(n.created_at)}</div>
    </a>
  `).join('');
  list.querySelectorAll('.notif-item').forEach((el) => {
    el.addEventListener('click', async (e) => {
      e.preventDefault();
      const id = el.getAttribute('data-id');
      await supabase.rpc('mark_notification_read', { p_id: id });
      window.location.href = 'notifications?id=' + encodeURIComponent(id);
    });
  });
}

function initNotifDropdown() {
  const bellBtn = document.getElementById('notif-bell-btn');
  const dropdown = document.getElementById('notif-dropdown');
  const bell = document.getElementById('notif-bell');
  const markAll = document.getElementById('notif-mark-all');
  const staffSend = document.getElementById('notif-staff-send');
  if (bellBtn && dropdown) {
    bellBtn.addEventListener('click', () => {
      const open = dropdown.classList.toggle('open');
      bellBtn.setAttribute('aria-expanded', open ? 'true' : 'false');
      if (open) loadNotifications();
    });
  }
  if (bell) {
    document.addEventListener('click', (e) => {
      if (!bell.contains(e.target)) dropdown && dropdown.classList.remove('open');
    });
  }
  if (markAll) {
    markAll.addEventListener('click', async () => {
      await supabase.rpc('mark_all_notifications_read');
      loadNotifications();
    });
  }
  if (staffSend) {
    staffSend.addEventListener('click', async () => {
      const input = document.getElementById('notif-staff-input');
      const title = input.value.trim();
      if (!title) return;
      const { error } = await supabase.rpc('send_broadcast_notification', { p_title: title, p_body_html: null, p_target_usernames: null });
      if (!error) { input.value = ''; loadNotifications(); }
    });
  }
}

async function initNotifBell(session) {
  const bell = document.getElementById('notif-bell');
  if (!bell) return;
  clearInterval(notifPollInterval);
  if (!session) { bell.style.display = 'none'; return; }
  bell.style.display = 'block';
  const { data: prof } = await supabase.from('profiles_public').select('badges').eq('id', session.user.id).maybeSingle();
  const isStaff = !!(prof && prof.badges && prof.badges.includes('staff'));
  const staffCompose = document.getElementById('notif-staff-compose');
  if (staffCompose) staffCompose.style.display = (isStaff || session.user.id === ADMIN_ID) ? 'flex' : 'none';
  loadNotifications();
  notifPollInterval = setInterval(loadNotifications, 60000);
}

function initThemeButton() {
  const btn = document.getElementById('theme-toggle');
  const icon = btn && btn.querySelector('[data-lucide]');
  if (!btn) return;
  initThemeToggle(btn);
  const sync = () => {
    const theme = currentEffectiveTheme();
    btn.setAttribute('data-theme-current', theme);
    btn.setAttribute('aria-label', theme === 'light' ? 'Switch to dark theme' : 'Switch to light theme');
    if (icon) { icon.setAttribute('data-lucide', theme === 'light' ? 'sun' : 'moon'); refreshIcons(); }
  };
  sync();
  btn.addEventListener('click', sync);
}

async function onSession(session) {
  renderUserMenu(session);
  checkBan(session);
  initNotifBell(session);
  window.dispatchEvent(new CustomEvent('poly:session', { detail: { session } }));
}

initMobileToggle();
initActiveLink();
initUserMenu();
initNotifDropdown();
initThemeButton();

supabase.auth.onAuthStateChange((_event, session) => onSession(session));
supabase.auth.getSession().then(({ data }) => onSession(data.session));
