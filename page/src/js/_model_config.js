const providerProbe = {};
let modelView = "add";
let modelProvider = "";
let modelOpen = "";
let modelStream = null;
const modelOAuth = { id: "", code: "", url: "" };

function modelDom() {
  return {
    form: $("#model-form"),
    list: $("#model-list"),
    catalog: $("#model-catalog"),
    models: $("#model-models"),
  };
}

function modelError(text) {
  alert(text);
}

const PROVIDER_CONSOLE = {
  openai: "https://platform.openai.com/settings/organization/billing",
  claude: "https://console.anthropic.com/settings/billing",
  gemini: "https://aistudio.google.com/apikey",
  grok: "https://console.x.ai/",
  deepseek: "https://platform.deepseek.com/top_up",
  mistral: "https://console.mistral.ai/billing",
  nvidia: "https://build.nvidia.com/settings/api-keys",
  openrouter: "https://openrouter.ai/credits",
  cloudflare: "https://dash.cloudflare.com/profile/api-tokens",
};

function providerMethod(catalog, id) {
  const provider = catalog.find((item) => item.id === id);
  return Object.keys((provider || {}).methods || {})[0] || "api_key";
}

function providerFailure(id, method, error) {
  const status = (String(error).match(/HTTP (\d{3})/) || [])[1] || "";
  const text = String(error);

  if (/credit|spending limit|quota|insufficient|billing|payment/i.test(text)) {
    return {
      text: "This account is out of credits or has hit its spending limit. Top it up, then reload.",
      link: PROVIDER_CONSOLE[id],
      linkLabel: "open billing",
    };
  }
  if (method === "oauth") {
    return { text: "The login expired or was revoked. Sign in again to restore access.", login: true };
  }
  if (status === "401" || status === "403") {
    return {
      text: "The stored key was rejected. Delete it and paste a new one.",
      remove: true,
      link: PROVIDER_CONSOLE[id],
      linkLabel: "open console",
    };
  }
  return {
    text: `The provider answered ${status ? `HTTP ${status}` : "an error"} instead of a model list.`,
    remove: true,
    link: PROVIDER_CONSOLE[id],
    linkLabel: "open console",
  };
}

function providerKeyName(id) {
  const compat = id.match(/^compat\[(.+)\]$/);
  if (compat) {
    return `COMPAT_${compat[1].toUpperCase()}_API_KEY`;
  }
  return `${id.toUpperCase()}_API_KEY`;
}

async function removeProviderKey(id) {
  const key = providerKeyName(id);
  if (!confirm(`Delete ${key}?`)) {
    return;
  }

  try {
    const response = await fetch(`${API}/v1/key?key=${encodeURIComponent(key)}`, { method: "DELETE" });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      modelError(detail.error || `HTTP ${response.status}`);
      return;
    }
  } catch (err) {
    console.error("removeProviderKey", err);
    modelError(err.message || "failed");
    return;
  }

  delete providerProbe[id];
  selectProviderAdd();
}

async function providerCatalog() {
  try {
    const response = await fetch(`${API}/v1/providers`);
    if (response.ok) {
      return (await response.json()).providers || [];
    }
  } catch (err) {
    console.error("providerCatalog", err);
  }
  return [];
}

async function registeredModels() {
  try {
    const response = await fetch(`${API}/v1/models`);
    if (response.ok) {
      return ((await response.json()).data || []).map((item) => item.id).filter((id) => id && id !== "auto");
    }
  } catch (err) {
    console.error("registeredModels", err);
  }
  return [];
}

async function storedKeys() {
  try {
    const response = await fetch(`${API}/v1/keys`);
    if (response.ok) {
      return (await response.json()).keys || [];
    }
  } catch (err) {
    console.error("storedKeys", err);
  }
  return [];
}

function modelPrefix(id) {
  const at = id.indexOf("@");
  return at < 0 ? "" : id.slice(0, at);
}

async function probeProvider(prefix) {
  if (providerProbe[prefix]) {
    return providerProbe[prefix];
  }
  const result = await providerModelList(prefix);
  providerProbe[prefix] = result.error ? { ok: false, error: result.error } : { ok: true, models: result };
  return providerProbe[prefix];
}

