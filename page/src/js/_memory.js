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

const MEMORY_ACTIONS = [
  { key: "summary", label: "Regenerate summary", hint: "rebuild the rolling summary of this conversation" },
  { key: "compact", label: "Compact history", hint: "drop older messages, keep the summary" },
  { key: "reset", label: "Reset conversation", hint: "clear every message in this session" },
];

function openMemoryPicker() {
  const list = _("div.list");
  const boxes = [];

  for (const one of MEMORY_ACTIONS) {
    const box = _("input", { type: "radio", name: "memory-pick", value: one.key });
    boxes.push(box);
    list.appendChild(_("label", [box, _("div", [_("strong", one.label), _("p", one.hint)])]));
  }

  const cancel = _("button", { type: "button" }, "cancel");
  const run = _("button", { type: "button", class: "submit" }, "run");

  const root = _("div.popup", [_("div.panel", [_("strong", "Memory"), list, _("footer", [cancel, run])])]);
  root.id = "memory-popup";

  const close = () => root.remove();
  cancel.addEventListener("click", close);
  root.addEventListener("click", (e) => {
    if (e.target === root) close();
  });
  run.addEventListener("click", () => {
    const picked = boxes.find((box) => box.checked);
    if (!picked) {
      return;
    }
    close();
    if (picked.value === "summary") {
      memorySummary();
      return;
    }
    if (picked.value === "compact") {
      memoryCompact();
      return;
    }
    memoryReset();
  });

  document.body.appendChild(root);
}

async function memorySummary() {
  if (!confirm("Regenerate the summary for this conversation?")) {
    return;
  }
  const result = await memoryPost("summary");
  if (!result) {
    return;
  }
  alert(`summary regenerated · ${result.count || 0} entr${result.count === 1 ? "y" : "ies"}`);
}

async function memoryCompact() {
  if (!confirm("Compact this conversation? Older messages are dropped.")) {
    return;
  }
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
