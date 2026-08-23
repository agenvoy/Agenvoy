let subscription = null;
let subscribedSession = "";
function subscribe(sessionId) {
  if (!sessionId || subscribedSession === sessionId) {
    return;
  }

  if (subscription) {
    subscription.close();
    subscribedSession = "";
    subscription = null;
  }
  subscribedSession = sessionId;
  subscription = new EventSource(`${API}/v1/log?sessions=${encodeURIComponent(sessionId)}&replay=0`);
  subscription.onmessage = (e) => {
    let event;
    try {
      event = JSON.parse(e.data);
    } catch (err) {
      console.error("subscribe", err);
      return;
    }

    if ((event.session && event.session !== subscribedSession) || event.type === "EventConnected") {
      return;
    }
    parseEvent(event);
  };
  subscription.onerror = (err) => {
    console.error("subscribe", err);
  };
}

let announceTimer = 0;
let announced = false;

function parseEvent(event) {
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
    streamDom = null;
    clearTodo();
    loadPending(subscribedSession);
    clearTimeout(announceTimer);
    announceTimer = 0;
    announced = false;
    return;
  }

  if (event.type === "EventDone") {
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
