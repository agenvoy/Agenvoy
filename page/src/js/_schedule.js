const SCHEDULE_TEMPLATE = `# <title>

## Task

-

## Output

-
`;

const SCHEDULE_SPEC = {
  cron: {
    list: "/v1/cron",
    run: "/v1/cron/run",
    key: "crons",
    tab: "Cron",
    hint: "Cron expressions · {min} {hour} {dom} {mon} {dow}",
    placeholder: "*/5 * * * *",
    empty: "at least one cron expression is required",
  },
  task: {
    list: "/v1/task",
    run: "/v1/task/run",
    key: "tasks",
    tab: "Task",
    hint: "Fire times · '+5m' · '15:04' · 'YYYY-MM-DD HH:MM'",
    placeholder: "2026-01-01 09:00",
    empty: "at least one fire time is required",
  },
};

const SCHEDULE_NAME = /^[A-Za-z0-9_-]{1,64}$/;

const scheduleEditing = { cron: "", task: "" };
const scheduleEntries = { cron: [], task: [] };
const scheduleGroup = { cron: {}, task: {} };

function scheduleDom(kind) {
  return {
    form: $(`#${kind}-form`),
    list: $(`#${kind}-list`),
    name: $(`#${kind}-name`),
    description: $(`#${kind}-description`),
    content: $(`#${kind}-content`),
    entries: $(`#${kind}-entries`),
    submit: document.querySelector(`#${kind}-form > footer button.submit`),
    test: document.querySelector(`#${kind}-form > footer button.test`),
  };
}

function scheduleError(text) {
  alert(text);
}

function scheduleValue(kind, item) {
  return kind === "cron" ? item.expression || "" : scheduleClock(item.at);
}

function scheduleClock(text) {
  const date = new Date(text);
  if (Number.isNaN(date.getTime())) {
    return String(text || "");
  }
  const pad = (n) => String(n).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function markScheduleMode(kind, editing) {
  const dom = scheduleDom(kind);
  if (dom.form) {
    if (editing) {
      dom.form.dataset.editing = "1";
    } else {
      delete dom.form.dataset.editing;
    }
  }
  if (dom.name) {
    dom.name.readOnly = Boolean(editing);
  }
  if (dom.submit) {
    dom.submit.textContent = editing ? "save" : "add";
  }
}

async function renderSchedule(kind) {
  const spec = SCHEDULE_SPEC[kind];
  const dom = scheduleDom(kind);
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
    console.error("renderSchedule", err);
  }

  const group = {};
  for (const item of items) {
    if (!item.skill) continue;
    group[item.skill] = group[item.skill] || [];
    group[item.skill].push(scheduleValue(kind, item));
  }
  scheduleGroup[kind] = group;

  dom.list.innerHTML = "";
  const picked = praseURL().target || "";

  for (const name of Object.keys(group)) {
    const remove = _("button", { type: "button" }, [_("span.material-symbols-outlined", "delete")]);
    remove.addEventListener("click", (e) => {
      e.stopPropagation();
      deleteSchedule(kind, name);
    });

    const card = _("div.card", [_("strong", name), _("p", group[name].join(" · ")), remove]);
    card.dataset.name = name;
    card.dataset.selected = name === picked ? "1" : "0";
    card.addEventListener("click", () => {
      window.location.href = getLink({ page: "features", tab: spec.tab, target: name });
    });
    dom.list.appendChild(card);
  }

  if (picked && group[picked]) {
    openSchedule(kind, picked);
  }
}

function renderScheduleEntries(kind, focus) {
  const spec = SCHEDULE_SPEC[kind];
  const dom = scheduleDom(kind);
  if (!spec || !dom.entries) {
    return;
  }

  dom.entries.innerHTML = "";
  dom.entries.dataset.open = "1";
  dom.entries.appendChild(_("strong", spec.hint));

  scheduleEntries[kind].forEach((value, index) => {
    const shown = _("input", { type: "text", value: value, readonly: "readonly" });
    const drop = _("button.remove", { type: "button" }, "delete");
    drop.addEventListener("click", () => {
      scheduleEntries[kind].splice(index, 1);
      renderScheduleEntries(kind);
    });
    dom.entries.appendChild(_("div.row", [shown, drop]));
  });

  const draft = _("input", { type: "text", placeholder: spec.placeholder });
  const add = _("button.submit", { type: "button" }, "add");
  const commit = () => {
    const value = draft.value.trim();
    if (!value) {
      return;
    }
    scheduleEntries[kind].push(value);
    renderScheduleEntries(kind, true);
  };
  add.addEventListener("click", commit);
  draft.addEventListener("keydown", (e) => {
    if (e.key !== "Enter" || e.isComposing) {
      return;
    }
    e.preventDefault();
    commit();
  });
  dom.entries.appendChild(_("div.row", [draft, add]));

  markScheduleReady(kind);

  if (focus) {
    draft.focus();
  }
}

