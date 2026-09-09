const HINT_INTERVAL = 1000;

async function serverAlive() {
  try {
    const response = await fetch(`${API}/v1/info/version`, { cache: "no-store" });
    return response.ok;
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
