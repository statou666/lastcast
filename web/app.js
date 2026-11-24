const DICT = {
  en: {
    nav_updates: "Updates", nav_feeds: "Subscriptions", nav_playlist: "Playlist", nav_favorites: "Favorites",
    nav_downloads: "Downloads", nav_settings: "Settings",
    playlist_title: "Playlist", playlist_playall: "Play all", playlist_shuffle: "Shuffle", playlist_clear: "Clear",
    card_account: "Account", account_name: "Signed in as", logout: "Sign out", card_users: "Users", role_label: "Role",
    old_password: "Current password", new_password: "New password", change_password: "Change password",
    card_gpodder: "Podcast app (gPodder)", gpodder_server: "Sync server", gpodder_feeds: "Feed URLs (add these in the app)",
    gpodder_hint: "AntennaPod → Settings → Synchronization → gPodder: enter the server URL, your username and password. Then add one of the feed URLs below — video and text episodes play instantly through Lastcast.",
    copy: "Copy", copied: "Copied", import_done: "Imported {n} feed(s)",
    opml_pick: "Choose a local .opml file", opml_hint: "Pick a .opml file from this computer, or paste its content below.",
    opml_empty: "Choose a file or paste OPML content first.",
    detect_voices: "Detect voices", voices_found: "{n} voices found", voices_failed: "Detection failed — you can type a voice id",
    feed_fetch_fulltext: "Fetch full article text by default", get_fulltext: "Fetch full text", fulltext_done: "Full text saved",
    voices_click: "Detected voices — click to use:", images: "Images", select: "Select", select_all: "All", select_none: "None",
    select_exit: "Done", selected: "selected", confirm_delete_n: "Delete {n} selected item(s)?", kind: "Type",
    password_changed: "Password updated", proxy_url: "HTTP proxy (optional)",
    add_user: "Add user", role_user: "User", role_admin: "Administrator", users_admin_note: "Only administrators can change instance settings.",
    category: "Category", category_none: "Uncategorized", edit_feed: "Edit", save_feed: "Save",
    feed_url: "Feed URL", feed_auto_download: "Auto-download new episodes of this feed",
    updated: "Updated", published: "Published", load_more: "Load more", just_now: "just now", ago_min: "{n} min ago", ago_hour: "{n} h ago", ago_day: "{n} d ago",
    timeline_today: "Today", timeline_yesterday: "Yesterday", edit_note: "URL changes trigger an immediate refetch.",
    playlist_empty: "Playlist is empty", fav_title: "Favorites", fav_empty: "No favorites yet",
    fav: "Favorite", unfav: "Unfavorite", qnext: "Play next", fav_auto_download: "Auto-download favorited episodes",
    downloads_title: "Downloads", dl_empty: "Nothing downloaded yet", dl_play: "Play", dl_del: "Delete",
    cache_title: "Cache", cache_clear: "Clear cache", cache_evict: "Trim to limit", card_lang: "Language",
    play_mode: "Video playback", play_mode_stream: "Stream (no disk)", play_mode_segment: "Cache segments",
    cache_dir: "Cache dir", download_dir: "Download dir", cache_limit: "Cache limit (MB, 0 = off)",
    download_format: "Download format", fmt_auto: "Auto (source)",
    dl_retag: "Re-tag / covers",
    downloads_used: "Downloads", cache_used: "Cache", deleted_kept: "subscription removed; downloads kept",
    filter: "Feed", add: "Add", opml: "OPML", export: "Export", import: "Import",
    chip_all: "All", chip_listening: "Listening", chip_new: "Unplayed", chip_done: "Finished", chip_downloaded: "Downloaded",
    queue_title: "Up next", queue_clear: "Clear", play: "Play", pause: "Pause", mute: "Mute", queue: "Queue",
    dl: "Download", del: "Delete", refetch: "Refetch", pack: "Pack", done: "Done", notes: "Show notes",
    card_runtime: "Runtime settings", card_sync: "Sync", card_env: "Environment & stats",
    fetch_interval: "Fetch interval (min)", concurrency: "Download concurrency", conflict_rule: "Conflict rule",
    auto_download: "Auto-download new episodes", storage_limit: "Storage limit (MB, 0 = off)", ui_theme: "Theme",
    device_name: "This device", sync_now: "Sync now", save: "Save", sleep_off: "Sleep off", sleep_end: "End of episode",
    env_precedence: "Env vars set defaults; a value saved here overrides that key. PORT/DATA_DIR are process-level.",
    status_new: "new", status_downloaded: "downloaded", status_error: "error", episodes_count: "episodes",
    view_grid: "Grid", view_list: "List", volume: "Volume",
    kind_auto: "Auto", kind_audio: "Audio", kind_video: "Video", kind_text: "Text",
    card_tts: "Text-to-speech", tts_provider: "Provider", tts_base: "Base URL", tts_key: "API key",
    tts_model: "Model", tts_voice: "Voice", tts_speed: "Speed", tts_max: "Max chars per request",
    tts_test: "Test", tts_render: "Read aloud",
    subscribe_one: "Subscribe one feed", kind_hint: "Type: auto detects it from the feed; choosing text makes every entry speakable.",
    back: "Back", not_found: "Not found", chunks: "chunks", segments: "chunks/segments", segment: "segment", not_ready: "still generating…", open_detail: "Open episode", tts_auto: "Auto-speak new text items",
  },
  zh: {
    nav_updates: "更新", nav_feeds: "订阅", nav_playlist: "播放列表", nav_favorites: "收藏",
    nav_downloads: "下载", nav_settings: "设置",
    playlist_title: "播放列表", playlist_playall: "播放全部", playlist_shuffle: "随机", playlist_clear: "清空",
    card_account: "账户", account_name: "当前登录", logout: "退出登录", card_users: "用户", role_label: "角色",
    old_password: "当前密码", new_password: "新密码", change_password: "修改密码",
    card_gpodder: "播客客户端（gPodder）", gpodder_server: "同步服务器", gpodder_feeds: "源地址（在客户端里添加）",
    gpodder_hint: "AntennaPod → 设置 → 同步 → gPodder：填同步服务器、用户名和密码；然后在客户端里添加下面任意一个源地址。视频和文字条目都会通过 Lastcast 即时播放。",
    copy: "复制", copied: "已复制", import_done: "已导入 {n} 个源",
    opml_pick: "选择本机 .opml 文件", opml_hint: "从电脑里选一个 .opml 文件，或把内容粘贴到下面。",
    opml_empty: "请先选择文件或粘贴 OPML 内容。",
    detect_voices: "检测音色", voices_found: "检测到 {n} 个音色", voices_failed: "检测失败，可直接填写音色 ID",
    feed_fetch_fulltext: "默认抓取全文", get_fulltext: "获取全文", fulltext_done: "全文已保存",
    voices_click: "检测到的音色（点击选用）：", images: "图片", select: "多选", select_all: "全选", select_none: "取消选择",
    select_exit: "完成", selected: "已选", confirm_delete_n: "确定删除选中的 {n} 项？", kind: "类型",
    password_changed: "密码已更新", proxy_url: "代理地址（可选）",
    add_user: "添加用户", role_user: "普通用户", role_admin: "管理员", users_admin_note: "只有管理员能修改实例设置。",
    category: "分类", category_none: "未分类", edit_feed: "编辑", save_feed: "保存",
    feed_url: "源地址", feed_auto_download: "此源更新时自动下载新集",
    updated: "更新于", published: "发布于", load_more: "加载更多", just_now: "刚刚", ago_min: "{n} 分钟前", ago_hour: "{n} 小时前", ago_day: "{n} 天前",
    timeline_today: "今天", timeline_yesterday: "昨天", edit_note: "修改地址后会立即重新抓取。",
    playlist_empty: "播放列表是空的", fav_title: "收藏", fav_empty: "还没有收藏",
    fav: "收藏", unfav: "取消收藏", qnext: "下一首播放", fav_auto_download: "收藏单集自动下载",
    downloads_title: "下载管理", dl_empty: "还没有下载", dl_play: "播放", dl_del: "删除",
    cache_title: "缓存", cache_clear: "清空缓存", cache_evict: "按上限清理", card_lang: "语言",
    play_mode: "视频播放方式", play_mode_stream: "流式（不落盘）", play_mode_segment: "缓存分段",
    cache_dir: "缓存目录", download_dir: "下载目录", cache_limit: "缓存上限（MB，0=关闭）",
    download_format: "下载格式", fmt_auto: "自动（跟源）",
    dl_retag: "补封面/标签",
    downloads_used: "下载", cache_used: "缓存", deleted_kept: "订阅已删除，下载已保留",
    filter: "订阅源", add: "添加", opml: "OPML", export: "导出", import: "导入",
    chip_all: "全部", chip_listening: "在听", chip_new: "未听", chip_done: "已听完", chip_downloaded: "已下载",
    queue_title: "接下来播放", queue_clear: "清空", play: "播放", pause: "暂停", mute: "静音", queue: "队列",
    dl: "下载", del: "删除", refetch: "重新抓取", pack: "打包", done: "听完", notes: "节目笔记",
    card_runtime: "运行参数", card_sync: "同步", card_env: "环境变量与统计",
    fetch_interval: "抓取间隔（分钟）", concurrency: "下载并发", conflict_rule: "冲突规则",
    auto_download: "新集自动下载", storage_limit: "存储上限（MB，0 = 关闭）", ui_theme: "主题",
    device_name: "本设备名称", sync_now: "立即同步", save: "保存", sleep_off: "不定时的", sleep_end: "本集播完即停",
    env_precedence: "环境变量提供默认值；在此页保存过的设置会覆盖该键的环境变量。PORT/DATA_DIR 属进程级配置，不受影响。",
    status_new: "新", status_downloaded: "已下载", status_error: "错误", episodes_count: "单集",
    view_grid: "海报", view_list: "列表", volume: "音量",
    kind_auto: "自动", kind_audio: "音频", kind_video: "视频", kind_text: "文字",
    card_tts: "语音合成（TTS）", tts_provider: "服务商", tts_base: "Base URL", tts_key: "密钥",
    tts_model: "模型", tts_voice: "音色", tts_speed: "语速", tts_max: "每次请求最大字符数",
    tts_test: "测试", tts_render: "朗读",
    subscribe_one: "订阅单个源", kind_hint: "类型：自动＝从源自动判断；选文字后每条内容都能朗读。", tts_auto: "自动朗读新文字条目",
    back: "返回", not_found: "未找到", chunks: "分段", segments: "分段",
    segment: "段", not_ready: "正在生成，稍后重试", open_detail: "打开本集",
  },
};

