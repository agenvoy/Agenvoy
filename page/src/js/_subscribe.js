const SESSION_CREATED = "session created";

let subscription = null;
let streamWasDown = false;
let subscribedSession = "";
let subscribeBound = false;

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
  if (subscription && subscribedSession === sessionId) {
    return;
  }

  closeSubscription();
  subscribedSession = sessionId;
  const url = sessionId
    ? `${API}/v1/log?sessions=${encodeURIComponent(sessionId)}&replay=0`
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

    if ((event.session && event.session !== subscribedSession) || event.type === "EventConnected") {
      return;
    }
    parseEvent(event);
  };
  subscription.onopen = () => {
    if (streamWasDown) {
      streamWasDown = false;
      renderChatList();
      if (subscribedSession) {
        loadPending(subscribedSession);
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

let currentTaskId = "";

function parseEvent(event) {
  if (event.task_id) {
    currentTaskId = event.task_id;
  }

  if (event.type === "EventTextDone") {
    clearTimeout(announceTimer);
    announceTimer = setTimeout(function () {
      announceTimer = 0;
      announced = true;
      voiceAnnounce();
    }, 300);
    return;
  }

  if (announceTimer && event.type !== "EventDone") {
    clearTimeout(announceTimer);
    announceTimer = 0;
  }

  if (event.type === "EventToolConfirm") {
    renderToolConfirm(event);
    return;
  }

  if (event.type === "EventUserInput") {
    appendInboundUser(event.text);
    return;
  }

  if (event.type === "EventPending") {
    loadPending(subscribedSession);
    return;
  }

  if (eventSkip(event)) {
    return;
  }

  if (event.type === "EventTodoUpdate") {
    renderTodo(event.todos || []);
    return;
  }

  if (!streamDom) {
    if (event.type === "EventDone" || event.type === "EventCanceled") return;
    streamDom = newStreamItem();
    announced = false;
  }

  renderEvent(streamDom, event);

  if (event.type === "EventCanceled") {
    currentTaskId = "";
    streamDom = null;
    clearTodo();
    loadPending(subscribedSession);
    clearTimeout(announceTimer);
    announceTimer = 0;
    announced = false;
    return;
  }

  if (event.type === "EventDone") {
    currentTaskId = "";
    streamDom = null;
    clearTodo();
    loadPending(subscribedSession);
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
