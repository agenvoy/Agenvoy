const CONFIG_KEY = "webui_config";
const AUTO_SCROLL_SLACK = 8;

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

const CHAT_DRAFT = "new";

function readChatConfig(chatId) {
  const entry = (readConfig().chat || {})[chatId || CHAT_DRAFT] || {};
  return {
    rule: typeof entry.rule === "string" ? entry.rule : "",
    knowledge: typeof entry.knowledge === "string" ? entry.knowledge : "",
    work_dir: typeof entry.work_dir === "string" ? entry.work_dir : "",
  };
}

function writeChatConfig(chatId, patch) {
  const key = chatId || CHAT_DRAFT;
  const config = readConfig();
  config.chat = config.chat || {};
  config.chat[key] = Object.assign(readChatConfig(key), patch);
  writeConfig(config);
}

function clearChatDraft() {
  const config = readConfig();
  if (!config.chat || !config.chat[CHAT_DRAFT]) {
    return;
  }
  delete config.chat[CHAT_DRAFT];
  writeConfig(config);
}

function adoptChatConfig(chatId) {
  const config = readConfig();
  const draft = (config.chat || {})[CHAT_DRAFT];
  if (!chatId || !draft) {
    return;
  }
  config.chat[chatId] = Object.assign(readChatConfig(chatId), draft);
  delete config.chat[CHAT_DRAFT];
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
    const entries = paths.length === 0 ? droppedEntries(e.dataTransfer) : [];
    if (paths.length === 0 && entries.length === 0) {
      return;
    }
    e.preventDefault();

    for (const entry of entries) {
      try {
        const found = await locateDropped(entry);
        if (found.length === 0) {
          alert(`can not find where [${entry.name}] lives on this machine`);
          continue;
        }
        const path = found.length === 1 ? found[0] : pickPath(entry.name, found);
        if (path) {
          paths.push(path);
        }
      } catch (err) {
        console.error("locateDropped", err);
        alert(`failed to locate [${entry.name}]：${err.message}`);
      }
    }
    if (paths.length > 0) {
      insertAtCursor(dom, paths.map((p) => `"${p}"`).join(" "));
    }
  });
}

function droppedEntries(transfer) {
  const out = [];
  for (const item of Array.from((transfer && transfer.items) || [])) {
    if (item.kind !== "file") {
      continue;
    }
    const entry = typeof item.webkitGetAsEntry === "function" ? item.webkitGetAsEntry() : null;
    if (entry && entry.isDirectory) {
      out.push({ name: entry.name, dir: true, entry: entry });
      continue;
    }
    const file = item.getAsFile();
    if (file) {
      out.push({ name: file.name, dir: false, file: file });
    }
  }
  if (out.length > 0) {
    return out;
  }
  return Array.from((transfer && transfer.files) || []).map((file) => ({ name: file.name, dir: false, file: file }));
}

const DIR_FINGERPRINT_MAX = 24;

function readEntryBatch(reader) {
  return new Promise(function (resolve) {
    reader.readEntries(resolve, function (err) {
      console.error("readEntries", err);
      resolve([]);
    });
  });
}

async function directoryChildren(entry) {
  if (!entry || typeof entry.createReader !== "function") {
    return [];
  }
  const reader = entry.createReader();
  const names = [];
  for (;;) {
    const batch = await readEntryBatch(reader);
    if (batch.length === 0) {
      return names;
    }
    for (const child of batch) {
      names.push(child.name);
      if (names.length >= DIR_FINGERPRINT_MAX) {
        return names;
      }
    }
  }
}

async function locateDropped(entry) {
  let query;
  if (entry.dir) {
    query = new URLSearchParams({ name: entry.name, dir: "1" });
    for (const child of await directoryChildren(entry.entry)) {
      query.append("child", child);
    }
  } else {
    query = new URLSearchParams({
      name: entry.file.name,
      size: entry.file.size,
      mtime: entry.file.lastModified,
    });
  }

  const res = await fetch(`${API}/v1/file/locate?${query}`);
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.error || res.status);
  }
  return data.paths || [];
}

function pickPath(name, list) {
  const menu = list.map((p, i) => `${i + 1}. ${p}`).join("\n");
  const answer = prompt(`${list.length} matches for [${name}], enter a number:\n${menu}`, "1");
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
