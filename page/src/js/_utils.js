const CONFIG_KEY = "webui_config";
const AUTO_SCROLL_SLACK = 96;

function praseURL() {
  const url = new URL(window.location.href);
  const params = {};
  for (const [key, value] of url.searchParams) {
    params[key] = value;
  }
  return params;
}

function writeConfig(config) {
  try {
    localStorage.setItem(CONFIG_KEY, JSON.stringify(config));
  } catch (err) {
    console.error(err);
  }
}

function readConfig() {
  let config = null;

  try {
    config = JSON.parse(localStorage.getItem(CONFIG_KEY));
  } catch (err) {
    console.error(err);
    config = null;
  }

  if (typeof config !== "object" || config === null || Array.isArray(config)) {
    config = {};
  }

  if (config.left_tab_collapsed !== "1" && config.left_tab_collapsed !== "0") {
    config.left_tab_collapsed = document.documentElement.clientWidth < 768 ? "1" : "0";
    writeConfig(config);
  }
  return config;
}

function sourceBox(text) {
  return _("pre.source", { textContent: text || "" });
}

function copyBtn() {
  const dom = _("button", [_("span.material-symbols-outlined", "content_copy")]);
  dom.addEventListener("click", function () {
    const bubble = dom.closest("div.assistant, div.user");
    const source = bubble && bubble.querySelector("pre.source");
    if (!source || !navigator.clipboard) {
      return;
    }
    navigator.clipboard.writeText(source.textContent).catch((err) => console.error("copy", err));
  });
  return dom;
}

function bindSelectPicker() {
  document.addEventListener("click", function (e) {
    const label = e.target.closest("label:has(select)");
    if (!label || e.target.tagName === "SELECT") {
      return;
    }

    const dom = label.querySelector("select");
    if (dom && typeof dom.showPicker === "function") {
      dom.showPicker();
    }
  });
}
