const TOAST_MAX = 32;
const TOAST_LEVEL = { WARN: "warn", ERROR: "error" };

function toastDom() {
  return $("#toast");
}

function pushToast(level, text, time) {
  const dom = toastDom();
  if (!dom || !text) {
    return;
  }

  const close = _("button", { type: "button" }, [_("span.material-symbols-outlined", "close")]);

  const item = _("div.item", [_("header", [_("span", time || ""), close]), _("p", text)]);
  item.dataset.level = TOAST_LEVEL[level] || "info";
  close.addEventListener("click", () => item.remove());

  dom.appendChild(item);
  while (dom.children.length > TOAST_MAX) {
    dom.firstElementChild.remove();
  }
  dom.scrollTop = dom.scrollHeight;
}

function daemonLogToast(level, text, time) {
  if (!text || level === "DEBUG") {
    return;
  }
  pushToast(level || "INFO", text, time || nowClock());
}

function nowClock() {
  const now = new Date();
  const pad = (n) => String(n).padStart(2, "0");
  return `${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}`;
}

