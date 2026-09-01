const SCHEDULE_TEMPLATE = `# <title>

## Task

-

## Output

-
`;

const SCHEDULE_TYPES = {
  cron: {
    label: "Cron",
    input: "text",
    hint: "Cron expressions · {min} {hour} {dom} {mon} {dow} · @daily · @every 30m",
    placeholder: "*/5 * * * *",
    value: "*/5 * * * *",
    empty: "at least one cron expression is required",
  },
  task: {
    label: "Task",
    input: "datetime-local",
    hint: "Fire times · local date and time",
    placeholder: "2026-01-01 09:00",
    value: "",
    empty: "at least one fire time is required",
  },
};

const SCHEDULE_NAME = /^[A-Za-z0-9_-]{1,64}$/;

let scheduleType = "cron";
let scheduleEditing = "";
let scheduleEntries = [];
let scheduleGroup = {};
let scheduleSessions = [];

function scheduleDom() {
  return {
    form: $("#schedule-form"),
    list: $("#schedule-list"),
    type: $("#schedule-type"),
    session: $("#schedule-session"),
    name: $("#schedule-name"),
    description: $("#schedule-description"),
    content: $("#schedule-content"),
    entries: $("#schedule-entries"),
    submit: document.querySelector("#schedule-form > footer button.submit"),
    test: document.querySelector("#schedule-form > footer button.test"),
  };
}

function scheduleError(text) {
  alert(text);
}

function scheduleValue(item) {
  return item.type === "cron" ? item.expression || "" : scheduleClock(item.at);
}

function scheduleInput(value) {
  return scheduleType === "task" ? value.replace(" ", "T") : value;
}

function scheduleEntry(value) {
  return scheduleType === "task" ? value.replace("T", " ") : value;
}