function markScheduleReady(kind) {
  const dom = scheduleDom(kind);
  const blocked = scheduleEntries[kind].length === 0;
  if (dom.submit) dom.submit.disabled = blocked;
  if (dom.test) dom.test.disabled = blocked;
}

async function openSchedule(kind, name) {
  const dom = scheduleDom(kind);
  if (!dom.name) {
    return;
  }

  try {
    const response = await fetch(`${API}/v1/schedule/${encodeURIComponent(name)}`);
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      scheduleError(detail.error || `HTTP ${response.status}`);
      return;
    }
    const body = await response.json();
    dom.name.value = name;
    dom.description.value = body.description || "";
    dom.content.value = body.body || "";
  } catch (err) {
    console.error("openSchedule", err);
    scheduleError(err.message || "failed");
    return;
  }

  scheduleEditing[kind] = name;
  scheduleEntries[kind] = (scheduleGroup[kind][name] || []).slice();
  markScheduleMode(kind, true);
  renderScheduleEntries(kind);
}

function resetSchedule(kind) {
  const dom = scheduleDom(kind);
  if (dom.name) dom.name.value = "";
  if (dom.description) dom.description.value = "";
  if (dom.content) dom.content.value = SCHEDULE_TEMPLATE;
  scheduleEditing[kind] = "";
  scheduleEntries[kind] = [];
  markScheduleMode(kind, false);
  markSelectedCard(dom.list, "");
  renderScheduleEntries(kind);
}

async function saveSchedule(kind) {
  const spec = SCHEDULE_SPEC[kind];
  const dom = scheduleDom(kind);
  if (!spec || !dom.name) {
    return null;
  }

  const editing = scheduleEditing[kind];
  const name = editing || dom.name.value.trim();
  if (!name) {
    scheduleError("name is required");
    return null;
  }
  if (!SCHEDULE_NAME.test(name)) {
    scheduleError("name allows only A-Z a-z 0-9 _ - and must be 1-64 characters, no spaces");
    return null;
  }
  if (scheduleEntries[kind].length === 0) {
    scheduleError(spec.empty);
    return null;
  }

  const body = {
    target: kind,
    name: name,
    description: dom.description ? dom.description.value.trim() : "",
    content: dom.content ? dom.content.value : "",
    expressions: scheduleEntries[kind],
  };

  let saved = null;
  try {
    const response = await fetch(`${API}/v1/schedule`, {
      method: editing ? "PATCH" : "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      scheduleError(detail.error || `HTTP ${response.status}`);
      return null;
    }
    saved = (await response.json()) || {};
    scheduleEditing[kind] = saved.name || name;
    markScheduleMode(kind, true);
  } catch (err) {
    console.error("saveSchedule", err);
    scheduleError(err.message || "failed");
    return null;
  }
  return { name: saved.name || name, session_id: saved.session_id || "" };
}

async function commitSchedule(kind) {
  const spec = SCHEDULE_SPEC[kind];
  const saved = await saveSchedule(kind);
  if (!spec || !saved) {
    return;
  }
  window.location.href = getLink({ page: "features", tab: spec.tab, target: saved.name });
}

async function testSchedule(kind) {
  const spec = SCHEDULE_SPEC[kind];
  if (!spec) {
    return;
  }

  const saved = await saveSchedule(kind);
  if (!saved) {
    return;
  }
  if (!saved.session_id) {
    scheduleError("no session bound to this schedule");
    return;
  }

  try {
    const response = await fetch(`${API}${spec.run}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ session_id: saved.session_id, skill: saved.name }),
    });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      scheduleError(detail.error || `HTTP ${response.status}`);
      return;
    }
  } catch (err) {
    console.error("testSchedule", err);
    scheduleError(err.message || "failed");
    return;
  }

  if (!CHAT_ID.test(saved.session_id)) {
    scheduleError(`running in ${saved.session_id} · that session is not viewable here`);
    return;
  }
  window.location.href = getLink({ page: "chat", chat: saved.session_id });
}

async function deleteSchedule(kind, name) {
  const spec = SCHEDULE_SPEC[kind];
  if (!spec || !confirm(`Delete "${name}"?`)) {
    return;
  }

  try {
    const response = await fetch(`${API}${spec.list}`, {
      method: "DELETE",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ skill: name }),
    });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      scheduleError(detail.error || `HTTP ${response.status}`);
      return;
    }
  } catch (err) {
    console.error("deleteSchedule", err);
    scheduleError(err.message || "failed");
    return;
  }

  window.location.href = getLink({ page: "features", tab: spec.tab });
}

function deleteEditingSchedule(kind) {
  if (scheduleEditing[kind]) {
    deleteSchedule(kind, scheduleEditing[kind]);
  }
}
