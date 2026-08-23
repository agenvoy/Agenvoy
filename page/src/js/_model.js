const CHAT_ID = /^(chat-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}|(tg|dc)-[0-9a-f]{64})$/i;

async function getModelList(sessionId) {
  const dom = $("#chat-model");
  if (!dom) {
    return;
  }

  let models = [];
  try {
    const response = await fetch(`${API}/v1/models`);
    if (!response.ok) {
      return;
    }
    models = (await response.json()).models || [];
  } catch (err) {
    console.error("loadModelList", err);
    return;
  }
  if (models.length === 0) {
    return;
  }

  dom.innerHTML = "";
  for (const name of models) {
    dom.appendChild(_("option", { value: name }, name));
  }
  dom.value = (await getSessionModel(sessionId)) || models[0];
}

function ensureModel() {
  const dom = $("#chat-model");
  return dom ? dom.value || "auto" : "auto";
}

async function saveSessionModel(sessionId, model) {
  if (!CHAT_ID.test(sessionId || "") || !model) {
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
  if (!CHAT_ID.test(sessionId || "")) {
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

function ensureReasoning() {
  const dom = $("#chat-reasoning");
  return dom ? dom.value || "medium" : "medium";
}

async function getReasoningList(sessionId) {
  const dom = $("#chat-reasoning");
  if (!dom || !CHAT_ID.test(sessionId || "")) {
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

    dom.innerHTML = "";
    for (const level of levels) {
      dom.appendChild(_("option", { value: level }, level));
    }
    dom.value = body.reasoning || levels[0];
  } catch (err) {
    console.error("getReasoningList", err);
  }
}

async function saveSessionReasoning(sessionId, level) {
  if (!CHAT_ID.test(sessionId || "") || !level) {
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
