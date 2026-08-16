function knowledgeSelected() {
  return readChatConfig(currentSessionId)
    .knowledge.split(",")
    .map((name) => name.trim())
    .filter((name) => name !== "");
}

async function openKnowledgePicker() {
  let items = [];
  try {
    const response = await fetch(`${API}/v1/knowledges`);
    if (response.ok) {
      items = (await response.json()).knowledges || [];
    }
  } catch (err) {
    console.error("openKnowledgePicker", err);
  }

  const picked = new Set(knowledgeSelected());
  const boxes = [];
  const list = _("div.list");

  if (items.length === 0) {
    list.appendChild(_("p.empty", "please generate first"));
  }
  for (const item of items) {
    if (!item.name) continue;
    const box = _("input", { type: "checkbox", value: item.name });
    box.checked = picked.has(item.name);
    boxes.push(box);

    list.appendChild(_("label", [box, _("div", [_("strong", item.name)])]));
  }

  const cancel = _("button", { type: "button" }, "cancel");
  const save = _("button", { type: "button", class: "submit" }, "save");

  const root = _("div.popup", [_("div.panel", [_("strong", "Knowledge"), list, _("footer", [cancel, save])])]);
  root.id = "knowledge-popup";

  const close = () => root.remove();
  cancel.addEventListener("click", close);
  root.addEventListener("click", (e) => {
    if (e.target === root) close();
  });
  save.addEventListener("click", () => {
    const names = boxes.filter((box) => box.checked).map((box) => box.value);
    writeChatConfig(currentSessionId, { knowledge: names.join(",") });
    markKnowledge(names.length);
    close();
  });

  document.body.appendChild(root);
}

function markKnowledge(count) {
  const dom = $("section.chat button.knowledge");
  if (!dom) {
    return;
  }
  if (count > 0) {
    dom.dataset.selected = "1";
    return;
  }
  delete dom.dataset.selected;
}

function renderKnowledgeMark() {
  markKnowledge(knowledgeSelected().length);
}
