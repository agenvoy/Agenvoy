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

function parseEvent(ev) {
  if (eventSkip(ev)) {
    return;
  }

  if (!streamDom) {
    if (ev.type === "EventDone") return;
    streamDom = newStreamItem();
  }

  renderEvent(streamDom, ev);

  if (ev.type === "EventDone") streamDom = null;
}
