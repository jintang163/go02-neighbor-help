const TOKEN_KEY = "nh_token";

function token() { return localStorage.getItem(TOKEN_KEY) || ""; }
function setToken(t) { localStorage.setItem(TOKEN_KEY, t); }
function clearToken() { localStorage.removeItem(TOKEN_KEY); }

async function api(path, opts = {}) {
  const headers = Object.assign({ "Content-Type": "application/json" }, opts.headers || {});
  const t = token();
  if (t) headers.Authorization = "Bearer " + t;
  const res = await fetch(path, Object.assign({}, opts, { headers }));
  if (res.status === 204) return null;
  const text = await res.text();
  let data = null;
  try { data = text ? JSON.parse(text) : null; } catch { data = { message: text }; }
  if (!res.ok) {
    const msg = (data && (data.message || data.code)) || ("HTTP " + res.status);
    throw new Error(msg);
  }
  return data;
}

function requireLogin() {
  if (!token()) { location.href = "/login"; return false; }
  return true;
}

async function currentMe() {
  return api("/api/auth/me");
}

function logout() {
  api("/api/auth/logout", { method: "POST" }).catch(() => {});
  clearToken();
  location.href = "/login";
}

function $(id) { return document.getElementById(id); }

function fillUserBar(me) {
  const el = $("userbar");
  if (!el || !me) return;
  const u = me.user || me;
  el.innerHTML = `<span>${u.display_name || u.username} · 信用 ${u.credit_score}（${u.credit_level}）</span>
    <a href="/app">广场</a> <a href="/me">我的</a>
    ${u.role === "admin" ? '<a href="/admin">后台</a>' : ""}
    <button class="btn btn-ghost" onclick="logout()">退出</button>`;
}

function catLabel(id) {
  const m = {grocery:"买菜代购",delivery:"代取快递",pet:"照看宠物",moving:"搬运重物",repair:"上门修理",escort:"陪同就医",childcare:"临时看护",other:"其他"};
  return m[id] || id;
}
function urgLabel(id) {
  const m = {low:"不急",normal:"普通",high:"较急",urgent:"紧急"};
  return m[id] || id;
}

window.NH = { api, token, setToken, clearToken, requireLogin, currentMe, logout, $, fillUserBar, catLabel, urgLabel };