let lang = localStorage.getItem("lang") || (navigator.language.startsWith("zh") ? "zh" : "en");
// The device id must survive reloads, otherwise progress/playlist/favorites
// silently belong to a brand-new "device" after every refresh.
const DEVICE = (() => {
  let d = localStorage.getItem("device");
  if (!d) {
    d = "dev-" + Math.random().toString(36).slice(2, 8);
    localStorage.setItem("device", d);
  }
  return d;
})();
const DEVICE_NAME = localStorage.getItem("device_name") || navigator.platform || "device";
const t = (k) => DICT[lang][k] || DICT.en[k] || k;
const esc = (s) => String(s == null ? "" : s).replace(/[&<>"'"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
const strip = (s) => String(s == null ? "" : s).replace(/<[^>]*>/g, "").trim();
const api = (path, opts) => fetch(path, opts).then((r) => {
  if (r.status === 401) { location.replace("/login"); throw new Error("unauthorized"); }
  return r.json();
});
const flash = (msg) => {
  for (const id of ["add-msg", "tts-msg"]) {
    const el = document.getElementById(id);
    if (el) el.textContent = msg;
  }
};
const fmtTime = (s) => { s = Math.floor(s || 0); return String(Math.floor(s / 60)).padStart(2, "0") + ":" + String(s % 60).padStart(2, "0"); };
const fmtDur = (s) => (s ? Math.floor(s / 60) + "m" : "");
const fmtBytes = (n) => (n > 1048576 ? (n / 1048576).toFixed(1) + "MB" : n > 1024 ? (n / 1024).toFixed(1) + "KB" : n + "B");
const ago = (k, n) => t(k).replace("{n}", n);
function timeAgo(sec) {
  if (!sec) return "";
  const d = Math.max(0, Math.floor(Date.now() / 1000) - sec);
  if (d < 60) return t("just_now");
  if (d < 3600) return ago("ago_min", Math.floor(d / 60));
  if (d < 86400) return ago("ago_hour", Math.floor(d / 3600));
  if (d < 86400 * 7) return ago("ago_day", Math.floor(d / 86400));
  return new Date(sec * 1000).toLocaleDateString();
}
// The entry's "when did this arrive/change in my library" timestamp.
const tsOf = (e) => e.published_at || 0;
function dayLabel(sec) {
  if (!sec) return "";
  const d = new Date(sec * 1000), now = new Date();
  const same = (a, b) => a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
  if (same(d, now)) return t("timeline_today");
  const y = new Date(now.getTime() - 86400000);
  if (same(d, y)) return t("timeline_yesterday");
  return d.toLocaleDateString();
}

/* ---------- persistent player (never torn down by view switching) ---------- */
const audio = document.getElementById("audio");
const barTitle = document.getElementById("bar-title");
const barFill = document.getElementById("bar-fill");
const barTime = document.getElementById("bar-time");
const barRem = document.getElementById("bar-rem");
const pcover = document.getElementById("pcover");
const playBtn = document.getElementById("playpause");
let current = null;
let queue = [];
let sleepAt = 0, sleepMode = "";
let segIdx = 0, segTotal = 0;

async function loadQueue() {
  const r = await api("/api/queue?device_id=" + encodeURIComponent(DEVICE));
  queue = Array.isArray(r) ? r : [];
  renderQueue();
}

function renderQueue() {
  const box = document.getElementById("queue");
  if (!box) return;
  const panel = document.getElementById("queuebox");
  panel.hidden = !queue.length;
  box.innerHTML = queue.map((ep, i) => `<div class="qrow">
    <span class="qnum">${i + 1}</span>
    <span class="qtitle">${esc(ep.title)}</span>
    <button data-act="qplay" data-id="${ep.id}">▶</button>
    <button data-act="qdel" data-id="${ep.queue_id}">✕</button>
  </div>`).join("");
}

function applyLang() {
  document.querySelectorAll("[data-i18n]").forEach((el) => (el.textContent = t(el.dataset.i18n)));
  document.documentElement.lang = lang === "zh" ? "zh-CN" : "en";
  document.querySelectorAll(".lang button").forEach((b) => b.classList.toggle("active", b.dataset.lang === lang));
}

let pendingSeek = 0;
let playToken = 0;

function saveLastEpisode() {
  if (!current || !current.id) return;
  localStorage.setItem("last_episode", JSON.stringify({
    id: current.id, title: current.title, position: Math.floor(current.position || 0),
    url: current.file ? "/media/" + current.id : current.enclosure_url,
    cover: current.cover_url || "", dl: current.dl_file || "",
  }));
}

// Report the current position; keepalive survives page close / mobile backgrounding.
function reportProgress(extra, keepalive) {
  if (!current || !current.id) return;
  const pos = Math.floor(audio.currentTime || current.position || 0);
  current.position = pos;
  saveLastEpisode();
  const payload = Object.assign({ episode_id: current.id, device_id: DEVICE, position_sec: pos, segment_idx: segIdx }, extra || {});
  fetch("/api/progress", {
    method: "POST", headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload), keepalive: !!keepalive,
  }).catch(() => {});
}

function setBar(ep) {
  playToken++;
  const prev = current && current.id;
  if (prev && prev !== ep.id) {
    // Switching episodes must silence the old one immediately, before any
    // segment lookup / network wait for the new one.
    try { audio.pause(); } catch (e) {}
    try { audio.removeAttribute("src"); audio.load(); } catch (e) {}
    pendingSeek = 0;
  }
  current = ep;
  current._posLoaded = false;
  if (segEp(ep)) { segTotal = ep.segments || segTotal; segIdx = ep.segment_idx || segIdx || 1; }
  barTitle.textContent = ep.title || "";
  pcover.src = ep.cover_url || "";
  // The playing row is a link to the episode detail page too.
  pcover.dataset.act = "openepisode";
  pcover.dataset.id = ep.id;
  pcover.setAttribute("role", "link");
  pcover.setAttribute("aria-label", t("open_detail") + ": " + ep.title);
  barTitle.setAttribute("role", "link");
  barTitle.setAttribute("aria-label", t("open_detail") + ": " + ep.title);
  barTitle.dataset.act = "openepisode";
  barTitle.dataset.id = ep.id;
  renderBar();
  localStorage.setItem("last_episode", JSON.stringify({ id: ep.id, title: ep.title, position: ep.position, url: ep.file ? "/media/" + ep.id : ep.enclosure_url, cover: ep.cover_url, dl: ep.dl_file || "" }));
}

function renderBar() {
  const pos = audio.currentTime || (current ? current.position : 0);
  const dur = (current && current.duration_sec) || audio.duration || 0;
  const seg = current && segEp(current) && segTotal;
  barFill.style.width = seg ? Math.min(100, (segIdx / segTotal) * 100) + "%" : (dur ? Math.min(100, (pos / dur) * 100) + "%" : "0%");
  barTime.textContent = seg ? `${t("segment")} ${segIdx}/${segTotal} · ${fmtTime(pos)}` : fmtTime(pos);
  barRem.textContent = dur ? "−" + fmtTime(Math.max(0, dur - pos)) : "";
}

// Segment mode applies to text renders and to video audio cuts already in cache.
// Everything else streams straight from the origin through /stream/{id}.
const segEp = (ep) => !!ep && !ep.file && (ep.segments || 0) > 0 && (ep.kind === "text" || ep.kind === "video");

function source(ep) {
  if (ep.dl_file) return ep.dl_file;
  if (ep.file) return "/media/" + ep.id;
  if (segEp(ep)) return "/media/" + ep.id + "/" + (segIdx || ep.segment_idx || 1);
  return "/stream/" + ep.id;
}

// Segments are written one by one, so a long text item can start playing on
// segment 1 instead of waiting for the whole episode to be rendered.
async function segmentInfo(id) {
  const r = await api("/api/segments/" + id);
  return r && !r.error ? r : null;
}

async function waitSegment(n) {
  const waitingFor = current && current.id;
  for (let i = 0; i < 40; i++) {
    if (!current || current.id !== waitingFor) return false;
    const r = await segmentInfo(current.id);
    if (r) {
      segTotal = r.segments || segTotal;
      if (current) current.segments = segTotal;
      if ((r.ready || []).indexOf(n) >= 0) { segIdx = n; return true; }
    }
    if (i >= 20) return false;
    await new Promise((res) => setTimeout(res, 1500));
  }
  return false;
}

async function play() {
  if (!current) return;
  const token = playToken;
  try { audio.pause(); } catch (e) {}
  if (current.id && !current._posLoaded) {
    try {
      const fresh = await api("/api/episodes/" + current.id + "?device_id=" + encodeURIComponent(DEVICE));
      if (fresh && !fresh.error) {
        current.position = Math.max(current.position || 0, fresh.position || 0);
        if (fresh.duration_sec) current.duration_sec = fresh.duration_sec;
        if (fresh.file) current.file = fresh.file;
        if (fresh.cover_url) current.cover_url = fresh.cover_url;
      }
    } catch (e) { /* offline: play from the cached position */ }
    current._posLoaded = true;
    if (token !== playToken) return;
  }
  if (current.kind === "text" && !current.file && !current.dl_file) {
    const info = await segmentInfo(current.id);
    if (info) segTotal = info.segments || segTotal;
    if (!info || !(info.ready || []).length) {
      // Render the paragraph cache only — playing must never create a download.
      await api("/api/episodes/" + current.id + "/segments", { method: "POST" });
    }
    if (!(await waitSegment(segIdx || current.segment_idx || 1))) {
      flash(t("not_ready")); renderBar(); return;
    }
    if (token !== playToken) return;
  }
  if (token !== playToken) return;
  const src = source(current);
  if (!src) return;
  const abs = new URL(src, location.origin).href;
  if (audio.src !== abs) { pendingSeek = current.position || 0; audio.src = abs; }
  audio.playbackRate = parseFloat(document.getElementById("speed").value) || 1;
  audio.play().catch(() => {});
  playBtn.textContent = t("pause");
  if ("mediaSession" in navigator) {
    navigator.mediaSession.setActionHandler("play", play);
    navigator.mediaSession.setActionHandler("pause", () => audio.pause());
    navigator.mediaSession.setActionHandler("previoustrack", () => jump(-1));
    navigator.mediaSession.setActionHandler("nexttrack", () => jump(1));
    navigator.mediaSession.metadata = new MediaMetadata({ title: current.title, album: current.feed_title || "" });
  }
}

async function jump(dir) {
  if (!queue.length) return;
  const ep = queue[0];
  if (dir > 0) {
    await api("/api/queue/" + ep.queue_id + "?device_id=" + encodeURIComponent(DEVICE), { method: "DELETE" });
    await loadQueue();
  }
  setBar(ep);
  play();
}

function sleepTimer() {
  const v = document.getElementById("sleep").value;
  if (!v || v === "0") { sleepMode = ""; sleepAt = 0; return; }
  sleepMode = v;
  sleepAt = v === "end" ? 0 : Date.now() + Number(v) * 1000;
}

audio.addEventListener("loadedmetadata", () => {
  if (pendingSeek > 0 && isFinite(audio.duration)) {
    try { audio.currentTime = pendingSeek; } catch (e) {}
    pendingSeek = 0;
  }
  if (!current || !isFinite(audio.duration) || audio.duration <= 0) return;
  if (current.duration_sec) return;
  current.duration_sec = Math.round(audio.duration);
  if (!current.id) return;
  api("/api/progress", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ episode_id: current.id, device_id: DEVICE, position_sec: Math.floor(current.position || 0), duration_sec: current.duration_sec }) });
  renderBar();
});

