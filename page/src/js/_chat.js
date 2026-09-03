let chatListRetry = 0;

function retryChatList() {
  clearTimeout(chatListRetry);
  chatListRetry = setTimeout(renderChatList, 3000);
}

async function renderChatList() {
  const dom = $("#left-tab-chat-list");
  const pinDom = $("#left-tab-pin-list");
  const discordDom = $("#left-tab-discord-list");
  const telegramDom = $("#left-tab-telegram-list");
  const terminalDom = $("#left-tab-terminal-list");
  const tempDom = $("#left-tab-temp-list");
  if (!dom) {
    return;
  }

  let list = [];
  try {
    const response = await fetch(`${API}/v1/sessions`);
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }
    list = (await response.json()).sessions || [];
  } catch (err) {
    console.error("loadChatList", err);
    retryChatList();
    return;
  }
  clearTimeout(chatListRetry);

  dom.innerHTML = "";
  for (const one of [pinDom, discordDom, telegramDom, terminalDom, tempDom]) {
    if (one) {
      one.innerHTML = "";
    }
  }

  const pinned = readConfig().pin_chat || [];

  for (const e of list) {
    if (pinned.includes(e.id)) {
      if (pinDom) {
        pinDom.appendChild(pinListItem(e.id, e.name || e.id));
      }
      continue;
    }
    if (e.id.startsWith("chat-")) {
      dom.appendChild(chatListItem(e.id, e.name || e.id));
      continue;
    }
    if (e.id.startsWith("cli-")) {
      if (terminalDom) {
        terminalDom.appendChild(chatListItem(e.id, e.name || e.id));
      }
      continue;
    }
    if (e.id.startsWith("temp-")) {
      if (tempDom) {
        tempDom.appendChild(chatListItem(e.id, e.name || e.id));
      }
      continue;
    }
    if (e.id.startsWith("dc-")) {
      if (discordDom) {
        discordDom.appendChild(channelListItem(e.id, e.name || e.id));
      }
      continue;
    }
    if (e.id.startsWith("tg-")) {
      if (telegramDom) {
        telegramDom.appendChild(channelListItem(e.id, e.name || e.id));
      }
    }
  }

  for (const [box, label] of [
    [discordDom, "Discords"],
    [telegramDom, "Telegrams"],
    [terminalDom, "Terminals"],
    [tempDom, "Temps"],
  ]) {
    if (!box) {
      continue;
    }
    const group = box.closest("details");
    const title = group ? group.querySelector("summary > p") : null;
    if (title) {
      title.textContent = `${label} (${box.childElementCount})`;
    }
  }

  const active = currentSessionId
    ? document.querySelector(`section.chats [data-id="${currentSessionId}"]`)
    : null;
  if (active) {
    const group = active.closest("details");
    if (group) {
      group.open = true;
    }
  }
}

function rowMenuItem(icon, label, style, action) {
  const props = { type: "button" };
  if (style) {
    props.class = style;
  }

  const dom = _("button", props, [_("span.material-symbols-outlined", icon), _("p", label)]);
  dom.addEventListener("click", function (e) {
    e.preventDefault();
    e.stopPropagation();
    closeChatMenu();
    action();
  });
  return dom;
}

function rowMenu(entries) {
  const menu = _("div.menu", entries);
  menu.dataset.show = "0";

  const more = _("button", { type: "button", class: "more" }, [_("span.material-symbols-outlined", "more_horiz")]);
  more.addEventListener("click", function (e) {
    e.preventDefault();
    e.stopPropagation();
    openChatMenu(menu);
  });

  return [more, menu];
}

function pinListItem(sessionId, title) {
  const body = [_("a", { href: getLink({ page: "chat", chat: sessionId }) }, title)];
  body.push(...rowMenu([rowMenuItem("keep_off", "Unpin", "", () => removePinChat(sessionId))]));

  return _(
    "div",
    {
      "data-id": sessionId,
      "data-selected": sessionId === currentSessionId ? 1 : 0,
    },
    body,
  );
}

function channelListItem(sessionId, title) {
  return _(
    "div",
    {
      "data-id": sessionId,
      "data-selected": sessionId === currentSessionId ? 1 : 0,
    },
    [
      _("a", { href: getLink({ page: "chat", chat: sessionId }) }, title),
      ...rowMenu([rowMenuItem("keep", "Pin", "", () => addPinChat(sessionId))]),
    ],
  );
}

