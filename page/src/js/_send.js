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
  setSession(sessionId);
  subscribe(sessionId);
  if (fresh) {
    prependChat(sessionId, content);
  }

  const dom = $("#right-content-chat-messages");
  dom.appendChild(newUserItem({ content: content, meta: { send_at: sendAt() } }));
  streamDom = newStreamItem();
  scrollToBottom(true);

  try {
    const response = await fetch(`${API}/v1/send`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ content: content, sse: false, session_id: sessionId, persist: true }),
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

async function ensureSessionId() {
  if (currentSessionId) {
    return currentSessionId;
  }

  const response = await fetch(`${API}/v1/session`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ prefix: "http-" }),
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

// TODO: 要補上 todo 區塊
function eventSkip(ev) {
  const type = ev.type || "";
  if (ev.source || (type === "EventToolCall" && ev.tool_name === "write_todo")) {
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
  const footer = _("footer");
  const body = _("section", [model, think, answer, footer]);
  const dom = _("div.assistant", [_("img", "public/logo-min.svg"), body]);

  $("#right-content-chat-messages").appendChild(dom);

  const view = {
    body: body,
    model: model,
    think: think,
    reasoning: reasoning,
    answer: answer,
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