// A browser cannot demux every container (bilibili DASH .m4s): fall back to the
// ffmpeg segment cache for that episode, then play segment 1.
audio.addEventListener("error", async () => {
  if (!current || !current.id) return;
  if (!String(audio.src || "").includes("/stream/")) return;
  flash(t("not_ready"));
  await api("/api/download", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ episode_id: current.id }) });
  for (let i = 0; i < 40; i++) {
    await new Promise((res) => setTimeout(res, 1500));
    const ep = await api("/api/episodes/" + current.id + "?device_id=" + encodeURIComponent(DEVICE));
    if (ep && ep.file) {
      current.file = ep.file;
      if (ep.duration_sec) current.duration_sec = ep.duration_sec;
      audio.src = new URL(source(current), location.origin).href;
      audio.currentTime = current.position || 0;
      audio.play().catch(() => {});
      return;
    }
    if (ep && ep.status === "error") { flash(t("not_ready")); return; }
  }
});

audio.addEventListener("timeupdate", () => {
  renderBar();
  if (!current) return;
  const pos = Math.floor(audio.currentTime);
  current.position = pos;
  if (pos - (current._rep || 0) >= 5 || audio.ended) {
    current._rep = pos;
    reportProgress({ completed: audio.ended && (!segEp(current) || segIdx >= segTotal) ? 1 : 0 });
  }
  if (sleepAt && Date.now() >= sleepAt && !audio.paused) { audio.pause(); playBtn.textContent = t("play"); sleepAt = 0; }
});

audio.addEventListener("pause", () => { if (!audio.ended) reportProgress({}); });
window.addEventListener("pagehide", () => reportProgress({}, true));
document.addEventListener("visibilitychange", () => { if (document.visibilityState === "hidden") reportProgress({}, true); });

audio.addEventListener("ended", async () => {
  if (current && segEp(current) && segTotal && segIdx < segTotal) {
    const next = segIdx + 1;
    if (await waitSegment(next)) { audio.src = new URL(source(current), location.origin).href; audio.currentTime = 0; audio.play().catch(() => {}); renderBar(); return; }
    playBtn.textContent = t("play");
    return;
  }
  if (queue.length) { const ep = queue[0]; await api("/api/queue/" + ep.queue_id + "?device_id=" + encodeURIComponent(DEVICE), { method: "DELETE" }); await loadQueue(); setBar(ep); play(); return; }
  playBtn.textContent = t("play");
  if (sleepMode === "end") document.getElementById("sleep").value = "off";
});

playBtn.addEventListener("click", () => {
  if (audio.paused) play(); else audio.pause();
  playBtn.textContent = audio.paused ? t("play") : t("pause");
});
document.getElementById("rewind").addEventListener("click", () => { audio.currentTime = Math.max(0, audio.currentTime - 30); });
document.getElementById("forward").addEventListener("click", () => { audio.currentTime = Math.min(audio.duration || 0, audio.currentTime + 30); });
document.getElementById("speed").addEventListener("change", () => { audio.playbackRate = parseFloat(document.getElementById("speed").value) || 1; });
document.getElementById("sleep").addEventListener("change", sleepTimer);
document.getElementById("mute").addEventListener("click", () => { audio.muted = !audio.muted; document.getElementById("mute").classList.toggle("active", audio.muted); });
const volEl = document.getElementById("volume");
const volNum = document.getElementById("volnum");
function applyVolume(v) {
  v = Math.max(0, Math.min(100, Number(v) || 0));
  audio.volume = v / 100;
  if (volEl) volEl.value = v;
  if (volNum) volNum.textContent = String(v);
  localStorage.setItem("volume", String(v));
  if (volEl) volEl.title = t("volume") + " " + v;
  if (volEl) volEl.classList.toggle("low", v < 100);
}
if (volEl) {
  applyVolume(localStorage.getItem("volume") || 100);
  volEl.addEventListener("input", () => applyVolume(volEl.value));
}
document.getElementById("bar-progress").addEventListener("click", (e) => {
  if (!audio.duration) return;
  const r = e.currentTarget.getBoundingClientRect();
  audio.currentTime = Math.max(0, Math.min(audio.duration, ((e.clientX - r.left) / r.width) * audio.duration));
  renderBar();
});
setInterval(() => { if (sleepAt && Date.now() >= sleepAt && !audio.paused) { audio.pause(); playBtn.textContent = t("play"); sleepAt = 0; } }, 5000);
document.addEventListener("keydown", (e) => {
  if (e.target.matches("input, select, textarea")) return;
  if (e.code === "Space") { e.preventDefault(); playBtn.click(); }
  if (e.code === "ArrowLeft") document.getElementById("rewind").click();
  if (e.code === "ArrowRight") document.getElementById("forward").click();
});

