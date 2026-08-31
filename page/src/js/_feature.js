const RULE_TEMPLATE = `# Role

<one line: who this agent is and who it works for>

## Always

-

## Never

-

## Output

-
`;

const KNOWLEDGE_TEMPLATE = `# <topic>

## Summary

<one or two lines on what this covers, so the agent can tell whether to read on>

## Details

-

## Source

-
`;

const FEATURE_SPEC = {
  rule: { list: "/v1/rules", item: "/v1/rule", key: "rules", tab: "Rules", template: RULE_TEMPLATE },
  knowledge: {
    list: "/v1/knowledges",
    item: "/v1/knowledge",
    key: "knowledges",
    tab: "Knowledge",
    template: KNOWLEDGE_TEMPLATE,
    titleOptional: true,
  },
};

const featureEditing = { rule: "", knowledge: "" };

function featureDom(kind) {
  return {
    form: $(`#${kind}-form`),
    list: $(`#${kind}-list`),
    title: $(`#${kind}-title`),
    content: $(`#${kind}-content`),
    submit: document.querySelector(`#${kind}-form button.submit`),
  };
}

function featureError(text) {
  alert(text);
}

function markFeatureMode(dom, editing) {
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

async function renderFeature(kind) {
  const spec = FEATURE_SPEC[kind];
  const dom = featureDom(kind);
  if (!spec || !dom.list) {
    return;
  }

  let items = [];
  try {
    const response = await fetch(`${API}${spec.list}`);
    if (response.ok) {
      items = (await response.json())[spec.key] || [];
    }
  } catch (err) {
    console.error("renderFeature", err);
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
      deleteFeature(kind, item.name);
    });

    const card = _("div.card", [
      _("strong", item.name),
      _("p", item.updated_at ? featureDate(item.updated_at) : ""),
      remove,
    ]);
    card.dataset.name = item.name;
    card.dataset.selected = item.name === picked ? "1" : "0";
    card.addEventListener("click", () => {
      window.location.href = getLink({ page: "features", tab: spec.tab, target: item.name });
    });
    dom.list.appendChild(card);
  }

  if (found) {
    openFeature(kind, picked);
  }
}

function featureDate(seconds) {
  const date = new Date(seconds * 1000);
  const pad = (n) => String(n).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

async function openFeature(kind, name) {
  const spec = FEATURE_SPEC[kind];
  const dom = featureDom(kind);
  if (!spec || !dom.title) {
    return;
  }

  try {
    const response = await fetch(`${API}${spec.item}/${encodeURIComponent(name)}`);
    if (!response.ok) {
      featureError(`HTTP ${response.status}`);
      return;
    }
    const body = await response.json();
    dom.title.value = body.name || "";
    dom.content.value = body.content || "";
    featureEditing[kind] = body.name || "";
    markFeatureMode(dom, true);
  } catch (err) {
    console.error("openFeature", err);
    featureError(err.message || "failed");
  }
}

function resetFeature(kind) {
  const spec = FEATURE_SPEC[kind];
  const dom = featureDom(kind);
  if (dom.title) dom.title.value = "";
  if (dom.content) dom.content.value = spec ? spec.template : "";
  markFeatureMode(dom, false);
  featureEditing[kind] = "";
  markSelectedCard(dom.list, "");
}

async function saveFeature(kind) {
  const spec = FEATURE_SPEC[kind];
  const dom = featureDom(kind);
  if (!spec || !dom.title) {
    return;
  }

  const name = dom.title.value.trim();
  if (!name && !spec.titleOptional) {
    featureError("name is required");
    return;
  }

  const body = { name: name, content: dom.content ? dom.content.value : "" };

  let method = "POST";
  if (featureEditing[kind]) {
    method = "PATCH";
    if (featureEditing[kind] !== name) {
      body.rename = name;
      body.name = featureEditing[kind];
    }
  }

  try {
    const response = await fetch(`${API}${spec.item}`, {
      method: method,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      featureError(detail.error || `HTTP ${response.status}`);
      return;
    }
    const saved = ((await response.json()) || {}).name || name;
    window.location.href = getLink({ page: "features", tab: spec.tab, target: saved });
  } catch (err) {
    console.error("saveFeature", err);
    featureError(err.message || "failed");
  }
}

async function deleteFeature(kind, name) {
  const spec = FEATURE_SPEC[kind];
  if (!spec || !confirm(`Delete "${name}"?`)) {
    return;
  }

  try {
    const response = await fetch(`${API}${spec.item}?name=${encodeURIComponent(name)}`, { method: "DELETE" });
    if (!response.ok) {
      featureError(`HTTP ${response.status}`);
      return;
    }
  } catch (err) {
    console.error("deleteFeature", err);
    featureError(err.message || "failed");
    return;
  }

  window.location.href = getLink({ page: "features", tab: spec.tab });
}

function deleteEditing(kind) {
  if (featureEditing[kind]) {
    deleteFeature(kind, featureEditing[kind]);
  }
}
