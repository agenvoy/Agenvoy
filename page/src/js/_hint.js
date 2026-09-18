const HINT_INTERVAL = 1000;
const UPDATE_INTERVAL = 1800000;

let daemonVersion = null;

async function serverAlive() {
  try {
    const response = await fetch(`${API}/v1/info/version`, { cache: "no-store" });
    if (!response.ok) {
      return false;
    }
    const version = ((await response.json().catch(() => ({}))) || {}).version;
    if (version && version !== daemonVersion) {
      if (daemonVersion !== null) {
        checkUpdate();
      }
      daemonVersion = version;
    }
    return true;
  } catch (err) {
    return false;
  }
}

function showHint(show) {
  const dom = $("section.hint");
  if (dom) {
    dom.dataset.hide = show ? "0" : "1";
  }
}

function watchServer() {
  const check = async function () {
    showHint(!(await serverAlive()));
  };

  check();
  setInterval(check, HINT_INTERVAL);
}
