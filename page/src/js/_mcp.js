let mcpEditing = "";
let mcpStream = null;

function mcpDom() {
  return {
    form: $("#mcp-form"),
    list: $("#mcp-list"),
    name: $("#mcp-name"),
    transport: $("#mcp-transport"),
    auth: $("#mcp-auth"),
    command: $("#mcp-command"),
    args: $("#mcp-args"),
    env: $("#mcp-env"),
    url: $("#mcp-url"),
    headers: $("#mcp-headers"),
    oauth: $("#mcp-oauth"),
    tools: $("#mcp-tools"),
  };
}

function mcpError(text) {
  alert(text);
}

function mcpLines(text) {
  return (text || "")
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line.length > 0);
}

function mcpPairs(text, separator) {
  const dic = {};
  for (const line of mcpLines(text)) {
    const at = line.indexOf(separator);
    if (at < 1) continue;
    const key = line.slice(0, at).trim();
    if (!key) continue;
    dic[key] = line.slice(at + separator.length).trim();
  }
  return dic;
}

function mcpPairText(dic, separator) {
  return Object.entries(dic || {})
    .map(([k, v]) => `${k}${separator}${v}`)
    .join("\n");
}

async function mcpServers() {
  const out = { servers: {}, oauth: {}, status: {} };

  try {
    const [listRes, statusRes] = await Promise.all([fetch(`${API}/v1/mcp`), fetch(`${API}/v1/mcp/status`)]);
    if (listRes.ok) {
      const body = await listRes.json();
      out.servers = body.servers || {};
      out.oauth = body.oauth || {};
    }
    if (statusRes.ok) {
      for (const item of (await statusRes.json()).servers || []) {
        out.status[item.Name] = item;
      }
    }
  } catch (err) {
    console.error("mcpServers", err);
  }
  return out;
}

async function renderMcp() {
  const dom = mcpDom();
  if (!dom.list) {
    return;
  }

  const { servers, oauth, status } = await mcpServers();
  const names = Object.keys(servers).sort();

  dom.list.innerHTML = "";

  for (const name of names) {
    const server = servers[name] || {};
    const state = status[name];
    const transport = server.url ? "http" : "stdio";

    let mark = "not connected";
    let flag = "off";
    if (state && state.Connected) {
      mark = "connected";
      flag = "on";
    } else if (state && state.Error) {
      mark = state.Error;
      flag = "error";
    }
    if (server.auth === "oauth") {
      mark = `${mark} · oauth ${oauth[name] ? "authorized" : "pending"}`;
    }

    const remove = _("button", { type: "button" }, [_("span.material-symbols-outlined", "delete")]);
    remove.addEventListener("click", (e) => {
      e.stopPropagation();
      deleteMcp(name);
    });

    const card = _("div.card", [_("strong", name), _("p", `${transport} · ${mark}`), remove]);
    card.dataset.name = name;
    card.dataset.state = flag;
    card.addEventListener("click", () => openMcp(name));
    dom.list.appendChild(card);
  }
}

function fillMcpForm(name, server, authorized) {
  const dom = mcpDom();
  if (!dom.form) {
    return;
  }

  const transport = server.url ? "http" : "stdio";
  dom.name.value = name || "";
  dom.transport.value = transport;
  dom.auth.value = server.auth === "oauth" ? "oauth" : "";
  dom.command.value = server.command || "";
  dom.args.value = (server.args || []).join("\n");
  dom.env.value = mcpPairText(server.env, "=");
  dom.url.value = server.url || "";
  dom.headers.value = mcpPairText(server.headers, ": ");
  dom.form.dataset.transport = transport;

  mcpEditing = name || "";
  if (name) {
    dom.form.dataset.editing = "1";
  } else {
    delete dom.form.dataset.editing;
  }
  renderMcpOAuth(authorized);
  renderMcpTools(name);
}

async function openMcp(name) {
  const { servers, oauth } = await mcpServers();
  const server = servers[name];
  if (!server) {
    mcpError(`${name} not found`);
    return;
  }
  fillMcpForm(name, server, oauth[name] === true);
}

function resetMcp() {
  closeMcpLogin();
  fillMcpForm("", {}, false);
}

function mcpTransportChange() {
  const dom = mcpDom();
  if (dom.form) {
    dom.form.dataset.transport = dom.transport.value;
  }
  renderMcpOAuth(false);
}

function mcpAuthChange() {
  renderMcpOAuth(false);
}

