const api = (path, opts = {}) => {
  const headers = Object.assign({ "Content-Type": "application/json" }, opts.headers || {});
  const csrf = getCookie("csrf_token");
  if (csrf && opts.method && opts.method !== "GET") headers["X-CSRF-Token"] = csrf;
  return fetch(path, Object.assign({ credentials: "include" }, opts, { headers })).then(async (r) => {
    const text = await r.text();
    let data = null;
    try { data = text ? JSON.parse(text) : null; } catch { data = text; }
    if (!r.ok) throw Object.assign(new Error((data && data.error && data.error.message) || r.statusText), { status: r.status, data });
    return data;
  });
};
function getCookie(n) {
  const m = document.cookie.match(new RegExp("(?:^|; )" + n + "=([^;]*)"));
  return m ? decodeURIComponent(m[1]) : "";
}
const $ = (id) => document.getElementById(id);
let accounts = [], streams = [], categories = [];
async function refreshAuth() {
  try {
    const me = await api("/api/v1/auth/me");
    $("auth-status").textContent = me.email;
    $("login-panel").hidden = true;
    $("app").hidden = false;
    await loadCatalog();
    await loadTxns();
  } catch {
    $("login-panel").hidden = false;
    $("app").hidden = true;
  }
}
async function loadCatalog() {
  accounts = (await api("/api/v1/accounts")).items || [];
  streams = (await api("/api/v1/income-streams")).items || [];
  categories = (await api("/api/v1/categories")).items || [];
  $("account").innerHTML = accounts.map((a) => `<option value="${a.ID || a.id}">${a.Name || a.name}</option>`).join("");
  $("income_stream").innerHTML = streams.map((s) => `<option value="${s.ID || s.id}">${s.Name || s.name}</option>`).join("");
}
$("direction").onchange = () => { $("income_stream").hidden = $("direction").value !== "income"; };
$("btn-login").onclick = async () => {
  try {
    await api("/api/v1/auth/login", { method: "POST", body: JSON.stringify({ email: $("email").value, password: $("password").value }) });
    await refreshAuth();
  } catch (e) { $("login-err").textContent = e.message; }
};
$("btn-logout").onclick = async () => { try { await api("/api/v1/auth/logout", { method: "POST", body: "{}" }); } catch {} refreshAuth(); };
document.querySelectorAll("nav button[data-tab]").forEach((b) => b.onclick = () => {
  ["txns", "review", "reports"].forEach((t) => { $("tab-" + t).hidden = t !== b.dataset.tab; });
  if (b.dataset.tab === "review") loadReview();
});
async function loadTxns() {
  const items = (await api("/api/v1/transactions")).items || [];
  $("txn-body").innerHTML = items.map((t) => `<tr><td>${t.txn_date}</td><td>${t.direction}</td><td>${t.amount}</td><td>${t.payee_raw || ""}</td>
    <td><button data-void="${t.id}">void</button></td></tr>`).join("");
  $("txn-body").querySelectorAll("[data-void]").forEach((btn) => btn.onclick = async () => {
    await api("/api/v1/transactions/" + btn.dataset.void + "/void", { method: "POST", body: "{}" });
    loadTxns();
  });
}
$("btn-create").onclick = async () => {
  const dir = $("direction").value;
  const body = { account_id: $("account").value, direction: dir, amount: $("amount").value, currency: "INR",
    txn_date: $("txn_date").value || new Date().toISOString().slice(0, 10), payee_raw: $("payee").value };
  if (dir === "income") body.income_stream_id = $("income_stream").value;
  await api("/api/v1/transactions", { method: "POST", body: JSON.stringify(body), headers: { "Idempotency-Key": crypto.randomUUID() } });
  loadTxns();
};
async function loadReview() {
  const items = (await api("/api/v1/review-queue")).items || [];
  if (!items.length) { $("review-list").textContent = "Queue empty"; return; }
  const expenseCats = categories.filter((c) => (c.Kind || c.kind) === "expense");
  $("review-list").innerHTML = items.map((it) => {
    const opts = expenseCats.map((c) => `<option value="${c.ID || c.id}">${c.Name || c.name}</option>`).join("");
    return `<div><div>${it.payee_raw || ""} · ${it.reason}</div>
      <select data-cat-for="${it.transaction_id}">${opts}</select>
      <button data-mod="${it.transaction_id}">Moderate</button></div>`;
  }).join("");
  $("review-list").querySelectorAll("[data-mod]").forEach((btn) => btn.onclick = async () => {
    const sel = document.querySelector(`[data-cat-for="${btn.dataset.mod}"]`);
    await api("/api/v1/transactions/" + btn.dataset.mod + "/moderation", { method: "POST", body: JSON.stringify({ category_id: sel.value }) });
    loadReview();
  });
}
$("btn-load-report").onclick = async () => {
  const m = $("rep-month").value;
  const q = m ? ("?month=" + m) : "";
  $("summary").textContent = JSON.stringify(await api("/api/v1/reports/summary" + q), null, 2);
  const pts = (await api("/api/v1/reports/timeseries" + q)).points || [];
  const max = Math.max(1, ...pts.map((p) => Math.max(parseFloat(p.income), parseFloat(p.expense))));
  $("chart").innerHTML = pts.map((p) => `<div style="flex:1;display:flex;flex-direction:column;justify-content:flex-end;height:100%">
    <div class="bar" style="height:${(parseFloat(p.income)/max)*100}%"></div>
    <div class="bar exp" style="height:${(parseFloat(p.expense)/max)*100}%"></div></div>`).join("");
  $("csv-link").href = "/api/v1/exports/transactions.csv" + q;
};
$("txn_date").value = new Date().toISOString().slice(0, 10);
$("rep-month").value = new Date().toISOString().slice(0, 7);
refreshAuth();
