const CHANNEL_SPEC = {
  telegram: { label: "Telegram", path: "/v1/channel/telegram" },
  discord: { label: "Discord", path: "/v1/channel/discord" },
  // LINE authenticates with a channel secret alongside the access token.
  line: { label: "LINE", path: "/v1/channel/line", secret: true },
};

function channelTarget() {
  const picked = praseURL().target || "";
  return picked === "telegram" || picked === "discord" || picked === "line" ? picked : "admin";
}

let channelActive = "admin";

function channelDom() {
  return {
    form: $("#channel-form"),
    status: $("#channel-status"),
    secret: $("#channel-secret"),
    token: $("#channel-token"),
    side: $("section.config div.side"),
    admin: $("#channel-admin"),
    chats: $("#channel-chats"),
  };
}

function channelError(text) {
  alert(text);
}

async function channelStatus() {
  try {
    const response = await fetch(`${API}/v1/channel`);
    if (response.ok) {
      return await response.json();
    }
  } catch (err) {
    console.error("channelStatus", err);
  }
  return {};
}

async function renderChannel() {
  const dom = channelDom();
  if (!dom.status) {
    return;
  }

  channelActive = channelTarget();
  const status = await channelStatus();

  for (const kind of Object.keys(CHANNEL_SPEC)) {
    const state = status[kind] || {};
    const button = $(`section.config div.side button[name="${kind}"]`);
    if (button) {
      button.dataset.state = state.enabled ? "on" : "off";
    }
  }
  for (const kind of Object.keys(CHANNEL_SPEC).concat(["admin"])) {
    const button = $(`section.config div.side button[name="${kind}"]`);
    if (button) {
      button.dataset.selected = kind === channelActive ? "1" : "0";
    }
  }

  if (dom.form) {
    dom.form.dataset.view = channelActive === "admin" ? "admin" : "channel";
    dom.form.dataset.kind = channelActive;
  }
  if (channelActive === "admin") {
    renderChannelChats("", false);
    renderAdminChannel();
    return;
  }

  const state = status[channelActive] || {};
  let title = "not connected";
  if (state.enabled) {
    title = state.username || "connecting...";
  }

  const name = _("input", { type: "text" });
  name.value = title;
  name.readOnly = true;

  dom.status.innerHTML = "";
  dom.status.appendChild(_("div.row", [name]));

  if (dom.form) {
    dom.form.dataset.enabled = state.enabled ? "1" : "0";
  }
  if (dom.token) {
    dom.token.value = "";
  }
  if (dom.secret) {
    dom.secret.value = "";
  }
  renderChannelChats(channelActive, state.enabled === true);
}

async function channelChats(kind) {
  try {
    const response = await fetch(`${API}/v1/channel/${encodeURIComponent(kind)}/chats`);
    if (response.ok) {
      return (await response.json()).chats || [];
    }
  } catch (err) {
    console.error("channelChats", err);
  }
  return [];
}

async function revokeChannelChat(kind, chat) {
  if (!confirm(`Revoke ${chat.name || chat.id}? It has to verify again to talk to the bot.`)) {
    return;
  }

  try {
    const response = await fetch(`${API}/v1/channel/${encodeURIComponent(kind)}/chat`, {
      method: "DELETE",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ id: chat.id }),
    });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      channelError(detail.error || `HTTP ${response.status}`);
      return;
    }
  } catch (err) {
    console.error("revokeChannelChat", err);
    channelError(err.message || "failed");
    return;
  }
  renderChannel();
}

async function renderChannelChats(kind, enabled) {
  const dom = channelDom();
  if (!dom.chats) {
    return;
  }

  dom.chats.innerHTML = "";
  if (!enabled) {
    delete dom.chats.dataset.open;
    return;
  }
  dom.chats.dataset.open = "1";
  dom.chats.appendChild(_("strong", "Authorized chats · verified once, allowed since"));

  const chats = await channelChats(kind);
  if (chats.length === 0) {
    dom.chats.appendChild(_("p.empty", "none yet · message the bot and enter the verification code"));
    return;
  }

  for (const chat of chats) {
    const remove = _("button.remove", { type: "button" }, "revoke");
    remove.addEventListener("click", () => revokeChannelChat(kind, chat));
    dom.chats.appendChild(_("div.row", [_("p", `${chat.name || chat.id} · ${chat.id}`), remove]));
  }
}