function chatListItem(sessionId, title) {
  const entries = [
    rowMenuItem("keep", "Pin", "", () => addPinChat(sessionId)),
    rowMenuItem("delete", "Delete", "remove", function () {
      if (confirm(`Delete "${title}"?`)) {
        deleteChat(sessionId);
      }
    }),
  ];

  return _(
    "div",
    {
      "data-id": sessionId,
      "data-selected": sessionId === currentSessionId ? 1 : 0,
    },
    [_("a", { href: getLink({ page: "chat", chat: sessionId }) }, title), ...rowMenu(entries)],
  );
}

function closeChatMenu() {
  for (const dom of document.querySelectorAll('section.chats [data-show="1"]')) {
    dom.dataset.show = "0";
  }
}

function openChatMenu(menu) {
  closeChatMenu();
  menu.dataset.show = "1";
}

function bindChatMenu() {
  document.addEventListener("click", closeChatMenu);
  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape") {
      closeChatMenu();
    }
  });
}

async function deleteChat(sessionId) {
  try {
    const response = await fetch(`${API}/v1/session/${encodeURIComponent(sessionId)}`, {
      method: "DELETE",
    });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      alert(detail.error || `HTTP ${response.status}`);
      return;
    }
  } catch (err) {
    console.error("deleteChat", err);
    alert(err.message || "failed");
    return;
  }

  const unpinned = unpinChat(sessionId);

  const row = document.querySelector(`section.chats [data-id="${sessionId}"]`);
  if (row) {
    row.remove();
  }

  const open = new URL(window.location.href).searchParams.get("chat") || "";
  if (sessionId === currentSessionId || sessionId === open) {
    window.location.href = getLink({ page: "chat" });
    return;
  }
  if (unpinned) {
    window.location.reload();
  }
}

function renameChat(sessionId, title) {
  if (!sessionId || !title) {
    return;
  }

  const label = document.querySelector(`section.chats [data-id="${sessionId}"] > a`);
  if (label) {
    label.textContent = title;
  }
}

async function renderChat(sessionId) {
  const dom = chatMessages(sessionId);
  if (!dom || !sessionId) {
    return;
  }

  let content = "";
  try {
    const response = await fetch(`${API}/v1/session/${encodeURIComponent(sessionId)}?chat=1`);
    if (!response.ok) {
      return;
    }
    content = (await response.json()).chat || "";
  } catch (err) {
    console.error("loadChat", err);
    return;
  }

  for (const bubble of dom.querySelectorAll(":scope > div.user, :scope > div.assistant")) {
    bubble.remove();
  }

  const items = parseActionLog(content);
  clearTodo(sessionId);

  for (let i = 0; i < items.length; i++) {
    const item = items[i];
    if (item.pending && item.rule === "assistant" && i === items.length - 1) {
      setStream(sessionId, newStreamItem({ model: item.meta.model, trace: item.Reasoning, text: item.content }, sessionId));
      renderTodo(item.todos, sessionId);
      continue;
    }

    dom.appendChild(item.rule === "user" ? newUserItem(item) : newAssisatantItem(item, sessionId));
  }
  scrollToBottom(true, sessionId);
}

function assistantFooter(meta) {
  const children = [copyBtn(), knowledgeBtn()];
  if (meta.canceled) {
    children.push(_("p.canceled", "canceled"));
  }
  if (meta.duration) {
    children.push(_("p", meta.duration));
  }
  if (meta.input) {
    children.push(_("div", [_("span.material-symbols-outlined", "arrow_upward_alt"), _("p", meta.input)]));
  }
  if (meta.output) {
    children.push(_("div", [_("span.material-symbols-outlined", "arrow_downward_alt"), _("p", meta.output)]));
  }
  children.push(_("p", meta.send_at || ""));
  return _("footer", children);
}

function newUserItem(item) {
  const dom = _("div.user", [
    _("p", item.content),
    sourceBox(item.content),
    _("footer", [_("p", item.meta.send_at), copyBtn()]),
  ]);

  if (item.steered) {
    dom.dataset.steered = "1";
  }
  return dom;
}

function newAssisatantItem(item, sessionId) {
  const body = [_("p", item.meta.model || "")];

  if (item.Reasoning) {
    body.push(
      _("details.reasoning", [
        _("summary", ["Reasoning", _("span.material-symbols-outlined", "keyboard_arrow_down")]),
        _("section.md-render", renderMarkdownHTML(channelText(item.Reasoning, sessionId))),
      ]),
    );
  }

  if (item.content) {
    body.push(_("section.md-render", renderMarkdownHTML(channelText(item.content, sessionId))));
  }

  body.push(sourceBox(item.content));
  body.push(fileBox(item.files || []));
  body.push(assistantFooter(item.meta));

  return _("div.assistant", [_("img", { src: "public/logo-min.svg" }), _("section", body)]);
}