/* ---------- router: switching views never touches the player ---------- */
let selectMode = false;
let picked = new Set();
let lastFeeds = [];
let lastDl = [];
let lastView = "";

function enterSelect() {
  if (selectMode) { exitSelect(); return; }
  selectMode = true;
  picked.clear();
  renderSelBar();
  refreshView();
}

function exitSelect() {
  selectMode = false;
  picked.clear();
  renderSelBar();
  refreshView();
}

function selActionsFor(view) {
  switch (view) {
    case "updates":
    case "feed":
      return [["download", t("dl")], ["queue", t("queue")], ["favorite", t("fav")], ["done", t("done")]];
    case "feeds":
      return [["delfeed", t("del")]];
    case "playlist":
      return [["delqueue", t("del")]];
    case "favorites":
      return [["unfavorite", t("unfav")]];
    case "downloads":
      return [["deldl", t("del")]];
  }
  return [];
}

function renderSelBar() {
  const bar = document.getElementById("selbar");
  if (!bar) return;
  document.body.classList.toggle("selecting", selectMode);
  bar.hidden = !selectMode;
  if (!selectMode) return;
  const c = document.getElementById("selcount");
  if (c) c.textContent = picked.size + " " + t("selected");
  const box = document.getElementById("selactions");
  if (box) box.innerHTML = selActionsFor(route().view).map(([op, label]) => `<button data-act="batch" data-op="${op}">${label}</button>`).join("");
}

function pickBox(id) {
  return selectMode ? `<input type="checkbox" class="pick" data-act="pick" data-id="${id}" ${picked.has(id) ? "checked" : ""}>` : "";
}

function route() {
  const p = (location.hash || "#/updates").slice(2).split("/");
  return { view: p[0] || "updates", id: p[1] || "" };
}

function refreshView() {
  const r = route();
  show(r.view, r.id);
}

let toastTimer = 0;
function toast(msg) {
  if (!msg) return;
  let el = document.getElementById("toast");
  if (!el) {
    el = document.createElement("div");
    el.id = "toast";
    document.body.appendChild(el);
  }
  el.textContent = msg;
  el.classList.add("show");
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => el.classList.remove("show"), 1800);
}

function show(view, id) {
  if (view !== lastView) { lastView = view; if (selectMode) { selectMode = false; picked.clear(); } }
  renderSelBar();
  const domView = view === "feed" || view === "episode" ? "detail" : view;
  const nav = view === "feed" ? "feeds" : view === "episode" ? "updates" : view;
  document.querySelectorAll(".view").forEach((v) => (v.hidden = v.id !== "view-" + domView));
  document.querySelectorAll(".navlink").forEach((a) => a.classList.toggle("active", a.dataset.view === nav));
  if (view === "updates") renderUpdates();
  else if (view === "feeds") renderFeeds();
  else if (view === "playlist") renderPlaylist();
  else if (view === "favorites") renderFavorites();
  else if (view === "downloads") renderDownloads();
  else if (view === "settings") { renderSettings(); renderGpodder(); }
  else if (view === "feed") renderFeedDetail(id);
  else if (view === "episode") renderEpisodeDetail(id);
}
window.addEventListener("hashchange", () => {
  const parts = (location.hash || "#/updates").slice(2).split("/");
  show(parts[0], parts[1]);
});

/* ---------- updates view ---------- */
let filterFeed = "", filterStatus = "all", lastEps = [];

function renderEpisodes() {
  const list = document.getElementById("episodes");
  const eps = lastEps.filter((e) => {
    if (filterStatus === "all") return true;
    if (filterStatus === "downloaded") return e.status === "downloaded";
    if (filterStatus === "done") return e.completed;
    if (filterStatus === "inprogress") return !e.completed && e.position > 0;
    return !e.completed && !e.position;
  });
  let html = "", lastDay = "";
  for (const e of eps.slice(0, 200)) {
    const day = dayLabel(tsOf(e));
    if (day && day !== lastDay) { html += `<li class="daysep">${day}</li>`; lastDay = day; }
    html += rowHtml(e);
  }
  list.innerHTML = html || `<p class="mut">—</p>`;
  renderSelBar();
}

function rowHtml(e) {
  return `
    <li class="ep${selectMode ? " has-pick" : ""}" data-id="${e.id}">
      ${pickBox(e.id)}
      ${e.cover_url ? `<img class="cover" data-act="openepisode" data-id="${e.id}" src="${esc(e.cover_url)}" alt="" loading="lazy" referrerpolicy="no-referrer" onerror="this.removeAttribute('src')">` : `<span class="cover ph" data-act="openepisode" data-id="${e.id}">${esc((e.feed_title || e.title || "?").slice(0, 1))}</span>`}
      <div class="t">
        <a class="title" data-act="openepisode" data-id="${e.id}" href="#/episode/${e.id}">${esc(e.title)}</a>
        <div class="sub">${[esc(e.feed_title || ""), fmtDur(e.duration_sec), tsOf(e) ? `<span class="time">${t("published")} ${timeAgo(tsOf(e))}</span>` : "", e.kind && e.kind !== "audio" ? `<span class="badge kind">${t("kind_" + e.kind)}</span>` : "", `<span class="badge ${e.status === "downloaded" ? "dl" : e.status === "error" ? "err" : "new"}">${t("status_" + e.status) || e.status}</span>`, e.completed ? `<span class="badge done">${t("chip_done")}</span>` : ""].filter(Boolean).join(" · ")}</div>
        <div class="bar"><i style="width:${e.duration_sec ? Math.min(100, (e.position / e.duration_sec) * 100) : 0}%"></i></div>
        <div class="notes" data-notes="${e.id}" hidden>${esc(e.description || "")}</div>
      </div>
      <div class="actions">
        <button data-act="play" data-id="${e.id}" title="${t("play")}">▶</button>
        ${e.status === "downloaded" ? "" : `<button data-act="dl" data-id="${e.id}" title="${t("dl")}">⤓</button>`}
        <button data-act="q" data-id="${e.id}" title="${t("queue")}">＋</button>
        <button data-act="fav" data-id="${e.id}" class="${e.favorite ? "active" : ""}" title="${e.favorite ? t("unfav") : t("fav")}">${e.favorite ? "★" : "☆"}</button>
        <button data-act="done" data-id="${e.id}" title="${t("done")}">✓</button>
      </div>
    </li>`;
}

let updLimit = 300;
async function renderUpdates() {
  const q = new URLSearchParams({ device_id: DEVICE, limit: String(updLimit) });
  if (filterFeed) q.set("feed_id", filterFeed);
  const [feeds, r] = await Promise.all([api("/api/feeds"), api("/api/sync?" + q)]);
  const sel = document.getElementById("filter-feed");
  sel.innerHTML = `<option value="">${t("filter")}: ${t("chip_all")}</option>` + (feeds || []).map((f) => `<option value="${f.id}">${esc(f.title)}</option>`).join("");
  sel.value = filterFeed;
  lastEps = (r && r.episodes) || [];
  const more = document.getElementById("ep-more");
  if (more) more.hidden = !(r && r.has_more);
  await loadQueue();
  renderEpisodes();
}
const epMoreBtn = document.getElementById("ep-more");
if (epMoreBtn) epMoreBtn.addEventListener("click", () => { updLimit += 300; renderUpdates(); });

/* ---------- subscriptions view (poster grid / list, Navidrome-style) ---------- */
let feedMode = localStorage.getItem("feedmode") || "grid";