function scheduleClock(text) {
  const date = new Date(text);
  if (Number.isNaN(date.getTime())) {
    return String(text || "");
  }
  const pad = (n) => String(n).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function markScheduleMode(editing) {
  const dom = scheduleDom();
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

async function loadScheduleSessions() {
  try {
    const response = await fetch(`${API}/v1/sessions`);
    if (response.ok) {
      return ((await response.json()).sessions || []).filter((one) => {
        const id = one.id || "";
        return SESSION_ID.test(id) && !id.startsWith("temp-");
      });
    }
  } catch (err) {
    console.error("loadScheduleSessions", err);
  }
  return [];
}

function renderScheduleSessions(picked) {
  const dom = scheduleDom();
  if (!dom.session) {
    return;
  }

  const list = scheduleSessions.slice();
  if (picked && !list.some((one) => one.id === picked)) {
    list.unshift({ id: picked, name: picked });
  }

  dom.session.innerHTML = "";
  if (list.length === 0) {
    dom.session.appendChild(_("option", { value: "" }, "no session available · start a chat first"));
  }
  for (const one of list) {
    const option = _("option", { value: one.id }, one.name || one.id);
    if (one.id === picked) {
      option.selected = true;
    }
    dom.session.appendChild(option);
  }

  markScheduleReady();
}

function renderScheduleType() {
  const dom = scheduleDom();
  if (!dom.type) {
    return;
  }

  dom.type.innerHTML = "";
  for (const [value, spec] of Object.entries(SCHEDULE_TYPES)) {
    const button = _("button", { type: "button" }, spec.label);
    button.dataset.selected = scheduleType === value ? "1" : "0";
    button.addEventListener("click", () => {
      if (scheduleType === value) {
        return;
      }
      if (scheduleEntries.length > 0 && !confirm(`Switch to ${spec.label}? The current entries will be cleared.`)) {
        return;
      }
      scheduleType = value;
      scheduleEntries = [];
      renderScheduleType();
      renderScheduleEntries();
    });
    dom.type.appendChild(button);
  }
}

async function renderSchedule() {
  const dom = scheduleDom();
  if (!dom.list) {
    return;
  }

  scheduleSessions = await loadScheduleSessions();

  let items = [];
  try {
    const response = await fetch(`${API}/v1/schedule`);
    if (response.ok) {
      items = (await response.json()).schedules || [];
    }
  } catch (err) {
    console.error("renderSchedule", err);
  }

  const group = {};
  for (const item of items) {
    if (!item.skill) continue;
    group[item.skill] = group[item.skill] || { type: item.type || "cron", session_id: item.session_id || "", values: [] };
    group[item.skill].values.push(scheduleValue(item));
  }
  scheduleGroup = group;

  dom.list.innerHTML = "";
  const picked = praseURL().target || "";

  for (const name of Object.keys(group)) {
    const remove = _("button", { type: "button" }, [_("span.material-symbols-outlined", "delete")]);
    remove.addEventListener("click", (e) => {
      e.stopPropagation();
      deleteSchedule(name);
    });

    const card = _("div.card", [
      _("strong", name),
      _("p", `${SCHEDULE_TYPES[group[name].type].label} · ${group[name].values.join(" · ")}`),
      remove,
    ]);
    card.dataset.name = name;
    card.dataset.selected = name === picked ? "1" : "0";
    card.addEventListener("click", () => {
      window.location.href = getLink({ page: "features", tab: "Schedule", target: name });
    });
    dom.list.appendChild(card);
  }

  if (picked && group[picked]) {
    openSchedule(picked);
    return;
  }
  renderScheduleSessions("");
}

function renderScheduleEntries(focus) {
  const dom = scheduleDom();
  if (!dom.entries) {
    return;
  }
  const spec = SCHEDULE_TYPES[scheduleType];

  dom.entries.innerHTML = "";
  dom.entries.dataset.open = "1";
  dom.entries.appendChild(_("strong", spec.hint));

  scheduleEntries.forEach((value, index) => {
    const shown = _("input", { type: spec.input, value: scheduleInput(value), readonly: "readonly" });
    const drop = _("button.remove", { type: "button" }, "delete");
    drop.addEventListener("click", () => {
      scheduleEntries.splice(index, 1);
      renderScheduleEntries();
    });
    dom.entries.appendChild(_("div.row", [shown, drop]));
  });

  const draft = _("input", { type: spec.input, placeholder: spec.placeholder, value: spec.value });
  const add = _("button.submit", { type: "button" }, "add");
  const commit = () => {
    const value = scheduleEntry(draft.value.trim());
    if (!value) {
      return;
    }
    scheduleEntries.push(value);
    renderScheduleEntries(true);
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

  markScheduleReady();

  if (focus) {
    draft.focus();
  }
}

function markScheduleReady() {
  const dom = scheduleDom();
  const blocked = scheduleEntries.length === 0 || scheduleSessions.length === 0;
  if (dom.submit) dom.submit.disabled = blocked;
  if (dom.test) dom.test.disabled = blocked;
}

async function openSchedule(name) {
  const dom = scheduleDom();
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

  const group = scheduleGroup[name] || { type: "cron", values: [] };
  scheduleEditing = name;
  scheduleType = group.type;
  scheduleEntries = group.values.slice();
  markScheduleMode(true);
  renderScheduleSessions(group.session_id);
  renderScheduleType();
  renderScheduleEntries();
}

function resetSchedule() {
  const dom = scheduleDom();
  if (dom.name) dom.name.value = "";
  if (dom.description) dom.description.value = "";
  if (dom.content) dom.content.value = SCHEDULE_TEMPLATE;
  scheduleEditing = "";
  scheduleType = "cron";
  scheduleEntries = [];
  markScheduleMode(false);
  markSelectedCard(dom.list, "");
  renderScheduleSessions("");
  renderScheduleType();
  renderScheduleEntries();
}

async function saveSchedule() {
  const spec = SCHEDULE_TYPES[scheduleType];
  const dom = scheduleDom();
  if (!spec || !dom.name) {
    return null;
  }

  const editing = scheduleEditing;
  const name = editing || dom.name.value.trim();
  if (!name) {
    scheduleError("name is required");
    return null;
  }
  if (!SCHEDULE_NAME.test(name)) {
    scheduleError("name allows only A-Z a-z 0-9 _ - and must be 1-64 characters, no spaces");
    return null;
  }
  if (scheduleEntries.length === 0) {
    scheduleError(spec.empty);
    return null;
  }
  const sessionID = dom.session ? dom.session.value : "";
  if (!sessionID) {
    scheduleError("no session available · start a chat first");
    return null;
  }

  const body = {
    type: scheduleType,
    name: name,
    description: dom.description ? dom.description.value.trim() : "",
    content: dom.content ? dom.content.value : "",
    session_id: sessionID,
    expressions: scheduleEntries,
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
    scheduleEditing = saved.name || name;
    markScheduleMode(true);
  } catch (err) {
    console.error("saveSchedule", err);
    scheduleError(err.message || "failed");
    return null;
  }
  return { name: saved.name || name, session_id: saved.session_id || "" };
}

async function commitSchedule() {
  const saved = await saveSchedule();
  if (!saved) {
    return;
  }
  window.location.href = getLink({ page: "features", tab: "Schedule", target: saved.name });
}

async function testSchedule() {
  const saved = await saveSchedule();
  if (!saved) {
    return;
  }
  if (!saved.session_id) {
    scheduleError("no session bound to this schedule");
    return;
  }

  try {
    const response = await fetch(`${API}/v1/schedule/run`, {
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

async function deleteSchedule(name) {
  if (!confirm(`Delete "${name}"?`)) {
    return;
  }

  try {
    const response = await fetch(`${API}/v1/schedule`, {
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

  window.location.href = getLink({ page: "features", tab: "Schedule" });
}

function deleteEditingSchedule() {
  if (scheduleEditing) {
    deleteSchedule(scheduleEditing);
  }
}
