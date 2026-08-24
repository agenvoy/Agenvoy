const CHANNEL_SPEC = {
  telegram: { label: "Telegram", path: "/v1/channel/telegram" },
  discord: { label: "Discord", path: "/v1/channel/discord" },
};

let channelActive = "telegram";

function channelDom() {
  return {
    form: $("#channel-form"),
    status: $("#channel-status"),
    token: $("#channel-token"),
    side: $("section.config div.side"),
    admin: $("#channel-admin"),
  };
}

function channelError(text) {
  alert(text);
}

async function channelStatus() {
  try {
    const response = await fetch(`${API}/v1/channel/status`);
    if (response.ok) {
      return await response.json();
    }
  } catch (err) {
    console.error("channelStatus", err);
  }
  return {};
}

function channelPill(text) {
  return _("span", text);
}

async function renderChannel() {
  const dom = channelDom();
  if (!dom.status) {
    return;
  }

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
  }
  if (channelActive === "admin") {
    renderAdminChannel();
    return;
  }

  const state = status[channelActive] || {};
  let title = "not connected";
  if (state.enabled) {
    title = state.username || "connecting…";
  }

  dom.status.innerHTML = "";
  dom.status.appendChild(_("strong", title));
  dom.status.appendChild(
    _("div.pills", [
      channelPill(state.enabled ? "enabled" : "disabled"),
      channelPill(state.has_token ? "token set" : "no token"),
    ]),
  );

  if (dom.form) {
    dom.form.dataset.enabled = state.enabled ? "1" : "0";
  }
  if (dom.token) {
    dom.token.value = "";
  }
}

async function adminChannel() {
  try {
    const response = await fetch(`${API}/v1/channel/admin`);
    if (response.ok) {
      return await response.json();
    }
  } catch (err) {
    console.error("adminChannel", err);
  }
  return { admin_channel: "", authorized: false, chats: [] };
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
  const current = state.admin_channel || "";
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
    dom.admin.appendChild(adminChannelRow(current, "not in the authorized list · codes stay in the log", true, current));
  }

  if (chats.length === 0) {
    dom.admin.appendChild(_("p.empty", "no authorized chats yet · message the bot once to authorize one"));
  }
}

function selectChannel(kind) {
  if (kind !== "admin" && !CHANNEL_SPEC[kind]) {
    return;
  }
  channelActive = kind;
  renderChannel();
}

async function sendChannel(action, token) {
  const spec = CHANNEL_SPEC[channelActive];
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

function enableChannel() {
  const dom = channelDom();
  const token = dom.token ? dom.token.value.trim() : "";
  if (!token) {
    channelError("bot token is required to enable");
    return;
  }
  sendChannel("enable", token);
}

function disableChannel() {
  const spec = CHANNEL_SPEC[channelActive];
  if (!spec || !confirm(`Disable ${spec.label}? The stored token is deleted.`)) {
    return;
  }
  sendChannel("disable", "");
}
