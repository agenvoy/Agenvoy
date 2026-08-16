let skill = "";

function clearSkill() {
  skill = "";
  markSkill("");
}

async function openSkillPicker() {
  let items = [];
  try {
    const response = await fetch(`${API}/v1/skills`);
    if (response.ok) {
      items = (await response.json()).skills || [];
    }
  } catch (err) {
    console.error("openSkillPicker", err);
  }

  const list = _("div.list");
  const boxes = [];

  const none = _("input", { type: "radio", name: "skill-pick", value: "" });
  none.checked = skill === "";
  boxes.push(none);
  list.appendChild(_("label", [none, _("div", [_("strong", "none")])]));

  for (const item of items) {
    if (!item.name) continue;
    const box = _("input", { type: "radio", name: "skill-pick", value: item.name });
    box.checked = skill === item.name;
    boxes.push(box);

    const body = item.description ? [_("strong", item.name), _("p", item.description)] : [_("strong", item.name)];
    list.appendChild(_("label", [box, _("div", body)]));
  }

  const cancel = _("button", { type: "button" }, "cancel");
  const save = _("button", { type: "button", class: "submit" }, "save");

  const root = _("div.popup", [_("div.panel", [_("strong", "Skill"), list, _("footer", [cancel, save])])]);
  root.id = "skill-popup";

  const close = () => root.remove();
  cancel.addEventListener("click", close);
  root.addEventListener("click", (e) => {
    if (e.target === root) close();
  });
  save.addEventListener("click", () => {
    const picked = boxes.find((box) => box.checked);
    skill = picked ? picked.value : "";
    markSkill(skill);
    close();
  });

  document.body.appendChild(root);
}

function markSkill(name) {
  const dom = $("section.chat button.skill");
  if (!dom) {
    return;
  }
  if (name) {
    dom.dataset.selected = "1";
    dom.title = name;
    return;
  }
  delete dom.dataset.selected;
  dom.removeAttribute("title");
}
