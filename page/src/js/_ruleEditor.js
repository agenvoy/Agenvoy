const RULE_TEMPLATE = `# Role

<one line: who this agent is and who it works for>

## Always

-

## Never

-

## Output

-
`;

let ruleEditing = "";

function ruleDom() {
  return {
    form: $("#rule-form"),
    list: $("#rule-list"),
    title: $("#rule-title"),
    content: $("#rule-content"),
    submit: document.querySelector("#rule-form button.submit"),
  };
}

function ruleError(text) {
  alert(text);
}

function markRuleEditing(dom, editing) {
  if (dom.form) {
    if (editing) {
      dom.form.dataset.editing = "1";
    } else {
      delete dom.form.dataset.editing;
    }
  }
  if (dom.submit) {
    dom.submit.textContent = editing ? "save" : "add";
  }
}

async function renderRule() {
  const dom = ruleDom();
  if (!dom.list) {
    return;
  }

  let items = [];
  try {
    const response = await fetch(`${API}/v1/rules`);
    if (response.ok) {
      items = (await response.json()).rules || [];
    }
  } catch (err) {
    console.error("renderRule", err);
  }

  dom.list.innerHTML = "";
  const picked = praseURL().target || "";
  let found = false;

  for (const item of items) {
    if (!item.name) continue;
    if (item.name === picked) {
      found = true;
    }

    const remove = _("button", { type: "button" }, [_("span.material-symbols-outlined", "delete")]);
    remove.addEventListener("click", (e) => {
      e.stopPropagation();
      deleteRule(item.name);
    });

    const card = _("div.card", [
      _("strong", item.name),
      _("p", item.updated_at ? ruleDate(item.updated_at) : ""),
      remove,
    ]);
    card.dataset.name = item.name;
    card.dataset.selected = item.name === picked ? "1" : "0";
    card.addEventListener("click", () => {
      window.location.href = getLink({ page: "features", tab: "Rules", target: item.name });
    });
    dom.list.appendChild(card);
  }

  if (found) {
    openRule(picked);
  }
}

function ruleDate(seconds) {
  const date = new Date(seconds * 1000);
  const pad = (n) => String(n).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

async function openRule(name) {
  const dom = ruleDom();
  if (!dom.title) {
    return;
  }

  try {
    const response = await fetch(`${API}/v1/rule/${encodeURIComponent(name)}`);
    if (!response.ok) {
      ruleError(`HTTP ${response.status}`);
      return;
    }
    const body = await response.json();
    dom.title.value = body.name || "";
    dom.content.value = body.content || "";
    ruleEditing = body.name || "";
    markRuleEditing(dom, true);
  } catch (err) {
    console.error("openRule", err);
    ruleError(err.message || "failed");
  }
}

function resetRule() {
  const dom = ruleDom();
  if (dom.title) dom.title.value = "";
  if (dom.content) dom.content.value = RULE_TEMPLATE;
  markRuleEditing(dom, false);
  ruleEditing = "";
  markSelectedCard(dom.list, "");
}

async function saveRule() {
  const dom = ruleDom();
  if (!dom.title) {
    return;
  }

  const name = dom.title.value.trim();
  if (!name) {
    ruleError("name is required");
    return;
  }

  const body = { name: name, content: dom.content ? dom.content.value : "" };

  let method = "POST";
  if (ruleEditing) {
    method = "PATCH";
    if (ruleEditing !== name) {
      body.rename = name;
      body.name = ruleEditing;
    }
  }

  try {
    const response = await fetch(`${API}/v1/rule`, {
      method: method,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      ruleError(detail.error || `HTTP ${response.status}`);
      return;
    }
    const saved = ((await response.json()) || {}).name || name;
    window.location.href = getLink({ page: "features", tab: "Rules", target: saved });
  } catch (err) {
    console.error("saveRule", err);
    ruleError(err.message || "failed");
  }
}

async function deleteRule(name) {
  if (!confirm(`Delete "${name}"?`)) {
    return;
  }

  try {
    const response = await fetch(`${API}/v1/rule?name=${encodeURIComponent(name)}`, { method: "DELETE" });
    if (!response.ok) {
      ruleError(`HTTP ${response.status}`);
      return;
    }
  } catch (err) {
    console.error("deleteRule", err);
    ruleError(err.message || "failed");
    return;
  }

  window.location.href = getLink({ page: "features", tab: "Rules" });
}

function deleteEditingRule() {
  if (ruleEditing) {
    deleteRule(ruleEditing);
  }
}