async function renderFeeds() {
  const feeds = await api("/api/feeds");
  lastFeeds = Array.isArray(feeds) ? feeds : [];
  const box = document.getElementById("feedcards");
  box.className = "catgroups";
  const actions = (f) => `<div class="actions">
      <button data-act="refetch" data-id="${f.id}">${t("refetch")}</button>
      <button data-act="pack" data-id="${f.id}">${t("pack")}</button>
      <button data-act="del" data-id="${f.id}">${t("del")}</button></div>`;
  const badge = (f) => `<span class="badge kind" title="${esc(f.setting || "auto")}">${t("kind_" + (f.kind || "auto"))}</span>`;
  const card = (f) => {
    const meta = `${f.episodes || 0} ${t("episodes_count")} · ${f.downloaded || 0} ${t("status_downloaded")}`;
    const autoTag = f.auto_download ? ` <span class="badge dl" title="${t("feed_auto_download")}">⤓ auto</span>` : "";
    if (feedMode === "grid") {
      const art = f.cover_url
        ? `<a class="plink" href="#/feed/${f.id}" data-act="openfeed" data-id="${f.id}"><img class="poster" loading="lazy" src="${esc(f.cover_url)}" alt="" referrerpolicy="no-referrer" onerror="this.removeAttribute('src')"></a>`
        : `<a class="plink" href="#/feed/${f.id}" data-act="openfeed" data-id="${f.id}"><span class="poster ph">${esc(f.title || "?")}</span></a>`;
      return `<div class="pcard" data-id="${f.id}">${pickBox(f.id)}${art}
        <a class="name" data-act="openfeed" data-id="${f.id}" href="#/feed/${f.id}">${esc(f.title || f.url)}</a>
        <div class="meta">${badge(f)} ${meta}${autoTag}</div>
        ${actions(f)}</div>`;
    }
    return `<div class="ep${selectMode ? " has-pick" : ""}" data-id="${f.id}">
      ${pickBox(f.id)}
      ${f.cover_url ? `<img class="cover" loading="lazy" data-act="openfeed" data-id="${f.id}" src="${esc(f.cover_url)}" alt="" referrerpolicy="no-referrer" onerror="this.removeAttribute('src')">` : `<span class="cover ph" data-act="openfeed" data-id="${f.id}">${esc((f.title || "?").slice(0, 1))}</span>`}
      <div class="t"><div class="title" data-act="openfeed" data-id="${f.id}">${esc(f.title || f.url)}</div><div class="sub">${esc(f.url)}</div></div>
      ${actions(f)}</div>`;
  };
  // group by category: named groups first (alphabetical), uncategorized last
  const groups = new Map();
  for (const f of feeds || []) {
    const c = (f.category || "").trim();
    if (!groups.has(c)) groups.set(c, []);
    groups.get(c).push(f);
  }
  const names = [...groups.keys()].sort((a, b) => (a === "" ? 1 : b === "" ? -1 : a.localeCompare(b)));
  renderSelBar();
  box.innerHTML = names.map((c) => {
    const head = `<h3 class="cathead">${c ? esc(c) : t("category_none")} <span class="mut">${groups.get(c).length}</span></h3>`;
    const inner = `<div class="${feedMode === "grid" ? "grid" : "feedlist"}">${groups.get(c).map(card).join("")}</div>`;
    return `<section class="catgroup">${head}${inner}</section>`;
  }).join("") || `<p class="mut">—</p>`;
  document.querySelectorAll("#feedmode button").forEach((b) => b.classList.toggle("active", b.dataset.mode === feedMode));
}

function setFeedMode(mode) {
  feedMode = mode === "list" ? "list" : "grid";
  localStorage.setItem("feedmode", feedMode);
  renderFeeds();
}
document.querySelectorAll("#feedmode button").forEach((b) => b.addEventListener("click", () => setFeedMode(b.dataset.mode)));

/* ---------- playlist view (music-player queue: order, play next, shuffle) ---------- */
async function renderPlaylist() {
  const list = await api("/api/queue?device_id=" + encodeURIComponent(DEVICE));
  queue = Array.isArray(list) ? list : [];
  renderQueue();
  const box = document.getElementById("playlist");
  if (!box) return;
  renderSelBar();
  box.innerHTML = queue.map((ep, i) => `<li class="ep${selectMode ? " has-pick" : ""}" data-id="${ep.id}">
      ${pickBox(ep.queue_id)}
      <span class="coverwrap">${ep.cover_url
        ? `<img class="cover" loading="lazy" src="${esc(ep.cover_url)}" alt="" referrerpolicy="no-referrer" onerror="this.style.visibility='hidden'">`
        : `<span class="cover ph"></span>`}<span class="qnum">${i + 1}</span></span>
      <div class="t">
        <a class="title" data-act="openepisode" data-id="${ep.id}" href="#/episode/${ep.id}">${esc(ep.title)}</a>
        <div class="sub">${[esc(ep.feed_title || ""), fmtDur(ep.duration_sec), ep.completed ? t("chip_done") : ""].filter(Boolean).join(" · ")}</div>
      </div>
      <div class="actions">
        <button data-act="plplay" data-id="${ep.queue_id}">▶</button>
        <button data-act="plup" data-id="${ep.queue_id}">↑</button>
        <button data-act="pldown" data-id="${ep.queue_id}">↓</button>
        <button data-act="pldel" data-id="${ep.queue_id}">✕</button>
      </div></li>`).join("") || `<p class="mut">${t("playlist_empty")}</p>`;
}

/* ---------- favorites view (RSS-reader star; no audio unless configured) ---------- */
async function renderFavorites() {
  const list = await api("/api/favorites");
  lastEps = Array.isArray(list) ? list : [];
  const box = document.getElementById("favorites");
  if (!box) return;
  box.innerHTML = lastEps.map(rowHtml).join("") || `<p class="mut">${t("fav_empty")}</p>`;
}

/* ---------- downloads view (independent of subscriptions) ---------- */
async function renderDownloads() {
  const list = await api("/api/downloads");
  const box = document.getElementById("downloads");
  if (!box) return;
  lastDl = Array.isArray(list) ? list : [];
  box.innerHTML = lastDl.map((d) => `<li class="ep${selectMode ? " has-pick" : ""}" data-id="${d.id}">
      ${pickBox(d.id)}
      <span class="cover ph">⤓</span>
      <div class="t"><div class="title">${esc(d.title || d.path)}</div>
        <div class="sub">${[esc(d.feed_title || ""), fmtBytes(d.bytes || 0), d.duration_sec ? fmtTime(d.duration_sec) : "", d.created_at ? new Date(d.created_at * 1000).toLocaleDateString() : ""].filter(Boolean).join(" · ")}</div></div>
      <div class="actions">
        <button data-act="dlplay" data-id="${d.id}" title="${t("dl_play")}">▶</button>
        <button data-act="dldel" data-id="${d.id}" title="${t("dl_del")}">✕</button>
      </div></li>`).join("") || `<p class="mut">${t("dl_empty")}</p>`;
}

/* ---------- settings view ---------- */
let ME = null, ADMIN = false;

async function renderAccount() {
  try { ME = await api("/api/me"); } catch (e) { return; }
  ADMIN = !!ME && ME.role === "admin";
  const n = document.getElementById("acct-name");
  if (n) n.textContent = ME ? ME.username : "—";
  const r = document.getElementById("acct-role");
  if (r) r.textContent = ME ? ME.role : "";
  const card = document.getElementById("card-users");
  if (card) card.hidden = !ADMIN;
  if (ADMIN) renderUsers();
}

async function renderUsers() {
  const list = await api("/api/users");
  const box = document.getElementById("users");
  if (!box) return;
  const rows = (Array.isArray(list) ? list : []).map((u) => `<tr>
      <td>${esc(u.username)}</td>
      <td><span class="badge">${esc(u.role)}</span></td>
      <td>${new Date((u.created_at || 0) * 1000).toLocaleDateString()}</td>
      <td>${ME && u.id === ME.id ? "" : `<button data-act="deluser" data-id="${u.id}">✕</button>`}</td>
    </tr>`).join("");
  box.innerHTML = `<tr><th>username</th><th>${t("role_label") || "role"}</th><th>created</th><th></th></tr>` + rows;
}

async function renderGpodder() {
  const d = await api("/api/gpodder");
  const srv = document.getElementById("gp-server");
  if (srv) srv.textContent = (d && d.sync_url) || location.origin;
  const box = document.getElementById("gp-feeds");
  if (!box) return;
  const feeds = (d && d.feeds) || [];
  box.innerHTML = feeds.map((f) => `<li class="ep" data-id="${f.id}">
      <span class="cover ph">${esc((f.title || "?").slice(0, 1))}</span>
      <div class="t">
        <div class="title">${esc(f.title || "")}</div>
        <div class="sub" style="overflow-wrap:anywhere">${esc(f.url)}</div>
      </div>
      <div class="actions"><button data-act="copy" data-copy="${esc(f.url)}">${t("copy")}</button></div>
    </li>`).join("") || `<p class="mut">—</p>`;
}

// Suggested voice ids; the real list is fetched from the provider's
// /audio/voices endpoint (OpenAI-compatible servers vary a lot).
const VOICE_DEFAULTS = ["alloy", "ash", "ballad", "coral", "echo", "fable", "nova", "onyx", "sage", "shimmer", "verse",
  "zh-CN-XiaoxiaoNeural", "zh-CN-XiaoyiNeural", "zh-CN-YunxiNeural", "zh-CN-YunjianNeural", "zh-CN-YunyangNeural",
  "en-US-AriaNeural", "en-US-GuyNeural", "en-US-JennyNeural"];
let detectedVoices = [];

