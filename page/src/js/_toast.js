const TOAST_MAX = 32;
const TOAST_LEVEL = { WARN: "warn", ERROR: "error" };

let toastStream = null;

function toastDom() {
  return $("#toast");
}

function pushToast(level, text, time) {
  const dom = toastDom();
  if (!dom || !text) {
    return;
  }

  const close = _("button", { type: "button" }, [_("span.material-symbols-outlined", "close")]);

  const item = _("div.item", [
    _("header", [_("span", time || ""), close]),
    _("p", text),
  ]);
  item.dataset.level = TOAST_LEVEL[level] || "info";
  close.addEventListener("click", () => item.remove());

  dom.appendChild(item);
  while (dom.children.length > TOAST_MAX) {
    dom.firstElementChild.remove();
  }
  dom.scrollTop = dom.scrollHeight;
}

function subscribeDaemonLog() {
  if (toastStream) {
    return;
  }

  toastStream = new EventSource(`${API}/v1/daemon/log`);
  toastStream.onmessage = function (e) {
    let event = {};
    try {
      event = JSON.parse(e.data);
    } catch (err) {
      console.error("subscribeDaemonLog", err);
      return;
    }
    if (!event.text || event.level === "DEBUG") {
      return;
    }
    pushToast(event.level || "INFO", event.text, event.time);
  };
  toastStream.onerror = function () {
    if (toastStream) {
      toastStream.close();
      toastStream = null;
    }
    setTimeout(subscribeDaemonLog, 5000);
  };
}
