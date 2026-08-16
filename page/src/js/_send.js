const API = "http://127.0.0.1:17989";
const SKIP_EVENTS = [
  "EventConnected",
  "EventTextDone",
  "EventAgentSelect",
  "EventSummaryGenerate",
  "EventToolCallStart",
  "EventToolCallText",
  "EventToolCallEnd",
  "EventToolResult",
  "EventUserInput",
  "EventUsageUpdate",
  "EventPending",
];
let currentSessionId = "";
let streamDom = null;

async function send(content) {
  content = (content || "").trim();
  if (content === "") return;

  let sessionId;
  try {
    sessionId = await ensureSessionId();
  } catch (err) {
    console.error("send", err);
    return;
  }

  const fresh = currentSessionId !== sessionId;
  const model = ensureModel();
  setSession(sessionId);
  subscribe(sessionId);
  if (fresh) {
    prependChat(sessionId, content);
    saveSessionModel(sessionId, model);
    adoptChatConfig(sessionId);
  }

  const chat = readChatConfig(sessionId);

  const picked = skill;
  clearSkill();

  const dom = $("#right-content-chat-messages");
  clearPending();
  if (streamDom) {
    appendUserText(dom, content);
  } else {
    dom.appendChild(newUserItem({ content: content, meta: { send_at: sendAt() } }));
    streamDom = newStreamItem();
    clearTodo();
  }
  scrollToBottom(true);

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
        system_prompt: rule,
        knowledge: chat.knowledge,
        work_dir: chat.work_dir,
        skill: picked,
      }),
    });
    if (!response.ok) {
      renderEvent(streamDom, { type: "EventError", text: `HTTP ${response.status}` });
    }
  } catch (err) {
    if (streamDom) {
      renderEvent(streamDom, { type: "EventError", text: err.message });
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

  const chat = $("section.chat");
  if (chat) chat.dataset.id = sessionId;

  const url = new URL(window.location.href);
  url.searchParams.set("page", "chat");
  url.searchParams.set("chat", sessionId);
  history.replaceState({}, "", url);
}

function scrollToBottom(force) {
  const dom = $("#right-content-chat-messages");
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

function eventSkip(event) {
  const type = event.type || "";
  if (event.source || (type === "EventToolCall" && event.tool_name === "write_todo")) {
    return true;
  }
  return SKIP_EVENTS.includes(type);
}

function newStreamItem(init) {
  init = init || {};

  const reasoning = _("section.md-render");
  const think = _("details", [
    _("summary", ["Reasoning", _("span.material-symbols-outlined", "keyboard_arrow_down")]),
    reasoning,
  ]);
  think.hidden = !init.trace;
  think.open = true;

  const model = _("p", init.model || "…");
  const answer = _("section.md-render");
  const source = sourceBox(init.text || "");
  const footer = _("footer");
  const body = _("section", [model, think, answer, source, footer]);
  const dom = _("div.assistant", [_("img", "public/logo-min.svg"), body]);

  $("#right-content-chat-messages").appendChild(dom);

  const view = {
    body: body,
    model: model,
    think: think,
    reasoning: reasoning,
    answer: answer,
    source: source,
    footer: footer,
    text: init.text || "",
    streamed: false,
    textStarted: Boolean(init.text),
    trace: init.trace || "",
  };
  if (view.trace) {
    render(view.reasoning, view.trace);
  }
  if (view.text) {
    render(view.answer, view.text);
  }
  return view;
}

function render(dom, markdown) {
  dom.innerHTML = renderMarkdownHTML(markdown);
  scrollToBottom();
}
