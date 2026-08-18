const $ = (id) => document.getElementById(id);

async function hmacHex(secret, message) {
  const enc = new TextEncoder();
  const key = await crypto.subtle.importKey(
    "raw",
    enc.encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"]
  );
  const buf = await crypto.subtle.sign("HMAC", key, enc.encode(message));
  return [...new Uint8Array(buf)].map((b) => b.toString(16).padStart(2, "0")).join("");
}

async function sha256Hex(text) {
  const buf = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(text));
  return [...new Uint8Array(buf)].map((b) => b.toString(16).padStart(2, "0")).join("");
}

function nonce() {
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  return [...bytes].map((b) => b.toString(16).padStart(2, "0")).join("");
}

function idemKey() {
  return "idemp-" + nonce();
}

async function api(path, opts) {
  const res = await fetch(path, opts);
  const text = await res.text();
  let data = null;
  try { data = text ? JSON.parse(text) : null; } catch { data = { raw: text }; }
  if (!res.ok) {
    throw new Error((data && data.error) || res.statusText);
  }
  return data;
}

async function refreshMeta() {
  try {
    const m = await api("/api/v1/meta");
    $("status").className = "status";
    $("status").textContent = `ok · queue ${m.queue_depth} · dlq ${m.dlq} · ${m.go}`;
  } catch (err) {
    $("status").className = "status bad";
    $("status").textContent = String(err);
  }
}

async function refreshRadars() {
  const data = await api("/api/v1/radars");
  $("radars").innerHTML = (data.radars || []).map((d) => `
    <div class="row">
      <div>
        <strong>${escapeHtml(d.name)}</strong>
        <div class="muted">${escapeHtml(d.id)} · ${escapeHtml(d.url)}</div>
      </div>
      <button data-toggle="${d.id}" data-enabled="${d.enabled}">${d.enabled ? "Disable" : "Enable"}</button>
    </div>
  `).join("");
}

async function refreshJournal() {
  const data = await api("/api/v1/journal");
  $("journal").innerHTML = (data.entries || []).map((e) => `
    <div class="row">
      <div>
        <strong>${escapeHtml(e.kind)}</strong> ${e.status || ""} ${escapeHtml(e.report_kind || "")}
        <div class="muted">${escapeHtml(e.forward_id)} · attempt ${e.attempt} · ${escapeHtml(e.note || e.error || "")}</div>
      </div>
    </div>
  `).join("") || '<div class="muted">empty</div>';
}

async function refreshDlq() {
  const data = await api("/api/v1/dlq");
  $("dlq").innerHTML = (data.items || []).map((it) => `
    <div class="row">
      <div>
        <strong>${escapeHtml(it.reason)}</strong>
        <div class="muted">${escapeHtml(it.forward_id)}</div>
      </div>
      <button data-replay="${it.forward_id}">Replay</button>
    </div>
  `).join("") || '<div class="muted">empty</div>';
}

async function refreshSink() {
  const data = await api("/api/v1/sink/recent");
  $("sink").innerHTML = (data.received || []).map((r) => `
    <div class="row">
      <div>
        <strong>${escapeHtml(r.forward_id || "")}</strong>
        <div class="muted">${escapeHtml(JSON.stringify(r.body))}</div>
      </div>
    </div>
  `).join("") || '<div class="muted">empty</div>';
}

async function refreshAll() {
  await refreshMeta();
  await Promise.all([refreshRadars(), refreshJournal(), refreshDlq(), refreshSink()]);
}

function escapeHtml(s) {
  return String(s ?? "").replace(/[&<>"']/g, (c) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;"
  }[c]));
}

$("radar-form").addEventListener("submit", async (ev) => {
  ev.preventDefault();
  const fd = new FormData(ev.target);
  const prefixes = String(fd.get("kind_prefixes") || "")
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
  await api("/api/v1/radars", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      name: fd.get("name"),
      url: fd.get("url"),
      secret: fd.get("secret"),
      kind_prefixes: prefixes,
      ordered: false,
      rate: 5,
      burst: 5
    })
  });
  ev.target.reset();
  await refreshRadars();
});

$("report-form").addEventListener("submit", async (ev) => {
  ev.preventDefault();
  const fd = new FormData(ev.target);
  const payloadText = String(fd.get("payload") || "{}");
  let payload;
  try { payload = JSON.parse(payloadText); }
  catch (err) { $("report-result").textContent = "payload json: " + err; return; }
  const bodyObj = {
    kind: fd.get("kind"),
    icao: payload.icao || "ABC123",
    lat: payload.lat ?? 51.47,
    lon: payload.lon ?? -0.4543,
    squawk: payload.squawk || "7000",
    alt_ft: payload.alt_ft || 0,
    payload
  };
  const body = JSON.stringify(bodyObj);
  const ts = Math.floor(Date.now() / 1000);
  const n = nonce();
  const canonical = `ads1.${ts}.${n}.${await sha256Hex(body)}`;
  const sig = "ads1=" + await hmacHex("dev-station-secret", canonical);
  try {
    const res = await api("/api/v1/reports", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Adsb-Timestamp": String(ts),
        "X-Adsb-Nonce": n,
        "X-Adsb-Signature": sig,
        "Idempotency-Key": idemKey(),
        "X-Adsb-Station-Key": "station"
      },
      body
    });
    $("report-result").textContent = JSON.stringify(res, null, 2);
    setTimeout(refreshAll, 400);
  } catch (err) {
    $("report-result").textContent = String(err);
  }
});

$("refresh").addEventListener("click", refreshAll);

document.body.addEventListener("click", async (ev) => {
  const t = ev.target;
  if (!(t instanceof HTMLElement)) return;
  if (t.dataset.toggle) {
    const enabled = t.dataset.enabled !== "true";
    await api(`/api/v1/radars/${t.dataset.toggle}/enable`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enabled })
    });
    await refreshRadars();
  }
  if (t.dataset.replay) {
    await api(`/api/v1/replay/${t.dataset.replay}`, { method: "POST" });
    await refreshAll();
  }
});

refreshAll();
setInterval(refreshMeta, 3000);