function fillVoices(list) {
  const dl = document.getElementById("tts-voice-list");
  if (!dl) return;
  dl.innerHTML = (list || []).map((v) => {
    const id = typeof v === "string" ? v : (v.id || v.name || "");
    const label = typeof v === "string" ? v : (v.name || v.id || "");
    return id ? `<option value="${esc(id)}" label="${esc(label)}">` : "";
  }).join("");
}

function renderVoiceChips(list) {
  const box = document.getElementById("tts-voice-list-box");
  if (!box) return;
  box.innerHTML = (list || []).map((v) => {
    const id = typeof v === "string" ? v : (v.id || v.name || "");
    const name = typeof v === "string" ? "" : (v.name || "");
    if (!id) return "";
    return `<button type="button" class="voicechip" data-voice="${esc(id)}">${esc(id)}${name && name !== id ? ` <span class="mut">· ${esc(name)}</span>` : ""}</button>`;
  }).join("");
}

let voicesTried = false;
async function detectVoices(silent) {
  const msg = document.getElementById("tts-voice-msg");
  if (msg && !silent) msg.textContent = "…";
  const r = await api("/api/tts/voices");
  if (!r || r.error) {
    if (r && r.voices) { fillVoices(r.voices); renderVoiceChips(r.voices); }
    if (msg && !silent) msg.textContent = (r && r.error) || t("voices_failed");
    return;
  }
  detectedVoices = r.voices || [];
  fillVoices(detectedVoices);
  renderVoiceChips(detectedVoices);
  if (msg) msg.textContent = (detectedVoices.length ? t("voices_click") + " " : "") + t("voices_found").replace("{n}", detectedVoices.length);
  const input = document.getElementById("tts-voice");
  if (input && !input.value && detectedVoices.length) {
    const first = detectedVoices[0];
    input.value = typeof first === "string" ? first : (first.id || first.name || "");
  }
}

async function renderSettings() {
  const s = await api("/api/settings");
  ADMIN = s.is_admin === "1";
  const note = document.getElementById("admin-note");
  if (note) note.hidden = ADMIN;
  for (const id of ["s-save", "tts-save", "tts-test", "tts-detect", "cache-clear", "cache-evict"]) {
    const b = document.getElementById(id);
    if (b) b.disabled = !ADMIN;
  }
  const e = await api("/api/config");
  document.getElementById("s-fetch").value = s.fetch_interval_min;
  document.getElementById("s-conc").value = s.download_concurrency;
  document.getElementById("s-rule").value = s.conflict_rule;
  document.getElementById("s-auto").checked = s.auto_download_new === "1";
  document.getElementById("s-limit").value = s.storage_limit_mb;
  document.getElementById("s-theme").value = s.ui_theme;
  document.getElementById("tts-provider").value = s.tts_provider || "off";
  document.getElementById("tts-base").value = s.tts_base_url || "";
  document.getElementById("tts-key").value = s.tts_key || "";
  document.getElementById("tts-model").value = s.tts_model || "";
  document.getElementById("tts-voice").value = s.tts_voice || "";
  fillVoices(detectedVoices.length ? detectedVoices : VOICE_DEFAULTS);
  renderVoiceChips(detectedVoices);
  if (!voicesTried && ADMIN && (s.tts_provider || "off") !== "off") {
    voicesTried = true;
    setTimeout(() => detectVoices(true), 300);
  }
  document.getElementById("tts-speed").value = s.tts_speed || "1";
  document.getElementById("tts-max").value = s.tts_max_chars || "3000";
  document.getElementById("tts-auto").checked = (s.tts_auto || "1") === "1";
  document.getElementById("s-mode").value = s.play_mode || "stream";
  document.getElementById("s-cache-dir").value = s.cache_dir || "";
  document.getElementById("s-dl-dir").value = s.download_dir || "";
  document.getElementById("s-cache-limit").value = s.cache_limit_mb || "0";
  document.getElementById("s-dlformat").value = s.download_format || "auto";
  document.getElementById("s-proxy").value = s.http_proxy || "";
  document.getElementById("s-favauto").checked = (s.favorite_auto_download || "0") === "1";
  document.getElementById("device_name").value = DEVICE_NAME;
  const st = await api("/api/stats");
  document.getElementById("stats").innerHTML = `<tr><td>${t("card_env")}</td><td>${st.episodes} ${t("episodes_count") || "eps"}</td></tr>
    <tr><td>${t("chip_downloaded")}</td><td>${st.downloaded}</td></tr>
    <tr><td>${t("downloads_used")}</td><td>${st.downloads || 0} · ${fmtBytes(st.downloads_bytes || 0)} <span class="mut">${esc(st.download_dir || "")}</span></td></tr>
    <tr><td>${t("cache_used")}</td><td>${st.cache_files || 0} files · ${fmtBytes(st.cache_bytes || 0)} <span class="mut">${esc(st.cache_dir || "")}</span></td></tr>
    <tr><td>db</td><td>${fmtBytes(st.db_bytes)}</td></tr>`;
  document.getElementById("envtable").innerHTML = Object.entries(e.env || {}).map(([k, v]) => `<tr><td>${k}</td><td>${v || "—"} ${v ? "" : "(default)"}</td></tr>`).join("");
  const devs = await api("/api/devices");
  document.getElementById("devices").innerHTML = `<tr><th>${t("device_name")}</th><th>last seen</th></tr>` + (devs || []).map((d) => `<tr><td>${esc(d.name || d.id)}</td><td>${new Date(d.last_seen * 1000).toLocaleString()}</td></tr>`).join("");
}

/* ---------- actions (no page reloads) ---------- */
async function renderFeedDetail(id) {
  const box = document.getElementById("detail");
  const f = await api("/api/feeds/" + id);
  if (!f || f.error) { box.innerHTML = `<p class="mut">${t("not_found")}</p>`; return; }
  const r = await api("/api/episodes?feed_id=" + id);
  lastEps = (r && (r.episodes || (Array.isArray(r) ? r : null))) || [];
  box.innerHTML = `
    <a class="back" href="#/feeds">${t("back")}</a>
    <div class="detail-head">
      ${f.cover_url ? `<img class="dcover" src="${esc(f.cover_url)}" alt="" referrerpolicy="no-referrer" onerror="this.removeAttribute('src')">` : `<span class="dcover ph">${esc(f.title || "?")}</span>`}
      <div>
        <h2>${esc(f.title || f.url)}</h2>        <div class="sub">${f.episodes} ${t("episodes_count")} · ${f.downloaded} ${t("status_downloaded")} · <span class="badge kind">${t("kind_" + (f.kind || "auto"))}</span> · <span class="badge">${t("category")}: ${esc(f.category || t("category_none"))}</span>${f.auto_download ? ` · <span class="badge dl">⤓ auto</span>` : ""}</div>
        <div class="actions"><button data-act="sel" data-id="${f.id}">${t("select")}</button><button data-act="editfeed" data-id="${f.id}">${t("edit_feed")}</button><button data-act="refetch" data-id="${f.id}">${t("refetch")}</button><button data-act="pack" data-id="${f.id}">${t("pack")}</button><button data-act="del" data-id="${f.id}">${t("del")}</button></div>
      </div>
    </div>
    ${f.description ? `<div class="article">${esc(f.description)}</div>` : ""}
    <section class="card" id="feed-edit" hidden>
      <h3>${t("edit_feed")}</h3>
      <div class="kv"><label>${t("feed_url")}</label><input id="fe-url" value="${esc(f.url)}"></div>
      <div class="kv"><label>${t("category")}</label><input id="fe-cat" value="${esc(f.category || "")}" placeholder="${esc(t("category_none"))}"></div>
      <div class="kv"><label>${t("kind")}</label><select id="fe-kind"><option value="auto" ${f.setting === "auto" ? "selected" : ""}>${t("kind_auto")}</option><option value="audio" ${f.setting === "audio" ? "selected" : ""}>${t("kind_audio")}</option><option value="video" ${f.setting === "video" ? "selected" : ""}>${t("kind_video")}</option><option value="text" ${f.setting === "text" ? "selected" : ""}>${t("kind_text")}</option></select></div>
      <div class="kv"><label>${t("feed_auto_download")}</label><input id="fe-auto" type="checkbox" ${f.auto_download ? "checked" : ""}></div>
      <div class="kv"><label>${t("feed_fetch_fulltext")}</label><input id="fe-fulltext" type="checkbox" ${f.fetch_fulltext ? "checked" : ""}></div>
      <button data-act="savefeed" data-id="${f.id}">${t("save_feed")}</button>
      <div class="mut" id="fe-msg">${t("edit_note")}</div>
    </section>
    <ul class="list" id="detail-list">${lastEps.slice(0, 200).map(rowHtml).join("") || `<p class="mut">—</p>`}</ul>`;
}

