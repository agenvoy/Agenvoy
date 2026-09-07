const HINT_INTERVAL = 5000;

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

function paintHint(left) {
  const dom = $("#hint-retry");
  if (dom) {
    dom.textContent = `Retrying in ${left}s \u00b7 check that the daemon is running`;
  }
}

function watchServer() {
  const span = HINT_INTERVAL / 1000;
  let left = span;

  const check = async function () {
    showHint(!(await serverAlive()));
    left = span;
    paintHint(left);
  };

  check();
  setInterval(function () {
    left -= 1;
    if (left <= 0) {
      check();
      return;
    }
    paintHint(left);
  }, 1000);
}
