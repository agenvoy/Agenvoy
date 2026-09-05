const providerProbe = {};
let providerUsage = null;
let providerUsageLoading = null;
let modelView = "add";
let modelProvider = "";
let modelOpen = "";
let modelAbort = null;
let modelStream = null;
const modelOAuth = { id: "", code: "", url: "" };

function modelDom() {
  return {
    form: $("#model-form"),
    list: $("#model-list"),
    catalog: $("#model-catalog"),
    routing: $("#model-routing"),
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

function providerKeyNames(id) {
  if (id === "cloudflare") {
    return ["CLOUDFLARE_API_KEY", "CLOUDFLARE_ACCOUNT_ID", "CLOUDFLARE_GATEWAY_ID"];
  }
  return [providerKeyName(id)];
}

async function deleteKey(key) {
  try {
    const response = await fetch(`${API}/v1/key?key=${encodeURIComponent(key)}`, { method: "DELETE" });
    if (!response.ok && response.status !== 404) {
      const detail = await response.json().catch(() => ({}));
      console.error("deleteKey", key, detail.error || response.status);
    }
  } catch (err) {
    console.error("deleteKey", key, err);
  }
}

async function deleteProvider(prefix) {
  const catalog = await providerCatalog();
  const method = providerMethod(catalog, prefix);
  const target = method === "oauth" ? "its stored login" : providerKeyNames(prefix).join(", ");
  if (!confirm(`Remove ${prefix}? Its registered models and ${target} are deleted.`)) {
    return;
  }

  await saveProviderModels(prefix, []).catch((err) => console.error("deleteProvider models", err));

  if (method === "oauth") {
    try {
      const response = await fetch(`${API}/v1/provider/${encodeURIComponent(prefix)}/oauth`, { method: "DELETE" });
      if (!response.ok) {
        const detail = await response.json().catch(() => ({}));
        modelError(detail.error || `HTTP ${response.status}`);
      }
    } catch (err) {
      console.error("deleteProvider oauth", err);
    }
  } else {
    for (const key of providerKeyNames(prefix)) {
      await deleteKey(key);
    }
  }

  delete providerProbe[prefix];
  providerUsage = null;
  selectProviderAdd();
}

async function removeProviderKey(id) {
  const keys = providerKeyNames(id);
  if (!confirm(`Delete ${keys.join(", ")}?`)) {
    return;
  }

  for (const key of keys) {
    await deleteKey(key);
  }

  delete providerProbe[id];
  providerUsage = null;
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

function fetchProviderUsage() {
  if (providerUsage) {
    return Promise.resolve(providerUsage);
  }
  if (providerUsageLoading) {
    return providerUsageLoading;
  }

  providerUsageLoading = fetch(`${API}/v1/providers/usage`)
    .then((response) => (response.ok ? response.json() : { usage: {} }))
    .then((body) => {
      providerUsage = body.usage || {};
      return providerUsage;
    })
    .catch((err) => {
      console.error("fetchProviderUsage", err);
      return {};
    })
    .finally(() => {
      providerUsageLoading = null;
    });
  return providerUsageLoading;
}

function applyUsageBadges(usage) {
  const dom = modelDom();
  if (!dom.list) {
    return;
  }

  for (const card of dom.list.querySelectorAll("div.card")) {
    const existing = card.querySelector("span.quota");
    if (existing) {
      existing.remove();
    }
    const quota = usageLabel((usage || {})[card.dataset.name]);
    if (!quota) {
      continue;
    }
    const badge = _("span.quota", quota.text);
    badge.dataset.state = quota.state;
    card.insertBefore(badge, card.querySelector("button"));
  }
}

function usageLabel(entry) {
  if (!entry || entry.value === undefined) {
    return null;
  }
  if (entry.kind === "balance") {
    return { text: `$${entry.value.toFixed(2)}`, state: entry.value > 0 ? "on" : "error" };
  }
  const value = Math.round(entry.value);
  let state = "on";
  if (value < 20) {
    state = "error";
  } else if (value < 50) {
    state = "warn";
  }
  return { text: `${value}%`, state: state };
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
  if (result.aborted) {
    return { aborted: true };
  }
  providerProbe[prefix] = result.error ? { ok: false, error: result.error } : { ok: true, models: result };
  return providerProbe[prefix];
}

async function renderModel() {
  const dom = modelDom();
  if (!dom.list) {
    return;
  }

  const [catalog, registered, keys] = await Promise.all([providerCatalog(), registeredModels(), storedKeys()]);
  const usage = providerUsage;

  const active = [];
  for (const provider of catalog) {
    if (provider.logged_in || keys.includes(providerKeyName(provider.id))) {
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
  const prefixes = order.filter((id) => active.includes(id)).concat(active.filter((id) => !order.includes(id)));

  const label = {};
  for (const provider of catalog) {
    label[provider.id] = provider.label;
  }
  const count = (prefix) => registered.filter((id) => modelPrefix(id) === prefix).length;

  const picked = praseURL().target || "";
  if (picked && prefixes.includes(picked)) {
    modelView = "provider";
    modelProvider = picked;
  } else {
    modelView = "add";
    modelProvider = "";
  }

  dom.list.innerHTML = "";
  for (const [name, view] of [["add-provider", "add"]]) {
    const button = $(`section.config div.side button[name="${name}"]`);
    if (button) {
      button.dataset.selected = modelView === view ? "1" : "0";
    }
  }

  for (const prefix of prefixes) {
    const total = count(prefix);
    const children = [_("strong", label[prefix] || prefix), _("p", `${total} model${total === 1 ? "" : "s"}`)];

    const quota = usageLabel((usage || {})[prefix]);
    if (quota) {
      const badge = _("span.quota", quota.text);
      badge.dataset.state = quota.state;
      children.push(badge);
    }

    const remove = _("button", { type: "button" }, [_("span.material-symbols-outlined", "delete")]);
    remove.addEventListener("click", (e) => {
      e.stopPropagation();
      deleteProvider(prefix);
    });
    children.push(remove);

    const card = _("div.card", children);
    card.dataset.name = prefix;
    card.dataset.selected = modelView === "provider" && prefix === modelProvider ? "1" : "0";
    card.addEventListener("click", () => selectProvider(prefix));
    dom.list.appendChild(card);
  }

  if (!providerUsage) {
    fetchProviderUsage().then(applyUsageBadges);
  }

  if (dom.form) {
    dom.form.dataset.view = modelView;
  }
  if (modelView === "provider") {
    renderProviderModels(modelProvider, registered);
    return;
  }
  renderProviderCatalog(catalog, prefixes);
  renderModelRouting(registered);
}

async function modelRouting() {
  try {
    const response = await fetch(`${API}/v1/model`);
    if (response.ok) {
      const body = (await response.json()) || {};
      return {
        dispatcher: body.dispatcher || "",
        summary: body.summary || "",
        image: body.image || "",
        imageOptions: body.image_options || [],
        imageProviders: body.image_providers || [],
        stt: body.stt || "",
        tts: body.tts || "",
        audioProviders: body.audio_providers || [],
      };
    }
  } catch (err) {
    console.error("modelRouting", err);
  }
  return { dispatcher: "", summary: "", image: "", imageOptions: [], imageProviders: [], stt: "", tts: "", audioProviders: [] };
}

async function saveRoutingModel(kind, model) {
  try {
    const response = await fetch(`${API}/v1/model`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ [kind]: model }),
    });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      modelError(detail.error || `HTTP ${response.status}`);
    }
  } catch (err) {
    console.error("saveRoutingModel", err);
    modelError(err.message || "failed");
  }
  renderModel();
}

function routingLabel(label, providers) {
  const children = [_("strong", label)];
  if (providers && providers.length > 0) {
    children.push(_("p", providers.join(" / ")));
  }
  return _("div.label", children);
}

function routingRow(label, kind, current, options) {
  const select = _("select");
  select.appendChild(_("option", { value: "" }, "auto · first registered model"));
  for (const name of options) {
    select.appendChild(_("option", { value: name }, name));
  }
  select.value = current;
  select.addEventListener("change", () => saveRoutingModel(kind, select.value));

  return _("div.routing", [routingLabel(label), select]);
}

function audioRow(label, kind, current, providers) {
  const select = _("select");
  select.dataset.kind = kind;
  select.appendChild(_("option", { value: "" }, "off"));
  if (current) {
    select.appendChild(_("option", { value: current }, current));
  }
  select.value = current;
  select.addEventListener("change", () => saveRoutingModel(kind, select.value));

  return _("div.routing", [routingLabel(label, providers), select]);
}

async function fillAudioOptions(dom) {
  let body = {};
  try {
    const response = await fetch(`${API}/v1/model/audio`);
    if (!response.ok) {
      return;
    }
    body = (await response.json()) || {};
  } catch (err) {
    console.error("fillAudioOptions", err);
    return;
  }

  for (const [kind, options] of [
    ["stt", body.stt_options || []],
    ["tts", body.tts_options || []],
  ]) {
    const select = dom.querySelector(`select[data-kind="${kind}"]`);
    if (!select) {
      continue;
    }
    const current = select.value;
    select.innerHTML = "";
    select.appendChild(_("option", { value: "" }, "off"));
    for (const name of options) {
      select.appendChild(_("option", { value: name }, name));
    }
    select.value = options.includes(current) ? current : "";
  }
}

function imageRow(current, added, providers) {
  const select = _("select");
  select.appendChild(_("option", { value: "" }, "off"));
  for (const id of added) {
    select.appendChild(_("option", { value: id }, id));
  }
  select.value = added.includes(current) ? current : "";
  select.addEventListener("change", () => saveRoutingModel("image", select.value));

  return _("div.routing", [routingLabel("Image", providers), select]);
}

async function renderModelRouting(registered) {
  const dom = modelDom();
  if (!dom.routing) {
    return;
  }

  const routing = await modelRouting();
  if (modelView !== "add") {
    return;
  }

  dom.routing.innerHTML = "";
  dom.routing.dataset.open = "1";

  const box = _("section");
  if (registered.length === 0) {
    box.appendChild(_("p.empty", "no models registered yet · add one from a provider first"));
  } else {
    box.appendChild(routingRow("Dispatcher", "dispatcher", routing.dispatcher, registered));
    box.appendChild(routingRow("Summary", "summary", routing.summary, registered));
    box.appendChild(imageRow(routing.image, routing.imageOptions, routing.imageProviders));
    box.appendChild(audioRow("Speech to text", "stt", routing.stt, routing.audioProviders));
    box.appendChild(audioRow("Text to speech", "tts", routing.tts, routing.audioProviders));
  }

  const group = _("details.group", { open: "" }, [_("summary", ["Setting Model", _("span.material-symbols-outlined", "keyboard_arrow_down")]), box]);
  dom.routing.appendChild(group);
  fillAudioOptions(group);
}

function selectProvider(prefix) {
  window.location.href = getLink({ page: "config", tab: "Model", target: prefix });
}

function selectProviderAdd() {
  window.location.href = getLink({ page: "config", tab: "Model" });
}

function providerDetails(provider, method, added) {
  const label = MODEL_CUSTOM.includes(provider.id) ? "custom" : method;
  const pill = _("span", label);
  pill.dataset.method = label;

  const summary = _("summary", [_("strong", provider.label), _("div.pills", [pill])]);
  const details = _("details.provider", [summary, providerCredentialForm(provider, method, added)]);
  details.dataset.added = added ? "1" : "0";
  details.open = modelOpen === provider.id;
  details.addEventListener("toggle", () => {
    modelOpen = details.open ? provider.id : "";
  });
  return details;
}

const MODEL_CUSTOM = ["cloudflare", "compat"];

function modelFilter() {
  const nav = $("#model-filter");
  const active = nav && nav.querySelector('button[data-selected="1"]');
  return active ? active.name : "all";
}

function setModelFilter(value) {
  const nav = $("#model-filter");
  if (!nav) {
    return;
  }
  for (const one of nav.querySelectorAll("button")) {
    if (one.name === value) {
      one.dataset.selected = "1";
    } else {
      delete one.dataset.selected;
    }
  }
}

function modelFilterChange(e) {
  const button = e && e.target ? e.target.closest("button") : null;
  if (!button || !button.name) {
    return;
  }
  setModelFilter(button.name);
  renderModel();
}

function matchModelFilter(provider, filter) {
  const custom = MODEL_CUSTOM.includes(provider.id);
  if (filter === "custom") {
    return custom;
  }
  if (custom) {
    return filter === "all";
  }
  if (filter === "all") {
    return true;
  }
  return Object.keys(provider.methods || {}).includes(filter);
}

function renderProviderCatalog(catalog, added) {
  const dom = modelDom();
  if (!dom.catalog) {
    return;
  }

  dom.catalog.innerHTML = "";
  dom.catalog.dataset.open = "1";

  const filter = modelFilter();
  const visible = catalog.filter((item) => matchModelFilter(item, filter));

  const append = (provider) => {
    const method = Object.keys(provider.methods || {})[0] || "";
    dom.catalog.appendChild(providerDetails(provider, method, added.includes(provider.id)));
  };

  for (const provider of visible.filter((item) => !added.includes(item.id))) {
    append(provider);
  }

  const done = visible.filter((item) => added.includes(item.id));
  if (done.length === 0) {
    return;
  }
  const box = _("section");
  for (const provider of done) {
    const method = Object.keys(provider.methods || {})[0] || "";
    box.appendChild(providerDetails(provider, method, true));
  }
  dom.catalog.appendChild(_("details.group", [_("summary", ["Added", _("span.material-symbols-outlined", "keyboard_arrow_down")]), box]));
}

function providerCredentialForm(provider, method, added) {
  const submitLabel = added ? "renew" : method === "oauth" ? "login" : "add";

  if (method === "oauth") {
    if (modelOAuth.id === provider.id && modelOAuth.code) {
      return _("div.row", [_("p", "enter this code in the browser"), codeButton(modelOAuth.code)]);
    }
    if (modelOAuth.id === provider.id) {
      return _("div.row", [_("p", "waiting for the browser...")]);
    }
    const start = _("button.submit", { type: "button" }, submitLabel);
    start.addEventListener("click", () => startProviderOAuth(provider.id));
    return _("div.row", [_("p", "browser login · the daemon waits for the callback"), start]);
  }

  if (method === "custom") {
    const name = _("input", { type: "text", placeholder: "Name, e.g. ollama" });
    const url = _("input", { type: "text", placeholder: "http://127.0.0.1:11434/v1" });
    const key = _("input", { type: "password", placeholder: "API key · optional" });
    const save = _("button.submit", { type: "button" }, submitLabel);
    save.addEventListener("click", () =>
      saveProviderKey(provider.id, { name: name.value.trim(), url: url.value.trim(), api_key: key.value.trim() }),
    );
    return _("div.field", [name, url, _("div.row", [key, save])]);
  }

  const key = _("input", { type: "password", placeholder: "API key" });

  if (provider.id === "cloudflare") {
    const account = _("input", { type: "text", placeholder: "Account ID" });
    const gateway = _("input", { type: "text", placeholder: "AI Gateway ID · optional" });
    const saveCf = _("button.submit", { type: "button" }, submitLabel);
    saveCf.addEventListener("click", () =>
      saveProviderKey(provider.id, {
        api_key: key.value.trim(),
        account_id: account.value.trim(),
        gateway_id: gateway.value.trim(),
      }),
    );
    return _("div.field", [key, account, _("div.row", [gateway, saveCf])]);
  }

  const save = _("button.submit", { type: "button" }, submitLabel);
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

  if (id === "openai") {
    alert(
      "OpenAI key saved.\n\n" +
        "Vector storage needs a daemon restart before it works:\n" +
        "    agen stop && agen\n\n" +
        "Entries written before the restart keep no vector.",
    );
  }

  modelOpen = "";
  const prefix = id === "compat" ? `compat[${body.name}]` : id;
  delete providerProbe[prefix];
  providerUsage = null;
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
    providerUsage = null;
    selectProvider(id);
  };
  modelStream.onerror = function () {
    closeProviderOAuth();
    modelOAuth.id = "";
    modelOAuth.code = "";
    renderModel();
  };
}