async function saveMcp() {
  const dom = mcpDom();
  if (!dom.form) {
    return;
  }

  const name = dom.name.value.trim();
  if (!name) {
    mcpError("name is required");
    return;
  }

  const server = {};
  if (dom.transport.value === "http") {
    const url = dom.url.value.trim();
    if (!url) {
      mcpError("url is required for an http server");
      return;
    }
    server.url = url;
    const headers = mcpPairs(dom.headers.value, ":");
    if (Object.keys(headers).length > 0) {
      server.headers = headers;
    }
    if (dom.auth.value === "oauth") {
      server.auth = "oauth";
    }
  } else {
    const command = dom.command.value.trim();
    if (!command) {
      mcpError("command is required for a stdio server");
      return;
    }
    server.command = command;
    const args = mcpLines(dom.args.value);
    if (args.length > 0) {
      server.args = args;
    }
    const env = mcpPairs(dom.env.value, "=");
    if (Object.keys(env).length > 0) {
      server.env = env;
    }
  }

  try {
    const response = await fetch(`${API}/v1/mcp`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: name, server: server }),
    });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      mcpError(detail.error || `HTTP ${response.status}`);
      return;
    }
  } catch (err) {
    console.error("saveMcp", err);
    mcpError(err.message || "failed");
    return;
  }

  if (mcpEditing && mcpEditing !== name) {
    await removeMcpServer(mcpEditing);
  }
  await openMcp(name);
  renderMcp();
}

async function removeMcpServer(name) {
  const response = await fetch(`${API}/v1/mcp/remove`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name: name }),
  });
  if (!response.ok) {
    const detail = await response.json().catch(() => ({}));
    throw new Error(detail.error || `HTTP ${response.status}`);
  }
}

async function deleteMcp(name) {
  if (!name || !confirm(`Remove "${name}"?`)) {
    return;
  }

  try {
    await removeMcpServer(name);
  } catch (err) {
    console.error("deleteMcp", err);
    mcpError(err.message || "failed");
    return;
  }

  if (mcpEditing === name) {
    resetMcp();
  }
  renderMcp();
}

function deleteEditingMcp() {
  if (mcpEditing) {
    deleteMcp(mcpEditing);
  }
}

async function reconnectMcp() {
  try {
    const response = await fetch(`${API}/v1/mcp/reconnect`, { method: "POST" });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      mcpError(detail.error || `HTTP ${response.status}`);
      return;
    }
  } catch (err) {
    console.error("reconnectMcp", err);
    mcpError(err.message || "failed");
    return;
  }
  renderMcp();
}

function renderMcpOAuth(authorized, note) {
  const dom = mcpDom();
  if (!dom.oauth) {
    return;
  }

  dom.oauth.innerHTML = "";
  if (!mcpEditing || dom.transport.value !== "http" || dom.auth.value !== "oauth") {
    delete dom.oauth.dataset.open;
    delete dom.form.dataset.oauth;
    return;
  }
  dom.form.dataset.oauth = mcpStream ? "busy" : authorized ? "done" : "pending";

  if (!note) {
    delete dom.oauth.dataset.open;
    return;
  }

  dom.oauth.dataset.open = "1";
  dom.oauth.appendChild(_("div.row", [_("p", note)]));
}

function mcpToolPrefix(name) {
  return `mcp__${name}__`;
}

async function mcpToolList(prefix) {
  let tools = [];
  try {
    const response = await fetch(`${API}/v1/tools`);
    if (response.ok) {
      tools = (await response.json()).tools || [];
    }
  } catch (err) {
    console.error("mcpToolList", err);
  }

  return tools
    .filter((tool) => (tool.name || "").startsWith(prefix))
    .map((tool) => ({
      entry: tool.name,
      label: tool.name.slice(prefix.length),
      description: (tool.description || "").split("\n")[0],
    }))
    .sort((a, b) => a.label.localeCompare(b.label));
}

async function mcpGranted(prefix) {
  try {
    const response = await fetch(`${API}/v1/allowlist/tool?prefix=${encodeURIComponent(prefix)}`);
    if (response.ok) {
      return (await response.json()).entries || [];
    }
  } catch (err) {
    console.error("mcpGranted", err);
  }
  return [];
}

async function saveMcpPermission(prefix, entries) {
  try {
    const response = await fetch(`${API}/v1/allowlist/tool`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ prefix: prefix, entries: entries }),
    });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      mcpError(detail.error || `HTTP ${response.status}`);
    }
  } catch (err) {
    console.error("saveMcpPermission", err);
    mcpError(err.message || "failed");
  }
}

