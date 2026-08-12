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
