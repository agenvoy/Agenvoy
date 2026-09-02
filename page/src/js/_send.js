const API = window.location.origin;
const SKIP_EVENTS = [
  "EventConnected",
  "EventTextDone",
  "EventAgentSelect",
  "EventSummaryGenerate",
  "EventToolCallStart",
  "EventToolCallText",
  "EventToolCallEnd",
  "EventToolResult",
  "EventUsageUpdate",
  "EventPending",
];
let currentSessionId = "";
const streamViews = new Map();
const taskIds = new Map();

function streamOf(sessionId) {
  return streamViews.get(sessionId || currentSessionId) || null;
}

function setStream(sessionId, view) {
  const id = sessionId || currentSessionId;
  if (!id) {
    return;
  }
  if (view) {
    streamViews.set(id, view);
  } else {
    streamViews.delete(id);
  }
}

function taskOf(sessionId) {
  return taskIds.get(sessionId || currentSessionId) || "";
}

function setTask(sessionId, taskId) {
  const id = sessionId || currentSessionId;
  if (!id) {
    return;
  }
  if (taskId) {
    taskIds.set(id, taskId);
  } else {
    taskIds.delete(id);
  }
}

const localEcho = new Map();

function pushEcho(sessionId, text) {
  const id = sessionId || currentSessionId;
  const list = localEcho.get(id) || [];
  list.push(text);
  localEcho.set(id, list);
}

function takeEcho(sessionId, text) {
  const list = localEcho.get(sessionId || currentSessionId);
  if (!list) {
    return false;
  }
  const i = list.indexOf(text);
  if (i === -1) {
    return false;
  }
  list.splice(i, 1);
  return true;
}

async function stopRunning(sessionId) {
  const sid = sessionId || currentSessionId;
  if (!sid) return;
  if (!confirm("Cancel this task?")) return;

  const taskId = taskOf(sid) || "current";
  try {
    const response = await fetch(
      `${API}/v1/session/${encodeURIComponent(sid)}/cancel/${encodeURIComponent(taskId)}`,
      { method: "POST" },
    );
    if (!response.ok) {
      pushToast(TOAST_LEVEL.ERROR, `cancel failed: ${response.status}`, nowClock());
    }
  } catch (err) {
    console.error("stopRunning", err);
    pushToast(TOAST_LEVEL.ERROR, `cancel failed: ${err?.message || err}`, nowClock());
  }
}

async function send(content, target) {
  content = (content || "").trim();
  if (content === "") return;

  target = (target || "").trim();
  const primary = target === "" || target === currentSessionId;

  let sessionId = target;
  if (primary) {
    try {
      sessionId = await ensureSessionId();
    } catch (err) {
      console.error("send", err);
      return;
    }

    const fresh = currentSessionId !== sessionId;
    setSession(sessionId);
    subscribe(sessionId);
    if (fresh) {
      prependChat(sessionId, content);
      saveSessionModel(sessionId, ensureModel());
      saveSessionReasoning(sessionId, ensureReasoning());
      adoptChatConfig(sessionId);
    }
  }

  const model = primary ? ensureModel() : "auto";
  const chat = readChatConfig(sessionId);

  const picked = primary ? skill : "";
  if (primary) {
    clearSkill();
  }

  const dom = chatMessages(sessionId);
  clearPending(sessionId);
  if (streamOf(sessionId)) {
    appendUserText(dom, content);
  } else {
    dom.appendChild(newUserItem({ content: content, meta: { send_at: sendAt() } }));
    setStream(sessionId, newStreamItem({}, sessionId));
    clearTodo(sessionId);
  }
  pushEcho(sessionId, content);
  scrollToBottom(true, sessionId);

  try {
    const response = await fetch(`${API}/v1/send`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        content: content,
        sse: false,
        session_id: sessionId,
        persist: true,
        model: model === "auto" ? "" : model,
        system_prompt: primary ? rule : "",
        work_dir: chat.work_dir,
        skill: picked,
      }),
    });
    if (!response.ok) {
      const view = streamOf(sessionId);
      if (view) {
        renderEvent(view, { type: "EventError", text: `HTTP ${response.status}` });
      }
    }
  } catch (err) {
    const view = streamOf(sessionId);
    if (view) {
      renderEvent(view, { type: "EventError", text: err.message });
    }
  }
}

function appendUserText(dom, content) {
  const bubbles = dom.querySelectorAll("div.user");
  const bubble = bubbles[bubbles.length - 1];
  if (!bubble) {
    dom.appendChild(newUserItem({ content: content, meta: { send_at: sendAt() } }));
    return;
  }

  const mark = steerMark(sendAt(), content, bubble.dataset.steered !== "1");
  bubble.dataset.steered = "1";
  const text = bubble.querySelector("p");
  if (text) {
    text.textContent += mark;
  }
  const source = bubble.querySelector("pre.source");
  if (source) {
    source.textContent += mark;
  }
}

