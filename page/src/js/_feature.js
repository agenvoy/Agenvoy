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
  rule: { list: "/v1/rules", item: "/v1/rule", key: "rules", template: RULE_TEMPLATE },
  knowledge: {
    list: "/v1/knowledges",
    item: "/v1/knowledge",
    key: "knowledges",
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
  };
}

function featureError(text) {
  alert(text);
}

const FEATURE_TAB = { rule: "Rules", knowledge: "Knowledge" };

function featureCount(kind, total) {
  const tab = FEATURE_TAB[kind];
  const dom = tab && $(`section.feature header a[name="${tab}"] span:not(.material-symbols-outlined)`);
  if (dom) {
    dom.textContent = total;
  }
}

async function countFeature(kind) {
  const spec = FEATURE_SPEC[kind];
  if (!spec) {
    return;
  }
  try {
    const response = await fetch(`${API}${spec.list}`);
    if (response.ok) {
      featureCount(kind, ((await response.json())[spec.key] || []).length);
    }
  } catch (err) {
    console.error("countFeature", err);
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

  featureCount(kind, items.length);

  dom.list.innerHTML = "";

  for (const item of items) {
    if (!item.name) continue;

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
    card.addEventListener("click", () => openFeature(kind, item.name));
    dom.list.appendChild(card);
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
    if (dom.form) dom.form.dataset.editing = "1";
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
  if (dom.form) delete dom.form.dataset.editing;
  featureEditing[kind] = "";
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
    featureEditing[kind] = ((await response.json()) || {}).name || name;
    if (dom.form) dom.form.dataset.editing = "1";
  } catch (err) {
    console.error("saveFeature", err);
    featureError(err.message || "failed");
    return;
  }
  renderFeature(kind);
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

  if (featureEditing[kind] === name) {
    resetFeature(kind);
  }
  renderFeature(kind);
}

function deleteEditing(kind) {
  if (featureEditing[kind]) {
    deleteFeature(kind, featureEditing[kind]);
  }
}