async function renderEpisodeDetail(id) {
  const box = document.getElementById("detail");
  const ep = await api("/api/episodes/" + id + "?device_id=" + encodeURIComponent(DEVICE));
  if (!ep || ep.error) { box.innerHTML = `<p class="mut">${t("not_found")}</p>`; return; }
  lastEps = [ep];
  if (segEp(ep)) { const r = await segmentInfo(ep.id); if (r) segTotal = r.segments || segTotal; renderBar(); }
  box.innerHTML = `
    <a class="back" href="#/feed/${esc(String(ep.feed_id))}">${t("back")}</a>
    <div class="detail-head">
      ${ep.cover_url ? `<img class="dcover" src="${esc(ep.cover_url)}" alt="" referrerpolicy="no-referrer" onerror="this.removeAttribute('src')">` : `<span class="dcover ph">${esc((ep.feed_title || "?").slice(0, 2))}</span>`}
      <div>
        <h2>${esc(ep.title)}</h2>
        <div class="sub">${[esc(ep.feed_title || ""), fmtDur(ep.duration_sec), `<span class="badge kind">${t("kind_" + (ep.kind || "audio"))}</span>`, (ep.kind === "text" || ep.segments > 0) && (ep.segments || ep.chunks) ? `${ep.segments || ep.chunks} ${t("segments")}` : ""].filter(Boolean).join(" · ")}</div>
      </div>
    </div>
    <div class="actions">
      <button data-act="play" data-id="${ep.id}">${t("play")}</button>
      <button data-act="q" data-id="${ep.id}">${t("queue")}</button>
      <button data-act="qnext" data-id="${ep.id}">${t("qnext")}</button>
      <button data-act="done" data-id="${ep.id}">${t("done")}</button>
      ${ep.kind === "text" && ep.status !== "downloaded" ? `<button data-act="tts" data-id="${ep.id}">${t("tts_render")}</button>` : ""}
      ${(ep.link || /^https?:/i.test(ep.guid || "")) ? `<button data-act="fulltext" data-id="${ep.id}">${t("get_fulltext")}</button>` : ""}
    </div>
    <div class="article">${esc(strip(ep.body) || strip(ep.description) || "")}</div>
    ${(ep.images || []).length ? `<h3 class="mut">${t("images")}</h3><div class="imgrid">${ep.images.map((u) => `<a href="${esc(u)}" target="_blank" rel="noreferrer"><img src="${esc(u)}" loading="lazy" referrerpolicy="no-referrer" alt="" onerror="this.parentElement.remove()"></a>`).join("")}</div>` : ""}`;
}

const byId = (id) => lastEps.find((x) => x.id === id);