async function ensureSessionId() {
  if (currentSessionId) {
    return currentSessionId;
  }

  const response = await fetch(`${API}/v1/session`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ prefix: "chat-" }),
  });
  const sessionId = (await response.json()).session_id || "";
  if (!sessionId) {
    throw new Error("session_id missing");
  }
  return sessionId;
}

function setSession(sessionId) {
  if (!sessionId || currentSessionId === sessionId) return;
  currentSessionId = sessionId;

  const panel = $("section.chat > main");
  if (panel) {
    panel.dataset.id = sessionId;
  }

  const url = new URL(window.location.href);
  url.searchParams.set("page", "chat");
  url.searchParams.set("chat", sessionId);
  history.replaceState({}, "", url);
}

function atBottom(sessionId) {
  const dom = chatMessages(sessionId);
  if (!dom) {
    return false;
  }
  return dom.scrollHeight - dom.scrollTop - dom.clientHeight <= AUTO_SCROLL_SLACK;
}

function scrollToBottom(force, sessionId) {
  const dom = chatMessages(sessionId);
  if (!dom || dom.scrollHeight < dom.clientHeight) {
    return;
  }

  if (!force) {
    const distance = dom.scrollHeight - dom.scrollTop - dom.clientHeight;
    if (distance > AUTO_SCROLL_SLACK) return;
  }

  dom.scrollTop = dom.scrollHeight;
  requestAnimationFrame(() => {
    dom.scrollTop = dom.scrollHeight;
  });
}

function prependChat(sessionID, label) {
  const dom = $("#left-tab-chat-list");
  if (!dom || !sessionID || dom.querySelector(`[data-id="${sessionID}"]`)) {
    return;
  }

  for (const selected of dom.querySelectorAll("[data-selected]")) {
    delete selected.dataset.selected;
  }
  dom.prepend(chatListItem(sessionID, label || sessionID));
}

function sendAt() {
  const date = new Date();
  const pad = (n) => String(n).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function appendInboundUser(text, sessionId) {
  text = (text || "").trim();
  if (!text || text.startsWith("[Resumed Task")) {
    return;
  }
  if (takeEcho(sessionId, text)) {
    return;
  }

  const dom = chatMessages(sessionId);
  if (!dom) {
    return;
  }
  setStream(sessionId, null);
  clearTodo(sessionId);
  dom.appendChild(newUserItem({ content: text, meta: { send_at: sendAt() } }));
  scrollToBottom(true, sessionId);
}

function eventSkip(event) {
  const type = event.type || "";
  if (event.source || (type === "EventToolCall" && event.tool_name === "write_todo")) {
    return true;
  }
  return SKIP_EVENTS.includes(type);
}

function newStreamItem(init, sessionId) {
  init = init || {};
  const sid = sessionId || currentSessionId;

  const reasoning = _("section.md-render");
  const think = _("details.reasoning", [
    _("summary", ["Reasoning", _("span.material-symbols-outlined", "keyboard_arrow_down")]),
    reasoning,
  ]);
  think.hidden = !init.trace;
  think.open = true;

  const model = _("p", init.model || "…");
  const answer = _("section.md-render");
  const source = sourceBox(init.text || "");
  const files = fileBox([]);
  const footer = _("footer");
  const stop = _("button.stop", { type: "button" }, [_("span.material-symbols-outlined", "stop"), _("p", "cancel")]);
  stop.addEventListener("click", () => stopRunning(sid));
  const body = _("section", [model, think, answer, source, files, footer, stop]);
  const dom = _("div.assistant", [_("img", "public/logo-min.svg"), body]);

  chatMessages(sid).appendChild(dom);

  const view = {
    session: sid,
    body: body,
    model: model,
    think: think,
    reasoning: reasoning,
    answer: answer,
    source: source,
    files: files,
    footer: footer,
    stop: stop,
    text: init.text || "",
    answered: false,
    streamed: false,
    textStarted: Boolean(init.text),
    trace: init.trace || "",
  };
  if (view.trace) {
    render(view.reasoning, view.trace, sid);
  }
  if (view.text) {
    render(view.answer, view.text, sid);
  }
  return view;
}

function render(dom, markdown, sessionId) {
  const stick = atBottom(sessionId);
  dom.innerHTML = renderMarkdownHTML(channelText(markdown));
  if (stick) {
    scrollToBottom(true, sessionId);
  }
}