function mcpToolRow(label, description, checked, onToggle) {
  const box = _("input", { type: "checkbox" });
  box.checked = checked;
  box.addEventListener("change", () => onToggle(box.checked));

  return _("label.tool", [box, _("p", label), _("span", description || "")]);
}

async function renderMcpTools(name) {
  const dom = mcpDom();
  if (!dom.tools) {
    return;
  }

  dom.tools.innerHTML = "";
  if (!name) {
    delete dom.tools.dataset.open;
    return;
  }
  dom.tools.dataset.open = "1";

  const prefix = mcpToolPrefix(name);
  const [tools, granted] = await Promise.all([mcpToolList(prefix), mcpGranted(prefix)]);
  const all = granted.includes(`${prefix}*`);

  dom.tools.appendChild(_("strong", "Always allow · pick the tools that skip the confirmation prompt"));

  dom.tools.appendChild(
    mcpToolRow("all", "every tool of this server", all, (checked) => {
      saveMcpPermission(prefix, checked ? [`${prefix}*`] : []).then(() => renderMcpTools(name));
    }),
  );

  if (tools.length === 0) {
    dom.tools.appendChild(_("p.empty", "no tools · connect the server to see them"));
    return;
  }

  for (const tool of tools) {
    const checked = all || granted.includes(tool.entry);
    dom.tools.appendChild(
      mcpToolRow(tool.label, tool.description, checked, (on) => {
        let next = tools.filter((item) => (all || granted.includes(item.entry)) && item.entry !== tool.entry);
        if (on) {
          next = next.concat([tool]);
        }
        const entries = next.map((item) => item.entry);
        saveMcpPermission(prefix, entries.length === tools.length ? [`${prefix}*`] : entries).then(() =>
          renderMcpTools(name),
        );
      }),
    );
  }
}

function closeMcpLogin() {
  if (mcpStream) {
    mcpStream.close();
    mcpStream = null;
  }
}

function startMcpLogin(name) {
  closeMcpLogin();
  renderMcpOAuth(false, "waiting for the provider…");

  mcpStream = new EventSource(`${API}/v1/mcp/oauth?name=${encodeURIComponent(name)}`);
  mcpStream.onmessage = function (e) {
    let event = {};
    try {
      event = JSON.parse(e.data);
    } catch (err) {
      console.error("startMcpLogin", err);
      return;
    }

    if (event.url) {
      window.open(event.url, "_blank", "noreferrer");
      renderMcpLoginPaste(name, event.url);
      return;
    }
    if (!event.done) {
      return;
    }

    closeMcpLogin();
    if (!event.ok) {
      mcpError(event.error || "oauth failed");
    }
    openMcp(name);
    renderMcp();
  };
  mcpStream.onerror = function () {
    closeMcpLogin();
    renderMcpOAuth(false, "stream closed");
  };
}

function renderMcpLoginPaste(name, url) {
  const dom = mcpDom();
  if (!dom.oauth) {
    return;
  }

  const open = _("a", { href: url, target: "_blank", rel: "noreferrer" }, "open authorization page");
  const input = _("input", { type: "text", placeholder: "Paste the redirect URL if the browser cannot reach it" });
  const submit = _("button", { type: "button" }, "submit");
  submit.addEventListener("click", () => submitMcpCallback(name, input.value.trim()));

  dom.oauth.innerHTML = "";
  dom.oauth.dataset.open = "1";
  if (dom.form) {
    dom.form.dataset.oauth = "busy";
  }
  dom.oauth.appendChild(_("div.row", [open]));
  dom.oauth.appendChild(_("div.row", [input, submit]));
}

async function submitMcpCallback(name, url) {
  if (!url) {
    mcpError("redirect URL is required");
    return;
  }

  try {
    const response = await fetch(`${API}/v1/mcp/oauth/callback`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: name, url: url }),
    });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      mcpError(detail.error || `HTTP ${response.status}`);
    }
  } catch (err) {
    console.error("submitMcpCallback", err);
    mcpError(err.message || "failed");
  }
}

async function clearMcpOAuth(name) {
  if (!confirm(`Clear the stored token and client for "${name}"?`)) {
    return;
  }

  try {
    const response = await fetch(`${API}/v1/mcp/oauth`, {
      method: "DELETE",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: name }),
    });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      mcpError(detail.error || `HTTP ${response.status}`);
      return;
    }
  } catch (err) {
    console.error("clearMcpOAuth", err);
    mcpError(err.message || "failed");
    return;
  }
  openMcp(name);
  renderMcp();
}
