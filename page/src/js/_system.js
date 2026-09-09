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