async function renderModel() {
  const dom = modelDom();
  if (!dom.list) {
    return;
  }

  const [catalog, registered, keys] = await Promise.all([providerCatalog(), registeredModels(), storedKeys()]);

  const active = [];
  for (const provider of catalog) {
    if (keys.includes(providerKeyName(provider.id))) {
      active.push(provider.id);
    }
  }
  for (const id of registered) {
    const prefix = modelPrefix(id);
    if (prefix && !active.includes(prefix)) {
      active.push(prefix);
    }
  }

  const order = catalog.map((provider) => provider.id);
  const prefixes = order
    .filter((id) => active.includes(id))
    .concat(active.filter((id) => !order.includes(id)));

  const label = {};
  for (const provider of catalog) {
    label[provider.id] = provider.label;
  }
  const count = (prefix) => registered.filter((id) => modelPrefix(id) === prefix).length;

  dom.list.innerHTML = "";
  const add = $('section.config div.side button[name="add-provider"]');
  if (add) {
    add.dataset.selected = modelView === "add" ? "1" : "0";
  }

  for (const prefix of prefixes) {
    const total = count(prefix);
    const card = _("div.card", [
      _("strong", label[prefix] || prefix),
      _("p", `${total} model${total === 1 ? "" : "s"}`),
    ]);
    card.dataset.name = prefix;
    card.dataset.selected = modelView === "provider" && prefix === modelProvider ? "1" : "0";
    card.addEventListener("click", () => selectProvider(prefix));
    dom.list.appendChild(card);
  }

  if (dom.form) {
    dom.form.dataset.view = modelView;
  }
  if (modelView === "provider") {
    renderProviderModels(modelProvider, registered);
    return;
  }
  renderProviderCatalog(catalog, prefixes);
}

function selectProvider(prefix) {
  modelView = "provider";
  modelProvider = prefix;
  modelOpen = "";
  renderModel();
}

function selectProviderAdd() {
  modelView = "add";
  modelProvider = "";
  modelOpen = "";
  renderModel();
}

function providerDetails(provider, method, added) {
  const pills = [_("span", method)];
  if (added) {
    pills.push(_("span", "added"));
  }

  const summary = _("summary", [_("strong", provider.label), _("div.pills", pills)]);
  const details = _("details.provider", [summary, providerCredentialForm(provider, method)]);
  details.open = modelOpen === provider.id;
  details.addEventListener("toggle", () => {
    modelOpen = details.open ? provider.id : "";
  });
  return details;
}

function renderProviderCatalog(catalog, added) {
  const dom = modelDom();
  if (!dom.catalog) {
    return;
  }

  dom.catalog.innerHTML = "";
  dom.catalog.dataset.open = "1";

  for (const provider of catalog) {
    const method = Object.keys(provider.methods || {})[0] || "";
    dom.catalog.appendChild(providerDetails(provider, method, added.includes(provider.id)));
  }
}

function providerCredentialForm(provider, method) {
  if (method === "oauth") {
    if (modelOAuth.id === provider.id && modelOAuth.code) {
      return _("div.row", [_("p", "enter this code in the browser"), codeButton(modelOAuth.code)]);
    }
    if (modelOAuth.id === provider.id) {
      return _("div.row", [_("p", "waiting for the browser…")]);
    }
    const start = _("button.submit", { type: "button" }, "start login");
    start.addEventListener("click", () => startProviderOAuth(provider.id));
    return _("div.row", [_("p", "browser login · the daemon waits for the callback"), start]);
  }

  if (method === "custom") {
    const name = _("input", { type: "text", placeholder: "Name, e.g. ollama" });
    const url = _("input", { type: "text", placeholder: "http://127.0.0.1:11434/v1" });
    const key = _("input", { type: "password", placeholder: "API key · optional" });
    const save = _("button.submit", { type: "button" }, "save");
    save.addEventListener("click", () =>
      saveProviderKey(provider.id, { name: name.value.trim(), url: url.value.trim(), api_key: key.value.trim() }),
    );
    return _("div.field", [name, url, _("div.row", [key, save])]);
  }

  const key = _("input", { type: "password", placeholder: "API key" });

  if (provider.id === "cloudflare") {
    const account = _("input", { type: "text", placeholder: "Account ID" });
    const gateway = _("input", { type: "text", placeholder: "AI Gateway ID · optional" });
    const saveCf = _("button.submit", { type: "button" }, "save");
    saveCf.addEventListener("click", () =>
      saveProviderKey(provider.id, {
        api_key: key.value.trim(),
        account_id: account.value.trim(),
        gateway_id: gateway.value.trim(),
      }),
    );
    return _("div.field", [key, account, _("div.row", [gateway, saveCf])]);
  }

  const save = _("button.submit", { type: "button" }, "save");
  save.addEventListener("click", () => saveProviderKey(provider.id, { api_key: key.value.trim() }));
  return _("div.row", [key, save]);
}

async function saveProviderKey(id, body) {
  if (!body.api_key && id !== "compat") {
    modelError("api key is required");
    return;
  }
  if (id === "compat" && (!body.name || !body.url)) {
    modelError("name and url are required");
    return;
  }
  if (id === "cloudflare" && !body.account_id) {
    modelError("account ID is required for Cloudflare");
    return;
  }

  try {
    const response = await fetch(`${API}/v1/provider/${encodeURIComponent(id)}/key`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      modelError(detail.error || `HTTP ${response.status}`);
      return;
    }
  } catch (err) {
    console.error("saveProviderKey", err);
    modelError(err.message || "failed");
    return;
  }

  modelOpen = "";
  const prefix = id === "compat" ? `compat[${body.name}]` : id;
  delete providerProbe[prefix];
  selectProvider(prefix);
}

