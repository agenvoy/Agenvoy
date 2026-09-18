const CHANNEL_SPEC = {
  telegram: { label: "Telegram", path: "/v1/channel/telegram", prefix: "tg" },
  discord: { label: "Discord", path: "/v1/channel/discord", prefix: "dc" },
};

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

async function renderChannel() {
  const status = await channelStatus();

  for (const kind of Object.keys(CHANNEL_SPEC)) {
    renderChannelCard(kind, status[kind] || {}, status.admin || {});
  }
  renderAdminChannel(status);
}

function renderChannelCard(kind, state, admin) {
  const row = $(`#${kind}-row`);
  if (!row) {
    return;
  }

  row.innerHTML = "";
  if (!state.enabled) {
    const token = _("input", {
      type: "password",
      autocomplete: "new-password",
      "data-1p-ignore": "true",
      "data-lpignore": "true",
      placeholder: "Bot token",
    });
    const enable = _("button.submit", { type: "button" }, "enable");
    enable.addEventListener("click", () => enableChannel(kind, token.value.trim()));
    token.addEventListener("keydown", (e) => {
      if (e.key === "Enter") {
        e.preventDefault();
        enableChannel(kind, token.value.trim());
      }
    });
    row.appendChild(token);
    row.appendChild(enable);
    return;
  }

  const prefix = CHANNEL_SPEC[kind].prefix;
  const count = (admin.chats || []).filter((chat) => chat.type === prefix).length;
  const manage = _("button.submit", { type: "button" }, "manage");
  manage.addEventListener("click", () => openChannelPopup(kind));
  row.appendChild(_("span", `${state.username || "connecting..."} · ${count} chats`));
  row.appendChild(manage);
}

function renderAdminChannel(status) {
  const card = $("#admin-card");
  const select = $("#admin-channel");
  if (!card || !select) {
    return;
  }

  const anyEnabled = Object.keys(CHANNEL_SPEC).some((kind) => (status[kind] || {}).enabled === true);
  card.hidden = !anyEnabled;
  if (!anyEnabled) {
    return;
  }

  const admin = status.admin || {};
  const current = admin.channel || "";
  const chats = admin.chats || [];

  select.innerHTML = "";
  select.appendChild(_("option", { value: "" }, "off"));
  for (const chat of chats) {
    select.appendChild(_("option", { value: chat.value }, `${chat.name || chat.id} · ${chat.type} · ${chat.id}`));
  }
  if (current !== "" && !admin.authorized) {
    select.appendChild(_("option", { value: current }, `${current} · not authorized`));
  }
  select.value = current;
  if (select.selectedIndex < 0) {
    select.selectedIndex = 0;
  }
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
  renderChannel();
}

async function openChannelPopup(kind) {
  const spec = CHANNEL_SPEC[kind];
  if (!spec) {
    return;
  }

  const list = _("div.list");
  const disable = _("button.remove", { type: "button" }, "disable");
  const cancel = _("button", { type: "button" }, "close");
  const root = _("div.popup", [
    _("div.panel", [
      _("strong", `${spec.label} · authorized chats`),
      _("p", "verified once, allowed since"),
      list,
      _("footer", [cancel, disable]),
    ]),
  ]);
  root.id = "channel-popup";

  const close = () => root.remove();
  cancel.addEventListener("click", close);
  disable.addEventListener("click", () => {
    close();
    disableChannel(kind);
  });
  root.addEventListener("click", (e) => {
    if (e.target === root) close();
  });

  document.body.appendChild(root);

  const chats = await channelChats(kind);
  if (chats.length === 0) {
    list.appendChild(_("p.empty", "none yet · message the bot and enter the verification code"));
    return;
  }

  for (const chat of chats) {
    const revoke = _("button.remove", { type: "button" }, "revoke");
    revoke.addEventListener("click", async () => {
      close();
      await revokeChannelChat(kind, chat);
    });
    list.appendChild(_("div.row", [_("p", `${chat.name || chat.id} · ${chat.id}`), revoke]));
  }
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

async function sendChannel(kind, action, token) {
  const spec = CHANNEL_SPEC[kind];
  if (!spec) {
    return;
  }

  const body = { action: action };
  if (token) {
    body.token = token;
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

function enableChannel(kind, token) {
  if (!token) {
    channelError("bot token is required to enable");
    return;
  }
  sendChannel(kind, "enable", token);
}

function disableChannel(kind) {
  const spec = CHANNEL_SPEC[kind];
  if (!spec || !confirm(`Disable ${spec.label}? The stored token is deleted.`)) {
    return;
  }
  sendChannel(kind, "disable", "");
}
