async function openWorkDirPrompt() {
  const current = readChatConfig(currentSessionId).work_dir;
  const input = prompt("Working directory  (~/… or /…)", current);
  if (input === null) {
    return;
  }

  const path = input.trim();
  if (path === "") {
    writeChatConfig(currentSessionId, { work_dir: "" });
    markWorkDir("");
    return;
  }
  if (!path.startsWith("/") && !path.startsWith("~/")) {
    alert("Absolute path only: start with / or ~/");
    return;
  }

  let error = "";
  try {
    const response = await fetch(`${API}/v1/workdir?path=${encodeURIComponent(path)}`);
    if (!response.ok) {
      error = ((await response.json().catch(() => ({}))).error || `HTTP ${response.status}`).toString();
    }
  } catch (err) {
    error = err.message || "cannot reach the daemon";
  }
  if (error) {
    alert(error);
    return;
  }

  writeChatConfig(currentSessionId, { work_dir: path });
  markWorkDir(path);
}

function markWorkDir(path) {
  const dom = $("section.chat button.work-dir");
  if (!dom) {
    return;
  }
  if (path) {
    dom.dataset.selected = "1";
    dom.title = path;
    return;
  }
  delete dom.dataset.selected;
  dom.removeAttribute("title");
}

function renderWorkDirMark() {
  markWorkDir(readChatConfig(currentSessionId).work_dir);
}
