async function systemConfig() {
  try {
    const response = await fetch(`${API}/v1/config/system`);
    if (response.ok) {
      return await response.json();
    }
  } catch (err) {
    console.error("systemConfig", err);
  }
  return { reply_lang: "auto", languages: [] };
}

async function startupConfig() {
  try {
    const response = await fetch(`${API}/v1/config/startup`);
    if (response.ok) {
      return ((await response.json()) || {}).enabled === true;
    }
  } catch (err) {
    console.error("startupConfig", err);
  }
  return false;
}

async function saveSystemStartup() {
  const select = $("#system-startup");
  if (!select) {
    return;
  }

  try {
    const response = await fetch(`${API}/v1/config/startup`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enable: select.value === "enabled" }),
    });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      alert(detail.error || `HTTP ${response.status}`);
    }
  } catch (err) {
    console.error("saveSystemStartup", err);
    alert(err.message || "failed");
  }
  renderSystem();
}

async function renderSystem() {
  const select = $("#system-lang");
  if (!select) {
    return;
  }

  const config = await systemConfig();
  const current = config.reply_lang || "auto";
  const languages = config.languages || [];

  select.innerHTML = "";
  for (const one of languages) {
    select.appendChild(_("option", { value: one.code }, one.label || one.code));
  }
  if (!languages.some((one) => one.code === current)) {
    select.appendChild(_("option", { value: current }, current));
  }
  select.value = current;

  const startup = $("#system-startup");
  if (startup) {
    startup.value = (await startupConfig()) ? "enabled" : "disabled";
  }

  const output = $("#system-output");
  if (output) {
    const dir = await outputDirConfig();
    output.value = dir.output_dir || "";
    output.placeholder = dir.resolved || "~/Downloads";
  }

  renderSystemVersion();
}

async function systemUpdateInfo() {
  try {
    const response = await fetch(`${API}/v1/system/update`);
    const detail = (await response.json().catch(() => ({}))) || {};
    return { ok: response.ok, ...detail };
  } catch (err) {
    console.error("systemUpdateInfo", err);
    return { ok: false };
  }
}

function renderUpdateEntry(available) {
  const entry = $("#left-tab-update");
  if (entry) {
    entry.hidden = !available;
  }
}

async function checkUpdate() {
  const detail = await systemUpdateInfo();
  renderUpdateEntry(detail.ok && detail.update_available === true);
}

function watchUpdate() {
  checkUpdate();
  setInterval(checkUpdate, UPDATE_INTERVAL);
}

const RELEASE_TAG_URL = "https://github.com/agenvoy/Agenvoy/releases/tag/";

async function renderSystemVersion() {
  const label = $("#system-version");
  const button = $("#system-update");
  const release = $("#system-release");
  if (!label || !button) {
    return;
  }

  label.textContent = "";
  button.hidden = true;
  if (release) {
    release.hidden = true;
  }

  const detail = await systemUpdateInfo();
  if (!detail.ok) {
    label.textContent = `${detail.version || "unknown"} (latest unavailable)`;
    return;
  }
  label.textContent = `${detail.version} (${detail.latest})`;
  button.hidden = !detail.update_available;
  if (release && detail.latest) {
    release.href = RELEASE_TAG_URL + encodeURIComponent(detail.latest);
    release.hidden = false;
  }
  renderUpdateEntry(detail.update_available === true);
}

async function outputDirConfig() {
  try {
    const response = await fetch(`${API}/v1/config/output_dir`);
    if (response.ok) {
      return (await response.json()) || {};
    }
  } catch (err) {
    console.error("outputDirConfig", err);
  }
  return {};
}

async function saveSystemOutput() {
  const input = $("#system-output");
  if (!input) {
    return;
  }

  try {
    const response = await fetch(`${API}/v1/config/output_dir`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ output_dir: input.value.trim() }),
    });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      alert(detail.error || `HTTP ${response.status}`);
    }
  } catch (err) {
    console.error("saveSystemOutput", err);
    alert(err.message || "failed");
  }
  renderSystem();
}

async function runSystemUpdate() {
  const detail = await systemUpdateInfo();
  const target = detail.ok ? `${detail.version} → ${detail.latest}` : "the latest release";
  if (!confirm(`Install ${target}?\nThis overwrites the installed binary.`)) {
    return;
  }

  const button = $("#system-update");
  if (button) {
    button.disabled = true;
  }

  try {
    const response = await fetch(`${API}/v1/system/update`, { method: "POST" });
    if (response.ok) {
      if (button) {
        button.textContent = "opened in Terminal";
      }
      return;
    }
    const detail = await response.json().catch(() => ({}));
    alert(detail.error || `HTTP ${response.status}`);
  } catch (err) {
    console.error("runSystemUpdate", err);
    alert(err.message || "failed");
  }

  if (button) {
    button.disabled = false;
  }
}

async function saveSystemLang() {
  const select = $("#system-lang");
  if (!select) {
    return;
  }

  try {
    const response = await fetch(`${API}/v1/config/system`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ reply_lang: select.value }),
    });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      alert(detail.error || `HTTP ${response.status}`);
    }
  } catch (err) {
    console.error("saveSystemLang", err);
    alert(err.message || "failed");
  }
  renderSystem();
}
