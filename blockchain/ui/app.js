const el = (id) => document.getElementById(id);

let pollTimer = null;
let mempoolAuto = false;

async function getJSON(path) {
  const resp = await fetch(path, { headers: { Accept: "application/json" } });
  const txt = await resp.text();
  let data = null;
  try {
    data = txt ? JSON.parse(txt) : null;
  } catch {
    // ignore
  }
  return { ok: resp.ok, status: resp.status, data, text: txt, headers: resp.headers };
}

function setText(id, value) {
  el(id).textContent = value == null ? "-" : String(value);
}

function fmtHeader(h) {
  return `height=${h.height}  diff=${h.difficultyBits}  txs=${h.txCount}\n${h.hashHex}`;
}

function looksLikeHex(s) {
  return /^[0-9a-fA-F]+$/.test(s) && s.length >= 16;
}

async function refreshChain() {
  const { ok, data, status, text } = await getJSON("/chain/info");
  if (!ok) {
    setText("height", `error (${status})`);
    setText("latestHash", text || "-");
    return;
  }
  setText("height", data.height);
  setText("latestHash", data.latestHash);
  setText("difficultyBits", data.difficultyBits);
  setText("nextDifficultyBits", data.nextDifficultyBits);
  setText("peersCount", data.peersCount ?? data.peers_count ?? "-");

  const height = Number(data.height || 0);
  const from = height > 20 ? height - 20 : 0;
  const headersRes = await getJSON(`/headers/from/${from}?count=50`);
  const headersEl = el("headers");
  headersEl.innerHTML = "";
  if (!headersRes.ok || !Array.isArray(headersRes.data)) {
    headersEl.innerHTML = `<div class="err">failed to load headers</div>`;
    return;
  }
  const headers = headersRes.data.slice(-20).reverse();
  for (const h of headers) {
    const item = document.createElement("div");
    item.className = "item";
    item.innerHTML = `
      <div class="top">
        <div><a href="#" data-height="${h.height}">Block #${h.height}</a></div>
        <div class="badge">${h.hashHex.slice(0, 16)}…</div>
      </div>
      <div class="badge">${fmtHeader(h).replaceAll("\n", "  ")}</div>
    `;
    item.querySelector("a").addEventListener("click", async (e) => {
      e.preventDefault();
      await loadBlockByHeight(h.height);
    });
    headersEl.appendChild(item);
  }
}

async function loadBlockByHeight(height) {
  el("detailsStatus").textContent = "";
  el("details").textContent = "loading…";
  const res = await getJSON(`/block/height/${height}`);
  if (!res.ok) {
    el("details").textContent = res.text || "";
    el("detailsStatus").textContent = `error (${res.status})`;
    el("detailsStatus").className = "err";
    return;
  }
  el("details").textContent = JSON.stringify(res.data, null, 2);
  el("detailsStatus").textContent = `block #${height}`;
  el("detailsStatus").className = "ok";
}

async function loadBlockByHash(hashHex) {
  el("detailsStatus").textContent = "";
  el("details").textContent = "loading…";
  const res = await getJSON(`/blocks/hash/${hashHex}`);
  if (!res.ok) {
    el("details").textContent = res.text || "";
    el("detailsStatus").textContent = `error (${res.status})`;
    el("detailsStatus").className = "err";
    return;
  }
  el("details").textContent = JSON.stringify(res.data, null, 2);
  el("detailsStatus").textContent = `block ${hashHex.slice(0, 16)}…`;
  el("detailsStatus").className = "ok";
}

async function loadTx(txid) {
  el("detailsStatus").textContent = "";
  el("details").textContent = "loading…";
  const res = await getJSON(`/tx/${txid}`);
  if (!res.ok) {
    el("details").textContent = res.text || "";
    el("detailsStatus").textContent = `error (${res.status})`;
    el("detailsStatus").className = "err";
    return;
  }
  el("details").textContent = JSON.stringify(res.data, null, 2);
  el("detailsStatus").textContent = `tx ${txid.slice(0, 16)}…`;
  el("detailsStatus").className = "ok";
}

async function loadMempool() {
  el("mempoolStatus").textContent = "loading…";
  el("mempoolOut").textContent = "";
  const res = await getJSON("/mempool");
  if (!res.ok) {
    el("mempoolStatus").textContent = `error (${res.status})`;
    el("mempoolStatus").className = "err";
    el("mempoolOut").textContent = res.text || "";
    mempoolAuto = false;
    return;
  }
  el("mempoolStatus").textContent = `ok (size=${res.data?.size ?? "?"})`;
  el("mempoolStatus").className = "ok";
  el("mempoolOut").textContent = JSON.stringify(res.data, null, 2);
  mempoolAuto = true;
}

async function doSearch() {
  const q = (el("q").value || "").trim();
  if (!q) return;
  el("searchStatus").textContent = "";
  if (/^\d+$/.test(q)) {
    await loadBlockByHeight(Number(q));
    return;
  }
  if (looksLikeHex(q)) {
    // Try tx first, then block-by-hash.
    const txRes = await getJSON(`/tx/${q}`);
    if (txRes.ok) {
      el("details").textContent = JSON.stringify(txRes.data, null, 2);
      el("detailsStatus").textContent = `tx ${q.slice(0, 16)}…`;
      el("detailsStatus").className = "ok";
      return;
    }
    await loadBlockByHash(q);
    return;
  }
  el("searchStatus").textContent = "unrecognized input";
}

function setWSStatus(text, ok) {
  const s = el("wsStatus");
  if (!s) return;
  s.textContent = text ? `• ${text}` : "";
  s.className = ok ? "ok" : "err";
}

function setupWebSocket() {
  const proto = location.protocol === "https:" ? "wss" : "ws";
  const url = `${proto}://${location.host}/ws`;
  let ws = null;
  let reconnectMs = 500;

  const connect = () => {
    try {
      ws = new WebSocket(url);
    } catch {
      ws = null;
    }
    if (!ws) {
      setWSStatus("WS unavailable (polling)", false);
      if (!pollTimer) pollTimer = setInterval(refreshChain, 5000);
      return;
    }

    ws.onopen = () => {
      setWSStatus("WS connected", true);
      reconnectMs = 500;
      if (pollTimer) {
        clearInterval(pollTimer);
        pollTimer = null;
      }
    };

    ws.onclose = () => {
      setWSStatus("WS disconnected (reconnecting)", false);
      if (!pollTimer) pollTimer = setInterval(refreshChain, 5000);
      setTimeout(connect, reconnectMs);
      reconnectMs = Math.min(reconnectMs * 2, 8000);
    };

    ws.onerror = () => {
      // close triggers reconnect
      try {
        ws.close();
      } catch {
        // ignore
      }
    };

    ws.onmessage = async (ev) => {
      let msg = null;
      try {
        msg = JSON.parse(ev.data);
      } catch {
        return;
      }
      if (!msg || !msg.type) return;

      if (msg.type === "new_block" || msg.type === "reorg") {
        await refreshChain();
      }
      if ((msg.type === "mempool_added" || msg.type === "mempool_removed") && mempoolAuto) {
        await loadMempool();
      }
    };
  };

  connect();
}

document.addEventListener("DOMContentLoaded", async () => {
  el("go").addEventListener("click", doSearch);
  el("q").addEventListener("keydown", (e) => {
    if (e.key === "Enter") doSearch();
  });
  el("loadMempool").addEventListener("click", loadMempool);

  await refreshChain();
  pollTimer = setInterval(refreshChain, 5000);
  setupWebSocket();
});
