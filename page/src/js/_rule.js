let rule = "";

async function getRuleList() {
  const dom = $("#chat-rule");
  if (!dom) {
    return;
  }

  let rules = [];
  try {
    const response = await fetch(`${API}/v1/rules`);
    if (response.ok) {
      rules = (await response.json()).rules || [];
    }
  } catch (err) {
    console.error("getRuleList", err);
  }

  dom.innerHTML = "";
  dom.appendChild(_("option", { value: "" }, "default"));
  for (const item of rules) {
    if (!item.name) continue;
    dom.appendChild(_("option", { value: item.name }, item.name));
  }

  const saved = readChatConfig(currentSessionId).rule;
  dom.value = saved;
  await selectRule(dom.value);
}

async function selectRule(name) {
  const dom = $("#chat-rule");
  if (!name) {
    rule = "";
    writeChatConfig(currentSessionId, { rule: "" });
    return;
  }

  let content = "";
  try {
    const response = await fetch(`${API}/v1/rule/${encodeURIComponent(name)}`);
    if (response.ok) {
      content = (await response.json()).content || "";
    }
  } catch (err) {
    console.error("selectRule", err);
  }

  if (!content) {
    if (dom) dom.value = "";
    rule = "";
    writeChatConfig(currentSessionId, { rule: "" });
    return;
  }

  rule = content;
  writeChatConfig(currentSessionId, { rule: name });
}

function saveChatRule(chatId) {
  const dom = $("#chat-rule");
  writeChatConfig(chatId, { rule: dom ? dom.value : "" });
}
