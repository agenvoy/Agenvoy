async function memoryPost(path, body) {
  if (!currentSessionId) {
    return null;
  }

  try {
    const response = await fetch(`${API}/v1/session/${encodeURIComponent(currentSessionId)}/${path}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body || {}),
    });
    const detail = await response.json().catch(() => ({}));
    if (!response.ok) {
      console.error("memoryPost", path, response.status, detail);
      alert(`${path} failed: ${detail.error || response.status}`);
      return null;
    }
    return detail;
  } catch (err) {
    console.error("memoryPost", path, err);
    alert(`${path} failed: ${err.message || err}`);
    return null;
  }
}

async function memorySummary() {
  const result = await memoryPost("summary");
  if (!result) {
    return;
  }
  alert(`summary regenerated · ${result.count || 0} entr${result.count === 1 ? "y" : "ies"}`);
}

async function memoryCompact() {
  const result = await memoryPost("compact");
  if (!result) {
    return;
  }
  if (!result.removed) {
    alert("nothing to compact");
    return;
  }
  alert(`compacted · ${result.removed} removed`);
  window.location.reload();
}

async function memoryReset() {
  if (!confirm("Clear the whole conversation?")) {
    return;
  }
  const keep = confirm("Keep the summary? (cancel wipes it as well)");
  const result = await memoryPost("reset", { mode: keep ? "summary" : "all" });
  if (!result) {
    return;
  }
  alert(`conversation cleared · ${result.removed || 0} removed${keep ? " · summary kept" : ""}`);
  window.location.reload();
}
