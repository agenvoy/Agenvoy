const SESSION_CREATED = "session created";

let subscription = null;
let streamWasDown = false;
let subscribedSession = "";
let subscribedSessions = [];
let subscribeBound = false;

function subscribeTargets(sessionId) {
  const list = [];
  const pinned = isWide() ? pinChats() : [];
  for (const id of [sessionId].concat(pinned)) {
    if (typeof id === "string" && id !== "" && !list.includes(id)) {
      list.push(id);
    }
  }
  return list;
}

function closeSubscription() {
  if (!subscription) {
    return;
  }
  subscription.close();
  subscription = null;
}

function bindSubscribeLifecycle() {
  if (subscribeBound) {
    return;
  }
  subscribeBound = true;

  window.addEventListener("pagehide", function () {
    closeSubscription();
  });

  window.addEventListener("pageshow", function (e) {
    if (!e.persisted || subscription) {
      return;
    }
    const sessionId = subscribedSession;
    subscribedSession = "";
    streamWasDown = true;
    subscribe(sessionId);
  });
}

function subscribe(sessionId) {
  bindSubscribeLifecycle();

  sessionId = sessionId || "";
  const targets = subscribeTargets(sessionId);
  if (subscription && subscribedSession === sessionId && subscribedSessions.join(",") === targets.join(",")) {
    return;
  }

  closeSubscription();
  subscribedSession = sessionId;
  subscribedSessions = targets;
  const url = targets.length
    ? `${API}/v1/log?sessions=${encodeURIComponent(targets.join(","))}&replay=0`
    : `${API}/v1/log?replay=0`;
  subscription = new EventSource(url);
  subscription.onmessage = (e) => {
    let event;
    try {
      event = JSON.parse(e.data);
    } catch (err) {
      console.error("subscribe", err);
      return;
    }

    if (event.type === "EventDaemonLog") {
      if (String(event.text || "").includes(SESSION_CREATED)) {
        renderChatList();
        appendDaemonLog(event.source, event.text);
        return;
      }
      if (!appendDaemonLog(event.source, event.text)) {
        daemonLogToast(event.source, event.text);
      }
      return;
    }

    if ((event.session && !subscribedSessions.includes(event.session)) || event.type === "EventConnected") {
      return;
    }
    parseEvent(event);
  };
  subscription.onopen = () => {
    if (streamWasDown) {
      streamWasDown = false;
      renderChatList();
      for (const id of subscribedSessions) {
        renderResumeMark(id);
      }
    }
  };
  subscription.onerror = (err) => {
    streamWasDown = true;
    console.error("subscribe", err);
  };
}

let announceTimer = 0;
let announced = false;

function parseEvent(event) {
  const sessionId = event.session || subscribedSession;
  const active = sessionId === currentSessionId;

  if (event.once_id) {
    setTask(sessionId, event.once_id);
  }

  if (event.type === "EventTextDone") {
    if (!active) {
      return;
    }
    clearTimeout(announceTimer);
    announceTimer = setTimeout(function () {
      announceTimer = 0;
      announced = true;
      voiceAnnounce();
    }, 300);
    return;
  }

  if (active && announceTimer && event.type !== "EventDone") {
    clearTimeout(announceTimer);
    announceTimer = 0;
  }

  if (event.type === "EventToolConfirm") {
    renderToolConfirm(event, sessionId);
    return;
  }

  if (event.type === "EventUserInput") {
    appendInboundUser(event.text, sessionId);
    return;
  }

  if (event.type === "EventPending") {
    renderResumeMark(sessionId);
    return;
  }

  if (eventSkip(event)) {
    return;
  }

  if (event.type === "EventTodoUpdate") {
    renderTodo(event.todos || [], sessionId);
    return;
  }

  let view = streamOf(sessionId);
  if (!view) {
    if (event.type === "EventDone" || event.type === "EventCanceled" || event.type === "EventError") return;
    if (!chatMessages(sessionId)) return;
    view = newStreamItem({}, sessionId);
    setStream(sessionId, view);
    if (active) {
      announced = false;
    }
  }

  renderEvent(view, event);

  if (event.type === "EventCanceled" || event.type === "EventError") {
    setTask(sessionId, "");
    setStream(sessionId, null);
    clearTodo(sessionId);
    clearPending(sessionId);
    renderResumeMark(sessionId);
    if (active) {
      clearTimeout(announceTimer);
      announceTimer = 0;
      announced = false;
    }
    return;
  }

  if (event.type === "EventDone") {
    setTask(sessionId, "");
    setStream(sessionId, null);
    clearTodo(sessionId);
    clearPending(sessionId);
    renderResumeMark(sessionId);
    if (!active) {
      return;
    }
    if (announceTimer) {
      clearTimeout(announceTimer);
      announceTimer = 0;
    }
    if (!announced) {
      voiceAnnounce();
    }
    announced = false;
  }
}
