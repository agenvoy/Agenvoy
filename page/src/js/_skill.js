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

let skillTabName = "";
let skillTabPath = "";

function skillTabDom() {
  return {
    form: $("#skill-form"),
    list: $("#skill-list"),
    name: $("#skill-name"),
    content: $("#skill-content"),
    allowList: $("#skill-allow-list"),
    remove: document.querySelector("#skill-form button.remove"),
  };
}

function resetSkillTab() {
  const dom = skillTabDom();
  skillTabName = "";
  skillTabPath = "";
  if (dom.name) dom.name.value = "";
  if (dom.content) dom.content.value = "";
  if (dom.form) {
    delete dom.form.dataset.editing;
    delete dom.form.dataset.view;
  }
  if (dom.allowList) delete dom.allowList.dataset.open;
  markSelectedCard(dom.list, "");
}

async function skillTabState() {
  const out = { skills: [], allowed: {}, source: {} };
  try {
    const response = await fetch(`${API}/v1/allowlist/skill?scope=global`);
    if (response.ok) {
      const body = await response.json();
      out.skills = body.skills || [];
      out.allowed = body.allowed || {};
      out.source = body.source || {};
    }
  } catch (err) {
    console.error("skillTabState", err);
  }
  return out;
}

async function renderSkillTab() {
  const dom = skillTabDom();
  if (!dom.list) {
    return;
  }

  const { skills, allowed, source } = await skillTabState();

  dom.list.innerHTML = "";

  for (const name of skills) {
    const mark = allowed[name] ? "always allow" : "ask each time";
    const card = _("div.card", [_("strong", name), _("p", `${source[name] || "unknown"} · ${mark}`)]);
    card.dataset.name = name;
    card.dataset.selected = name === skillTabName ? "1" : "0";
    card.addEventListener("click", () => {
      markSelectedCard(dom.list, name);
      openSkillTab(name);
    });
    dom.list.appendChild(card);
  }
}

async function openSkillTab(name) {
  const dom = skillTabDom();
  if (!dom.form) {
    return;
  }

  let body = null;
  try {
    const response = await fetch(`${API}/v1/skill/${encodeURIComponent(name)}`);
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      alert(detail.error || `HTTP ${response.status}`);
      return;
    }
    body = await response.json();
  } catch (err) {
    console.error("openSkillTab", err);
    alert(err.message || "failed");
    return;
  }

  skillTabName = body.name || name;
  skillTabPath = body.path || "";
  dom.name.value = body.source ? `${skillTabName} · ${body.source}` : skillTabName;
  dom.content.value = body.content || "";
  dom.form.dataset.editing = "1";
  delete dom.form.dataset.view;
  delete dom.allowList.dataset.open;
  dom.remove.style.display = body.deletable ? "" : "none";
}

async function openSkillConfig() {
  const dom = skillTabDom();
  if (!dom.form) {
    return;
  }

  skillTabName = "";
  skillTabPath = "";
  delete dom.form.dataset.editing;
  dom.form.dataset.view = "config";
  markSelectedCard(dom.list, "");
  renderSkillAllowList();
}

async function renderSkillAllowList() {
  const dom = skillTabDom();
  if (!dom.allowList) {
    return;
  }

  const { skills, allowed } = await skillTabState();

  dom.allowList.innerHTML = "";
  dom.allowList.dataset.open = "1";
  dom.allowList.appendChild(_("strong", "Always allow · pick the skills that skip the confirmation prompt"));

  for (const name of skills) {
    const box = _("input", { type: "checkbox" });
    box.checked = Boolean(allowed[name]);
    box.addEventListener("change", () => toggleSkillAllow(name));
    dom.allowList.appendChild(_("label.tool", [box, _("p", name)]));
  }
}

async function toggleSkillAllow(name) {
  if (!name) {
    return;
  }

  try {
    const response = await fetch(`${API}/v1/allowlist/skill`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ scope: "global", name: name }),
    });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      alert(detail.error || `HTTP ${response.status}`);
      return;
    }
  } catch (err) {
    console.error("toggleSkillAllow", err);
    alert(err.message || "failed");
    return;
  }
  renderSkillTab();
}

async function deleteSkillTab() {
  if (!skillTabName || !confirm(`Delete "${skillTabName}"?`)) {
    return;
  }

  try {
    const response = await fetch(`${API}/v1/skill?name=${encodeURIComponent(skillTabName)}`, { method: "DELETE" });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      alert(detail.error || `HTTP ${response.status}`);
      return;
    }
  } catch (err) {
    console.error("deleteSkillTab", err);
    alert(err.message || "failed");
    return;
  }

  resetSkillTab();
  renderSkillTab();
}

async function openSkillFolder() {
  const at = skillTabPath.lastIndexOf("/");
  if (at < 1) {
    return;
  }

  try {
    const response = await fetch(`${API}/v1/file/open?path=${encodeURIComponent(skillTabPath.slice(0, at))}`);
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      alert(detail.error || `HTTP ${response.status}`);
    }
  } catch (err) {
    console.error("openSkillFolder", err);
    alert(err.message || "failed");
  }
}
