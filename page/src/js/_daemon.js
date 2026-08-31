const DAEMON_MAX_LINES = 2000;
const DAEMON_LEVELS = ["info", "debug", "warn", "error"];
const DAEMON_SLOG = /\blevel=([A-Z]+)/;
const DAEMON_STDLOG = /^\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2} ([A-Z]+)\b/;
const DAEMON_SCOPE = ["keyword", "from", "to"];
const DAEMON_UNIT = { m: 60000, h: 3600000, d: 86400000, w: 604800000 };
const DAEMON_SPAN = /^(\d+)([mhdw])$/;

function daemonDom() {
  return $("#daemon-log");
}

function daemonValue(id) {
  const dom = $(id);
  return dom ? dom.value.trim() : "";
}

function daemonDuration(text) {
  const matched = DAEMON_SPAN.exec(text.toLowerCase());
  if (!matched) {
    return 0;
  }
  return Number(matched[1]) * DAEMON_UNIT[matched[2]];
}

function daemonStamp(at) {
  const pad = (n) => String(n).padStart(2, "0");
  return [
    at.getFullYear(),
    pad(at.getMonth() + 1),
    pad(at.getDate()),
    pad(at.getHours()),
    pad(at.getMinutes()),
  ].join("-");
}

function daemonScope() {
  const params = praseURL();
  const query = new URLSearchParams();
  for (const key of DAEMON_SCOPE) {
    if (params[key]) {
      query.set(key, params[key]);
    }
  }
  const text = query.toString();
  return text ? `?${text}` : "";
}

function daemonSubmit() {
  const params = praseURL();
  const query = new URLSearchParams({ page: "monitor", tab: "Daemon" });
  if (params.target) {
    query.set("target", params.target);
  }

  const keyword = daemonValue("#daemon-keyword");
  if (keyword) {
    query.set("keyword", keyword);
  }

  const agoText = daemonValue("#daemon-ago");
  if (agoText) {
    const ago = daemonDuration(agoText);
    if (!ago) {
      pushToast("WARN", `"${agoText}" is not a duration · use 30m / 12h / 2d / 1w`, nowClock());
      return;
    }

    let span = ago;
    const spanText = daemonValue("#daemon-span");
    if (spanText) {
      span = daemonDuration(spanText);
      if (!span) {
        pushToast("WARN", `"${spanText}" is not a duration · use 30m / 12h / 2d / 1w`, nowClock());
        return;
      }
      if (span > ago) {
        pushToast("WARN", `window ${spanText} is wider than ${agoText} ago · it would run past now`, nowClock());
        return;
      }
    }

    const from = new Date(Date.now() - ago);
    query.set("from", daemonStamp(from));
    query.set("to", daemonStamp(new Date(from.getTime() + span)));
  }

  window.location.href = `?${query.toString()}`;
}

function daemonLevel(text) {
  const matched = DAEMON_SLOG.exec(text) || DAEMON_STDLOG.exec(text);
  if (!matched) {
    return "";
  }
  const level = matched[1].toLowerCase();
  return DAEMON_LEVELS.includes(level) ? level : "";
}

function daemonLine(text) {
  const line = _("span");
  line.textContent = text + "\n";
  line.dataset.level = daemonLevel(text);
  return line;
}

function trimDaemonLog(dom) {
  while (dom.childElementCount > DAEMON_MAX_LINES) {
    dom.firstElementChild.remove();
  }
}

async function renderDaemonPage() {
  const dom = daemonDom();
  if (!dom) {
    return;
  }

  const params = praseURL();
  const panel = $("#daemon-panel");
  const target = DAEMON_LEVELS.includes(params.target) ? params.target : "all";
  if (panel) {
    panel.dataset.target = target;
    for (const card of panel.querySelectorAll("div.side > a.new, div.list > a.card")) {
      card.dataset.selected = card.getAttribute("name") === target ? "1" : "0";
    }
  }

  const keyword = $("#daemon-keyword");
  if (keyword && params.keyword) {
    keyword.value = params.keyword;
  }

  let content = "";
  try {
    const response = await fetch(`${API}/v1/daemon${daemonScope()}`);
    if (response.ok) {
      content = (await response.json()).content || "";
    }
  } catch (err) {
    console.error("renderDaemonPage", err);
  }

  dom.innerHTML = "";
  content = content.replace(/\n+$/, "");
  if (content) {
    for (const line of content.split("\n")) {
      dom.appendChild(daemonLine(line));
    }
  }
  trimDaemonLog(dom);
  dom.scrollTop = dom.scrollHeight;
}

function appendDaemonLog(level, text, time) {
  const dom = daemonDom();
  if (!dom || !text) {
    return false;
  }
  if (daemonScope()) {
    return true;
  }

  const stick = dom.scrollHeight - dom.scrollTop - dom.clientHeight < AUTO_SCROLL_SLACK;
  const stamp = time || new Date().toISOString();
  dom.appendChild(daemonLine(`time=${stamp} level=${(level || "INFO").toUpperCase()} msg="${text}"`));
  trimDaemonLog(dom);
  if (stick) {
    dom.scrollTop = dom.scrollHeight;
  }
  return true;
}
