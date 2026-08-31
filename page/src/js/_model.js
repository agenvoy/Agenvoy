const CHAT_ID = /^(chat-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}|(tg|dc)-[0-9a-f]{64})$/i;
const SESSION_ID = /^((chat|cli|temp)-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}|(tg|dc)-[0-9a-f]{64})$/i;

let modelList = [];
let currentModel = "auto";

function modelLabel(name) {
  return name.slice(name.indexOf("@") + 1) || name;
}

function ensureModel() {
  return currentModel || "auto";
}

function markModel(name) {
  const dom = $("#chat-model");
  if (!dom) {
    return;
  }
  const label = modelLabel(name);
  dom.name = label;
  const text = dom.querySelector("p");
  if (text) {
    text.textContent = label;
  }
}

async function getModelList(sessionId) {
  modelList = [];
  try {
    const response = await fetch(`${API}/v1/models`);
    if (!response.ok) {
      return;
    }
    modelList = (await response.json()).models || [];
  } catch (err) {
    console.error("loadModelList", err);
    return;
  }
  if (modelList.length === 0) {
    return;
  }

  currentModel = (await getSessionModel(sessionId)) || modelList[0];
  markModel(currentModel);
}

function selectModel(name) {
  if (!name) {
    return;
  }
  currentModel = name;
  markModel(name);
  saveSessionModel(currentSessionId, name);
}

function openModelPicker() {
  const names = modelList.length > 0 ? modelList : [ensureModel()];
  const list = _("div.list");

  const cancel = _("button", { type: "button" }, "cancel");
  const root = _("div.popup", [_("div.panel", [_("strong", "Model"), list, _("footer", [cancel])])]);
  root.id = "model-popup";

  const close = () => root.remove();

  for (const name of names) {
    const box = _("input", { type: "radio", name: "model-pick", value: name });
    box.checked = currentModel === name;
    box.addEventListener("change", () => {
      selectModel(name);
      close();
    });
    list.appendChild(_("label", [box, _("div", [_("strong", modelLabel(name))])]));
  }

  cancel.addEventListener("click", close);
  root.addEventListener("click", (e) => {
    if (e.target === root) close();
  });

  document.body.appendChild(root);
}

async function saveSessionModel(sessionId, model) {
  if (!SESSION_ID.test(sessionId || "") || !model) {
    return;
  }

  try {
    const response = await fetch(`${API}/v1/session/${encodeURIComponent(sessionId)}/model`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ model: model }),
    });
    if (!response.ok) {
      console.error("saveSessionModel", response.status);
    }
  } catch (err) {
    console.error("saveSessionModel", err);
  }
}

async function getSessionModel(sessionId) {
  if (!SESSION_ID.test(sessionId || "")) {
    return "";
  }

  try {
    const response = await fetch(`${API}/v1/sessions`);
    if (!response.ok) {
      return "";
    }

    const list = (await response.json()).sessions || [];
    const session = list.find((item) => item.id === sessionId);
    return session ? session.model || "" : "";
  } catch (err) {
    console.error("sessionModel", err);
    return "";
  }
}

const REASONING_ICON = {
  none: "signal_cellular_0_bar",
  low: "signal_cellular_1_bar",
  medium: "signal_cellular_2_bar",
  high: "signal_cellular_3_bar",
  xhigh: "network_cell",
  max: "signal_cellular_4_bar",
};

let reasoningLevel = "medium";
let reasoningList = [];

function ensureReasoning() {
  return reasoningLevel || "medium";
}

function markReasoning(level) {
  const dom = $("#chat-reasoning");
  if (!dom) {
    return;
  }
  dom.name = level;
  const icon = dom.querySelector("span");
  if (icon) {
    icon.textContent = REASONING_ICON[level] || REASONING_ICON.medium;
  }
}

async function getReasoningList(sessionId) {
  if (!SESSION_ID.test(sessionId || "")) {
    return;
  }

  try {
    const response = await fetch(`${API}/v1/session/${encodeURIComponent(sessionId)}/reasoning`);
    if (!response.ok) {
      return;
    }
    const body = await response.json();
    const levels = body.levels || [];
    if (levels.length === 0) {
      return;
    }

    reasoningList = levels;
    reasoningLevel = body.reasoning || levels[0];
    markReasoning(reasoningLevel);
  } catch (err) {
    console.error("getReasoningList", err);
  }
}

function selectReasoning(level) {
  if (!level) {
    return;
  }
  reasoningLevel = level;
  markReasoning(level);
  saveSessionReasoning(currentSessionId, level);
}

function openReasoningPicker() {
  const levels = reasoningList.length > 0 ? reasoningList : Object.keys(REASONING_ICON);
  const list = _("div.list");

  const cancel = _("button", { type: "button" }, "cancel");
  const root = _("div.popup", [_("div.panel", [_("strong", "Reasoning"), list, _("footer", [cancel])])]);
  root.id = "reasoning-popup";

  const close = () => root.remove();

  for (const level of levels) {
    const box = _("input", { type: "radio", name: "reasoning-pick", value: level });
    box.checked = reasoningLevel === level;
    box.addEventListener("change", () => {
      selectReasoning(level);
      close();
    });
    list.appendChild(_("label", [box, _("div", [_("strong", level)])]));
  }

  cancel.addEventListener("click", close);
  root.addEventListener("click", (e) => {
    if (e.target === root) close();
  });

  document.body.appendChild(root);
}

async function saveSessionReasoning(sessionId, level) {
  if (!SESSION_ID.test(sessionId || "") || !level) {
    return;
  }

  try {
    const response = await fetch(`${API}/v1/session/${encodeURIComponent(sessionId)}/reasoning`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ reasoning: level }),
    });
    if (!response.ok) {
      console.error("saveSessionReasoning", response.status);
    }
  } catch (err) {
    console.error("saveSessionReasoning", err);
  }
}
