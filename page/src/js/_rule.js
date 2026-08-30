let rule = "";
let ruleName = "";
let ruleList = [];

async function getRuleList() {
  ruleList = [];
  try {
    const response = await fetch(`${API}/v1/rules`);
    if (response.ok) {
      ruleList = (await response.json()).rules || [];
    }
  } catch (err) {
    console.error("getRuleList", err);
  }

  await selectRule(readChatConfig(currentSessionId).rule);
}

function markRule(name) {
  const dom = $("section.chat button.rule");
  if (!dom) {
    return;
  }
  if (name) {
    dom.dataset.selected = "1";
    dom.title = name;
    dom.name = name.slice(0, 16);
    return;
  }
  delete dom.dataset.selected;
  dom.removeAttribute("title");
  dom.name = "Use persona";
}

async function selectRule(name) {
  if (!name) {
    rule = "";
    ruleName = "";
    writeChatConfig(currentSessionId, { rule: "" });
    markRule("");
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
    rule = "";
    ruleName = "";
    writeChatConfig(currentSessionId, { rule: "" });
    markRule("");
    return;
  }

  rule = content;
  ruleName = name;
  writeChatConfig(currentSessionId, { rule: name });
  markRule(name);
}

function openRulePicker() {
  const list = _("div.list");
  const boxes = [];

  const none = _("input", { type: "radio", name: "rule-pick", value: "" });
  none.checked = ruleName === "";
  boxes.push(none);
  list.appendChild(_("label", [none, _("div", [_("strong", "default")])]));

  for (const item of ruleList) {
    if (!item.name) continue;
    const box = _("input", { type: "radio", name: "rule-pick", value: item.name });
    box.checked = ruleName === item.name;
    boxes.push(box);
    list.appendChild(_("label", [box, _("div", [_("strong", item.name)])]));
  }

  const cancel = _("button", { type: "button" }, "cancel");
  const save = _("button", { type: "button", class: "submit" }, "save");

  const root = _("div.popup", [_("div.panel", [_("strong", "Rule"), list, _("footer", [cancel, save])])]);
  root.id = "rule-popup";

  const close = () => root.remove();
  cancel.addEventListener("click", close);
  root.addEventListener("click", (e) => {
    if (e.target === root) close();
  });
  save.addEventListener("click", () => {
    const picked = boxes.find((box) => box.checked);
    selectRule(picked ? picked.value : "");
    close();
  });

  document.body.appendChild(root);
}