document.getElementById("dl-retag").addEventListener("click", async () => {
  const r = await api("/api/downloads/retag", { method: "POST" });
  const el = document.getElementById("dl-msg");
  if (el) el.textContent = r.error || ("queued " + (r.queued || 0));
  setTimeout(renderDownloads, 6000);
});
for (const id of ["sel-updates", "sel-feeds", "sel-playlist", "sel-fav", "sel-dl"]) {
  const b = document.getElementById(id);
  if (b) b.addEventListener("click", enterSelect);
}
document.getElementById("pw-save").addEventListener("click", async () => {
  const msg = document.getElementById("pw-msg");
  const r = await api("/api/me/password", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({
    old: document.getElementById("pw-old").value,
    new: document.getElementById("pw-new").value,
  }) });
  if (msg) msg.textContent = r.error || t("password_changed");
  if (!r.error) { document.getElementById("pw-old").value = ""; document.getElementById("pw-new").value = ""; }
});
document.getElementById("logout").addEventListener("click", async () => {
  try { await fetch("/api/logout", { method: "POST" }); } catch (e) {}
  location.replace("/login");
});
document.getElementById("u-add").addEventListener("click", async () => {
  const msg = document.getElementById("u-msg");
  const r = await api("/api/users", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({
    username: document.getElementById("u-new-name").value,
    password: document.getElementById("u-new-pass").value,
    role: document.getElementById("u-new-role").value,
  }) });
  if (msg) msg.textContent = r.error || "created";
  document.getElementById("u-new-name").value = "";
  document.getElementById("u-new-pass").value = "";
  renderUsers();
});
document.getElementById("pl-playall").addEventListener("click", () => { if (queue.length) { setBar(queue[0]); play(); } });
document.getElementById("pl-shuffle").addEventListener("click", async () => {
  await api("/api/queue/shuffle?device_id=" + encodeURIComponent(DEVICE), { method: "POST" });
  renderPlaylist();
});
document.getElementById("pl-clear").addEventListener("click", async () => {
  await api("/api/queue/clear?device_id=" + encodeURIComponent(DEVICE), { method: "POST" });
  renderPlaylist();
});
document.getElementById("cache-clear").addEventListener("click", async () => {
  await api("/api/cache", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ action: "clear" }) });
  renderSettings();
});
document.getElementById("cache-evict").addEventListener("click", async () => {
  await api("/api/cache", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ action: "evict" }) });
  renderSettings();
});
document.getElementById("qclear").addEventListener("click", async () => { await api("/api/queue/clear?device_id=" + encodeURIComponent(DEVICE), { method: "POST" }); await loadQueue(); });
document.addEventListener("click", async (ev) => {
  const b = ev.target.closest("[data-act]");
  if (!b) return;
  const id = Number(b.dataset.id), act = b.dataset.act;
  if (act === "notes") { const n = document.querySelector(`[data-notes="${id}"]`); if (n) n.hidden = !n.hidden; return; }
  if (act === "openfeed") { show("feed", id); return; }
  if (act === "openepisode") { show("episode", String(id)); return; }
  const ep = byId(id);
  if (act === "sel") { enterSelect(); return; }
  if (act === "pick") { if (picked.has(id)) picked.delete(id); else picked.add(id); renderSelBar(); return; }
  if (act === "selall") {
    const v = route().view;
    let ids = [];
    if (v === "feeds") ids = lastFeeds.map((f) => f.id);
    else if (v === "playlist") ids = queue.map((q) => q.queue_id);
    else if (v === "downloads") ids = lastDl.map((d) => d.id);
    else ids = lastEps.map((e) => e.id);
    picked = new Set(ids);
    renderSelBar();
    refreshView();
    return;
  }
  if (act === "selnone") { picked.clear(); renderSelBar(); refreshView(); return; }
  if (act === "selexit") { exitSelect(); return; }
  if (act === "batch") {
    const op = b.dataset.op;
    const ids = [...picked];
    if (!ids.length) { toast(t("select_none")); return; }
    if ((op === "delfeed" || op === "deldl" || op === "delqueue") && !confirm(t("confirm_delete_n").replace("{n}", ids.length))) return;
    const r = await api("/api/batch", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ op, ids, device_id: DEVICE }) });
    if (r.error) { toast(r.error); return; }
    toast(t("done") + " " + (r.count || 0));
    selectMode = false;
    picked.clear();
    renderSelBar();
    refreshView();
    return;
  }
  if (act === "copy") {
    const text = b.dataset.copy || "";
    try {
      await navigator.clipboard.writeText(text);
      toast(t("copied"));
    } catch (e) {
      window.prompt(t("copy"), text);
    }
    return;
  }
  if (act === "deluser") { await api("/api/users/" + id, { method: "DELETE" }); renderUsers(); return; }
  if (act === "editfeed") { const panel = document.getElementById("feed-edit"); if (panel) panel.hidden = !panel.hidden; return; }
  if (act === "savefeed") {
    const body = {
      url: document.getElementById("fe-url").value,
      category: document.getElementById("fe-cat").value,
      kind: document.getElementById("fe-kind").value,
      auto_download: document.getElementById("fe-auto").checked,
      fetch_fulltext: document.getElementById("fe-fulltext").checked,
    };
    const r = await api("/api/feeds/" + id, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
    const msg = document.getElementById("fe-msg");
    if (msg) msg.textContent = r.error || ("saved " + (r.note || ""));
    renderFeedDetail(id);
    return;
  }
  if (act === "fav") {
    const on = !!(ep && ep.favorite);
    if (on) await api("/api/favorites/" + id, { method: "DELETE" });
    else { const r = await api("/api/favorites", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ episode_id: id }) }); if (r && r.error) flash(r.error); }
    if (ep) ep.favorite = !on;
    refreshView();
    toast(on ? t("unfav") : t("fav"));
    return;
  }
  if (act === "qnext") { await api("/api/queue", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ episode_id: id, device_id: DEVICE, next: true }) }); await loadQueue(); toast(t("qnext") + " ✓"); return; }
  if (act === "plplay") { const qe = queue.find((x) => x.queue_id === id); if (qe) { setBar(qe); play(); } return; }
  if (act === "plup" || act === "pldown") { await api("/api/queue/" + id + "/move", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ device_id: DEVICE, dir: act === "plup" ? "up" : "down" }) }); renderPlaylist(); return; }
  if (act === "pldel") { await api("/api/queue/" + id + "?device_id=" + encodeURIComponent(DEVICE), { method: "DELETE" }); renderPlaylist(); return; }
  if (act === "dlplay") {
    const list = await api("/api/downloads");
    const d = (Array.isArray(list) ? list : []).find((x) => x.id === id);
    if (d) {
      setBar({ id: d.episode_id || 0, title: d.title, feed_title: d.feed_title, cover_url: "", kind: "audio", dl_file: "/api/downloads/" + d.id + "/file", duration_sec: d.duration_sec, position: 0 });
      play();
    }
    return;
  }
  if (act === "dldel") { await api("/api/downloads/" + id, { method: "DELETE" }); refreshView(); return; }
  if (act === "play") { if (ep) { setBar(ep); play(); } return; }
  if (act === "fulltext") {
    flash("…");
    const r = await api("/api/episodes/" + id + "/fulltext", { method: "POST" });
    if (r && r.error) { toast(r.error); return; }
    toast(t("fulltext_done") + " ✓");
    await renderEpisodeDetail(id);
    return;
  }
  if (act === "q") { await api("/api/queue", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ episode_id: id, device_id: DEVICE }) }); await loadQueue(); toast(t("queue") + " ✓"); return; }
  if (act === "qplay") { const qe = queue.find((x) => x.id === id); if (qe) { setBar(qe); play(); } return; }
  if (act === "qdel") { await api("/api/queue/" + id + "?device_id=" + encodeURIComponent(DEVICE), { method: "DELETE" }); await loadQueue(); return; }
  if (act === "dl") { const r = await api("/api/download", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ episode_id: id }) }); if (r && r.error) { toast(r.error); return; } if (ep) ep.status = "queued"; refreshView(); toast(t("dl") + " ✓"); setTimeout(refreshView, 5000); return; }
  if (act === "tts") {
    const r = await api("/api/tts", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ episode_id: id }) });
    if (r.error) { flash(r.error); toast(r.error); } else { flash(t("tts_render") + " ✓"); toast(t("tts_render") + " ✓"); }
    refreshView();
    return;
  }
  if (act === "done") { await api("/api/progress", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ episode_id: id, device_id: DEVICE, position_sec: 0, completed: 1 }) }); if (ep) ep.completed = true; refreshView(); toast(t("done") + " ✓"); return; }
  if (act === "del") {
    const r = await api("/api/feeds/" + id, { method: "DELETE" });
    if (r && r.kept_downloads) flash(t("deleted_kept") + " (" + r.kept_downloads + ")");
    show("feeds", "");
    return;
  }
  if (act === "refetch") { const r = await api("/api/refetch", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ feed_id: id }) }); toast(r.error || (t("refetch") + " ✓ new " + (r.new_items || 0))); refreshView(); return; }
  if (act === "pack") {
    const r = await fetch("/api/pack/" + id);
    const ct = r.headers.get("Content-Type") || "";
    if (!ct.includes("zip")) { flash((await r.json().catch(() => ({}))).error || "pack failed"); return; }
    const blob = await r.blob();
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = "book-" + id + ".zip";
    a.click();
    return;
  }
});
document.getElementById("chips").addEventListener("click", (ev) => {
  const c = ev.target.closest(".chip"); if (!c) return;
  filterStatus = c.dataset.filter;
  document.querySelectorAll(".chip").forEach((x) => x.classList.toggle("active", x === c));
  renderEpisodes();
});
document.getElementById("filter-feed").addEventListener("change", (ev) => { filterFeed = ev.target.value; renderUpdates(); });
document.getElementById("addfeed").addEventListener("submit", async (ev) => {
  ev.preventDefault();
  const r = await api("/api/feeds", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ url: document.getElementById("feedurl").value, kind: document.getElementById("feedkind").value, category: (document.getElementById("feedcat") || {}).value || "" }) });
  flash(r.error || "");
  renderFeeds();
});
const opmlFileInput = document.getElementById("opml-file");
if (opmlFileInput) opmlFileInput.addEventListener("change", async () => {
  const f = opmlFileInput.files && opmlFileInput.files[0];
  if (!f) return;
  document.getElementById("opml-in").value = await f.text();
  document.getElementById("opml-msg").textContent = f.name;
});
document.getElementById("opml-import").addEventListener("click", async () => {
  const msg = document.getElementById("opml-msg");
  const body = document.getElementById("opml-in").value.trim();
  if (!body) { msg.textContent = t("opml_empty"); return; }
  const r = await fetch("/api/opml", { method: "POST", body });
  const j = await r.json();
  document.getElementById("opml-msg").textContent = j.error ? j.error :
    (t("import_done").replace("{n}", j.added || 0) + " · " +
      t("kind_video") + " " + (j.video || 0) + " / " + t("kind_audio") + " " + (j.audio || 0) + " / " + t("kind_text") + " " + (j.text || 0) +
      (j.unknown ? " / ? " + j.unknown : ""));
  renderFeeds();
});
document.getElementById("s-save").addEventListener("click", async () => {
  await api("/api/settings", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({
    fetch_interval_min: document.getElementById("s-fetch").value,
    download_concurrency: document.getElementById("s-conc").value,
    conflict_rule: document.getElementById("s-rule").value,
    auto_download_new: document.getElementById("s-auto").checked ? "1" : "0",
    storage_limit_mb: document.getElementById("s-limit").value,
    ui_theme: document.getElementById("s-theme").value,
    play_mode: document.getElementById("s-mode").value,
    http_proxy: document.getElementById("s-proxy").value,
    cache_dir: document.getElementById("s-cache-dir").value,
    download_dir: document.getElementById("s-dl-dir").value,
    cache_limit_mb: document.getElementById("s-cache-limit").value,
    download_format: document.getElementById("s-dlformat").value,
    favorite_auto_download: document.getElementById("s-favauto").checked ? "1" : "0",
  }) });
  document.body.dataset.theme = document.getElementById("s-theme").value;
  renderSettings();
});
document.getElementById("tts-save").addEventListener("click", async () => {
  await api("/api/settings", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({
    tts_provider: document.getElementById("tts-provider").value,
    tts_base_url: document.getElementById("tts-base").value,
    tts_key: document.getElementById("tts-key").value,
    tts_model: document.getElementById("tts-model").value,
    tts_voice: document.getElementById("tts-voice").value,
    tts_speed: document.getElementById("tts-speed").value,
    tts_max_chars: document.getElementById("tts-max").value,
    tts_auto: document.getElementById("tts-auto").checked ? "1" : "0",
  }) });
  flash("saved");
  renderSettings();
});
document.getElementById("tts-test").addEventListener("click", async () => {
  flash("…");
  const r = await api("/api/tts/test", { method: "POST" });
  flash(r.error || (t("tts_test") + " ✓ " + r.bytes + " bytes"));
});
document.getElementById("tts-detect").addEventListener("click", () => detectVoices(false));
document.getElementById("tts-voice-list-box").addEventListener("click", (e) => {
  const b = e.target.closest("[data-voice]");
  if (!b) return;
  document.getElementById("tts-voice").value = b.dataset.voice;
  document.querySelectorAll("#tts-voice-list-box .voicechip").forEach((x) => x.classList.toggle("active", x === b));
});
document.getElementById("syncnow").addEventListener("click", async () => {
  localStorage.setItem("device_name", document.getElementById("device_name").value);
  await api("/api/sync?device_id=" + encodeURIComponent(document.getElementById("device_name").value || DEVICE));
  renderSettings();
});
document.querySelectorAll("[data-lang]").forEach((b) => b.addEventListener("click", () => { lang = b.dataset.lang; localStorage.setItem("lang", lang); applyLang(); }));

/* boot */
applyLang();
renderAccount();
const savedEp = JSON.parse(localStorage.getItem("last_episode") || "null");
if (savedEp) {
  current = { id: savedEp.id, title: savedEp.title, position: savedEp.position, duration_sec: 0, cover_url: savedEp.cover || "", file: null, enclosure_url: savedEp.url, dl_file: savedEp.dl || "" };
  setBar(current);
  current._posLoaded = false;
  api("/api/episodes/" + savedEp.id + "?device_id=" + encodeURIComponent(DEVICE)).then((ep) => {
    if (!ep || ep.error || !current || current.id !== ep.id) return;
    current.position = ep.position || current.position || 0;
    current.duration_sec = ep.duration_sec || current.duration_sec || 0;
    current.file = ep.file || null;
    if (ep.cover_url) current.cover_url = ep.cover_url;
    current._posLoaded = true;
    renderBar();
    saveLastEpisode();
  }).catch(() => {});
}
document.body.dataset.theme = "dark";
api("/api/settings").then((s) => { document.body.dataset.theme = s.ui_theme || "dark"; });
const segs = (location.hash || "#/updates").slice(2).split("/");
const seg = segs;
show(seg[0] || seg[0], seg[1]);