function codeButton(code) {
  const button = _("button.code", { type: "button" }, [
    _("span.material-symbols-outlined", "content_copy"),
    _("p", `code: ${code}`),
  ]);
  button.addEventListener("click", () => {
    navigator.clipboard.writeText(code).catch((err) => console.error("codeButton", err));
    const label = button.querySelector("p");
    label.textContent = "copied";
    setTimeout(() => (label.textContent = `code: ${code}`), 1000);
  });
  return button;
}

function closeProviderOAuth() {
  if (modelStream) {
    modelStream.close();
    modelStream = null;
  }
}

function startProviderOAuth(id) {
  closeProviderOAuth();
  modelOAuth.id = id;
  modelOAuth.code = "";
  modelOAuth.url = "";
  modelOpen = id;
  modelView = "add";
  renderModel();

  modelStream = new EventSource(`${API}/v1/provider/${encodeURIComponent(id)}/oauth`);
  modelStream.onmessage = function (e) {
    let event = {};
    try {
      event = JSON.parse(e.data);
    } catch (err) {
      console.error("startProviderOAuth", err);
      return;
    }

    if (event.url) {
      modelOAuth.url = event.url;
      modelOAuth.code = event.user_code || "";
      window.open(event.url, "_blank", "noreferrer");
      renderModel();
      return;
    }
    if (!event.done) {
      return;
    }

    closeProviderOAuth();
    modelOAuth.id = "";
    modelOAuth.code = "";
    if (!event.ok) {
      modelError(event.error || "oauth failed");
      renderModel();
      return;
    }
    modelOpen = "";
    delete providerProbe[id];
    selectProvider(id);
  };
  modelStream.onerror = function () {
    closeProviderOAuth();
    modelOAuth.id = "";
    modelOAuth.code = "";
    renderModel();
  };
}

async function providerModelList(prefix) {
  const target = prefix.startsWith("compat[") ? "compat" : prefix;
  try {
    const response = await fetch(`${API}/v1/provider/${encodeURIComponent(target)}/models`);
    if (response.ok) {
      return (await response.json()).models || [];
    }
    const detail = await response.json().catch(() => ({}));
    return { error: detail.error || `HTTP ${response.status}` };
  } catch (err) {
    console.error("providerModelList", err);
    return { error: err.message || "failed" };
  }
}

async function saveProviderModels(prefix, models) {
  try {
    const response = await fetch(`${API}/v1/models`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ prefix: prefix, models: models }),
    });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      modelError(detail.error || `HTTP ${response.status}`);
    }
  } catch (err) {
    console.error("saveProviderModels", err);
    modelError(err.message || "failed");
  }
  renderModel();
}

function modelRow(label, checked, onToggle) {
  const box = _("input", { type: "checkbox" });
  box.checked = checked;
  box.addEventListener("change", () => onToggle(box.checked));
  return _("label.tool", [box, _("p", label)]);
}

async function renderProviderModels(prefix, registered) {
  const dom = modelDom();
  if (!dom.models || !prefix) {
    return;
  }

  dom.models.innerHTML = "";
  dom.models.dataset.open = "1";
  dom.models.appendChild(_("strong", `${prefix} · pick the models this agent can use`));

  const probe = await probeProvider(prefix);
  if (!probe.ok) {
    const catalog = await providerCatalog();
    const failure = providerFailure(prefix, providerMethod(catalog, prefix), probe.error);

    dom.models.appendChild(_("p.empty", failure.text));

    const actions = [];
    if (failure.login) {
      const login = _("button.submit", { type: "button" }, "sign in again");
      login.addEventListener("click", () => startProviderOAuth(prefix));
      actions.push(login);
    }
    if (failure.remove) {
      const remove = _("button.remove", { type: "button" }, "delete key");
      remove.addEventListener("click", () => removeProviderKey(prefix));
      actions.push(remove);
    }
    if (failure.link) {
      actions.push(_("a.submit", { href: failure.link, target: "_blank", rel: "noreferrer" }, failure.linkLabel));
    }
    if (actions.length > 0) {
      dom.models.appendChild(_("div.row", [_("p", "")].concat(actions)));
    }

    return;
  }
  const available = probe.models;

  const active = (registered || [])
    .filter((id) => modelPrefix(id) === prefix)
    .map((id) => id.slice(prefix.length + 1));
  const names = available.slice().sort();
  for (const name of active) {
    if (!names.includes(name)) {
      names.push(name);
    }
  }

  if (names.length === 0) {
    dom.models.appendChild(_("p.empty", "no models returned by this provider"));
    return;
  }

  const filter = _("input", { type: "text", placeholder: `Filter ${names.length} models` });
  filter.addEventListener("input", () => {
    const query = filter.value.trim().toLowerCase();
    for (const row of dom.models.querySelectorAll("label.tool")) {
      row.style.display = row.textContent.toLowerCase().includes(query) ? "" : "none";
    }
  });
  dom.models.appendChild(filter);

  for (const name of names) {
    dom.models.appendChild(
      modelRow(name, active.includes(name), (on) => {
        const next = on ? active.concat([name]) : active.filter((item) => item !== name);
        saveProviderModels(prefix, next);
      }),
    );
  }
}