async function adminChannel() {
  try {
    const response = await fetch(`${API}/v1/channel`);
    if (response.ok) {
      return ((await response.json()) || {}).admin || {};
    }
  } catch (err) {
    console.error("adminChannel", err);
  }
  return { channel: "", authorized: false, chats: [] };
}

async function saveAdminChannel(value) {
  try {
    const response = await fetch(`${API}/v1/channel/admin`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ value: value }),
    });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      channelError(detail.error || `HTTP ${response.status}`);
    }
  } catch (err) {
    console.error("saveAdminChannel", err);
    channelError(err.message || "failed");
  }
  renderAdminChannel();
}

function adminChannelRow(label, description, checked, value) {
  const box = _("input", { type: "radio", name: "admin-channel" });
  box.checked = checked;
  box.addEventListener("change", () => {
    if (box.checked) {
      saveAdminChannel(value);
    }
  });
  return _("label.tool", [box, _("p", label), _("span", description || "")]);
}

async function renderAdminChannel() {
  const dom = channelDom();
  if (!dom.admin) {
    return;
  }

  const state = await adminChannel();
  const current = state.channel || "";
  const chats = state.chats || [];

  dom.admin.innerHTML = "";
  dom.admin.dataset.open = "1";
  dom.admin.appendChild(_("strong", "Admin channel · where new-chat verification codes are relayed"));
  dom.admin.appendChild(adminChannelRow("off", "codes stay in the log", current === "", ""));

  for (const chat of chats) {
    dom.admin.appendChild(
      adminChannelRow(chat.name || chat.id, `${chat.type} · ${chat.id}`, chat.value === current, chat.value),
    );
  }

  if (current !== "" && !state.authorized) {
    dom.admin.appendChild(
      adminChannelRow(current, "not in the authorized list · codes stay in the log", true, current),
    );
  }

  if (chats.length === 0) {
    dom.admin.appendChild(_("p.empty", "no authorized chats yet · message the bot once to authorize one"));
  }
}

function selectChannel(kind) {
  if (kind !== "admin" && !CHANNEL_SPEC[kind]) {
    return;
  }
  window.location.href = getLink({ page: "config", tab: "Channel", target: kind });
}

async function sendChannel(action, token, secret) {
  const spec = CHANNEL_SPEC[channelActive];
  if (!spec) {
    return;
  }

  const body = { action: action };
  if (token) {
    body.token = token;
  }
  if (secret) {
    body.secret = secret;
  }

  try {
    const response = await fetch(`${API}${spec.path}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      channelError(detail.error || `HTTP ${response.status}`);
      return;
    }
  } catch (err) {
    console.error("sendChannel", err);
    channelError(err.message || "failed");
    return;
  }
  renderChannel();
}

function enableChannel() {
  const spec = CHANNEL_SPEC[channelActive];
  if (!spec) {
    return;
  }

  const dom = channelDom();
  const token = dom.token ? dom.token.value.trim() : "";
  if (!token) {
    channelError(spec.secret ? "channel access token is required to enable" : "bot token is required to enable");
    return;
  }

  const secret = dom.secret ? dom.secret.value.trim() : "";
  if (spec.secret && !secret) {
    channelError("channel secret is required to enable");
    return;
  }
  sendChannel("enable", token, spec.secret ? secret : "");
}

function disableChannel() {
  const spec = CHANNEL_SPEC[channelActive];
  if (!spec || !confirm(`Disable ${spec.label}? The stored token is deleted.`)) {
    return;
  }
  sendChannel("disable", "", "");
}
