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

  if (typeof config.harness_enable !== "boolean") {
    config.harness_enable = false;
    writeConfig(config);
  }
  return config;
}

function readChatConfig(chatId) {
  const entry = (readConfig().chat || {})[chatId] || {};
  return {
    rule: typeof entry.rule === "string" ? entry.rule : "",
    knowledge: typeof entry.knowledge === "string" ? entry.knowledge : "",
    work_dir: typeof entry.work_dir === "string" ? entry.work_dir : "",
  };
}

function writeChatConfig(chatId, patch) {
  if (!chatId) {
    return;
  }
  const config = readConfig();
  config.chat = config.chat || {};
  config.chat[chatId] = Object.assign(readChatConfig(chatId), patch);
  writeConfig(config);
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

function droppedPaths(transfer) {
  const paths = [];
  const push = function (value) {
    const path = filePathOf(value);
    if (path && !paths.includes(path)) {
      paths.push(path);
    }
  };

  for (const line of (transfer.getData("text/uri-list") || "").split("\n")) {
    push(line);
  }
  if (paths.length === 0) {
    push(transfer.getData("text/plain"));
  }
  return paths;
}

function filePathOf(value) {
  value = (value || "").trim();
  if (value === "" || value.startsWith("#")) {
    return "";
  }
  if (value.startsWith("file://")) {
    try {
      return decodeURIComponent(new URL(value).pathname);
    } catch (err) {
      console.error("droppedPaths", err);
      return "";
    }
  }
  return value.startsWith("/") || value.startsWith("~/") ? value : "";
}

function bindInputDrop() {
  window.addEventListener("dragover", function (e) {
    e.preventDefault();
    if (e.dataTransfer) {
      e.dataTransfer.dropEffect = "copy";
    }
    const dom = $("#chat-input");
    if (dom) {
      dom.dataset.dragging = "1";
    }
  });

  for (const name of ["dragleave", "dragend", "drop"]) {
    window.addEventListener(name, function (e) {
      if (name === "dragleave" && e.relatedTarget !== null) {
        return;
      }
      const dom = $("#chat-input");
      if (dom) {
        delete dom.dataset.dragging;
      }
    });
  }

  window.addEventListener("drop", async function (e) {
    const dom = $("#chat-input");
    if (!dom) {
      return;
    }

    const paths = droppedPaths(e.dataTransfer);
    const files = paths.length === 0 ? Array.from((e.dataTransfer && e.dataTransfer.files) || []) : [];
    if (paths.length === 0 && files.length === 0) {
      return;
    }
    e.preventDefault();

    for (const file of files) {
      try {
        const found = await locateDropped(file);
        if (found.length === 0) {
          alert(`can not find where [${file.name}] lives on this machine`);
          continue;
        }
        const path = found.length === 1 ? found[0] : pickPath(file.name, found);
        if (path) {
          paths.push(path);
        }
      } catch (err) {
        console.error("locateDropped", err);
        alert(`failed to locate [${file.name}]：${err.message}`);
      }
    }
    if (paths.length > 0) {
      insertAtCursor(dom, paths.map((p) => `"${p}"`).join(" "));
    }
  });
}

async function locateDropped(file) {
  const query = new URLSearchParams({
    name: file.name,
    size: file.size,
    mtime: file.lastModified,
  });

  const res = await fetch(`${API}/v1/file/locate?${query}`);
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.error || res.status);
  }
  return data.paths || [];
}

function pickPath(name, list) {
  const menu = list.map((p, i) => `${i + 1}. ${p}`).join("\n");
  const answer = prompt(`${list.length} files match [${name}], enter a number:\n${menu}`, "1");
  return list[Number(answer) - 1] || "";
}

function insertAtCursor(dom, text) {
  const start = dom.selectionStart;
  const end = dom.selectionEnd;
  const before = dom.value.slice(0, start);
  const after = dom.value.slice(end);
  const lead = before === "" || before.endsWith(" ") || before.endsWith("\n") ? "" : " ";

  dom.value = before + lead + text + after;
  const caret = (before + lead + text).length;
  dom.setSelectionRange(caret, caret);
  dom.nextElementSibling.textContent = dom.value + "\n";
  dom.focus();
}
