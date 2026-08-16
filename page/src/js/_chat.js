async function renderChatList() {
  const dom = $("#left-tab-chat-list");
  if (!dom) {
    return;
  }

  let list = [];
  try {
    const response = await fetch(`${API}/v1/sessions`);
    list = (await response.json()).sessions || [];
  } catch (err) {
    console.error("loadChatList", err);
    return;
  }

  dom.innerHTML = "";
  for (const e of list) {
    if (!e.id.startsWith("chat-")) {
      continue;
    }
    dom.appendChild(chatListItem(e.id, e.name || e.id));
  }
}

function chatListItem(sessionId, title) {
  const remove = _("button", { type: "button", class: "remove" }, [
    _("span.material-symbols-outlined", "delete"),
    _("p", "Delete"),
  ]);
  remove.addEventListener("click", function (e) {
    e.preventDefault();
    e.stopPropagation();
    closeChatMenu();
    if (confirm(`Delete "${title}"?`)) {
      deleteChat(sessionId);
    }
  });

  const menu = _("div.menu", [remove]);
  menu.dataset.show = "0";

  const more = _("button", { type: "button", class: "more" }, [_("span.material-symbols-outlined", "more_horiz")]);
  more.addEventListener("click", function (e) {
    e.preventDefault();
    e.stopPropagation();
    openChatMenu(menu);
  });

  return _(
    "div",
    {
      "data-id": sessionId,
      "data-selected": sessionId === currentSessionId ? 1 : 0,
    },
    [_("a", { href: getLink({ page: "chat", chat: sessionId }) }, title), more, menu],
  );
}

function closeChatMenu() {
  for (const dom of document.querySelectorAll('#left-tab-chat-list [data-show="1"]')) {
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
    const response = await fetch(`${API}/v1/session`, {
      method: "DELETE",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ session_id: sessionId }),
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

  const row = document.querySelector(`#left-tab-chat-list [data-id="${sessionId}"]`);
  if (row) {
    row.remove();
  }

  const open = new URL(window.location.href).searchParams.get("chat") || "";
  if (sessionId === currentSessionId || sessionId === open) {
    window.location.href = getLink({ page: "chat" });
  }
}

function renameChat(sessionId, title) {
  if (!sessionId || !title) {
    return;
  }

  const label = document.querySelector(`#left-tab-chat-list [data-id="${sessionId}"] > a`);
  if (label) {
    label.textContent = title;
  }
}

async function renderChat(sessionId) {
  const dom = $("#right-content-chat-messages");
  if (!dom || !sessionId) {
    return;
  }

  let content = "";
  try {
    const response = await fetch(`${API}/v1/session/${encodeURIComponent(sessionId)}/action`);
    if (!response.ok) {
      return;
    }
    content = (await response.json()).content || "";
  } catch (err) {
    console.error("loadChat", err);
    return;
  }

  for (const bubble of dom.querySelectorAll(":scope > div.user, :scope > div.assistant")) {
    bubble.remove();
  }

  const items = parseActionLog(content);
  clearTodo();

  for (let i = 0; i < items.length; i++) {
    const item = items[i];
    if (item.pending && item.rule === "assistant" && i === items.length - 1) {
      streamDom = newStreamItem({ model: item.meta.model, trace: item.Reasoning, text: item.content });
      renderTodo(item.todos);
      continue;
    }

    dom.appendChild(item.rule === "user" ? newUserItem(item) : newAssisatantItem(item));
  }
  scrollToBottom(true);
}

function assistantFooter(meta) {
  const children = [copyBtn()];
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

function newAssisatantItem(item) {
  const body = [_("p", item.meta.model || "")];

  if (item.Reasoning) {
    body.push(
      _("details", [
        _("summary", ["Reasoning", _("span.material-symbols-outlined", "keyboard_arrow_down")]),
        _("section.md-render", renderMarkdownHTML(item.Reasoning)),
      ]),
    );
  }

  if (item.content) {
    body.push(_("section.md-render", renderMarkdownHTML(item.content)));
  }

  body.push(sourceBox(item.content));
  body.push(assistantFooter(item.meta));

  return _("div.assistant", [_("img", { src: "public/logo-min.svg" }), _("section", body)]);
}