function cancelModelFetch() {
  if (modelAbort) {
    modelAbort.abort();
    modelAbort = null;
  }
}

async function providerModelList(prefix) {
  const target = prefix.startsWith("compat[") ? "compat" : prefix;
  const controller = new AbortController();
  cancelModelFetch();
  modelAbort = controller;

  try {
    const response = await fetch(`${API}/v1/provider/${encodeURIComponent(target)}/models`, {
      signal: controller.signal,
    });
    if (response.ok) {
      return (await response.json()).models || [];
    }
    const detail = await response.json().catch(() => ({}));
    return { error: detail.error || `HTTP ${response.status}` };
  } catch (err) {
    if (err.name === "AbortError") {
      return { aborted: true };
    }
    console.error("providerModelList", err);
    return { error: err.message || "failed" };
  } finally {
    if (modelAbort === controller) {
      modelAbort = null;
    }
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

  const remove = _("button.remove", { type: "button" }, "delete");
  remove.addEventListener("click", () => deleteProvider(prefix));

  dom.models.innerHTML = "";
  dom.models.dataset.open = "1";
  dom.models.appendChild(_("div.row", [_("strong", `${prefix} · pick the models this agent can use`), remove]));

  const probe = await probeProvider(prefix);
  if (probe.aborted) {
    return;
  }

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

  const active = (registered || []).filter((id) => modelPrefix(id) === prefix).map((id) => id.slice(prefix.length + 1));
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
