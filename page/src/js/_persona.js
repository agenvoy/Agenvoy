async function openPersonaPopup() {
  if (!currentSessionId) {
    return;
  }

  let current = {};
  try {
    const response = await fetch(`${API}/v1/session/${encodeURIComponent(currentSessionId)}/persona`);
    const detail = await response.json().catch(() => ({}));
    if (!response.ok) {
      alert(`persona failed: ${detail.error || response.status}`);
      return;
    }
    current = detail;
  } catch (err) {
    console.error("openPersonaPopup", err);
    alert(`persona failed: ${err.message || err}`);
    return;
  }

  const self = personaField(current.self_id || "", "A-Z a-z 0-9 _ - only, up to 32 characters", true);
  const name = personaField(current.name || "", "shown as the session title", true);
  const rule = personaField(current.body || "", "system prompt for this session", false);

  const cancel = _("button", { type: "button" }, "cancel");
  const save = _("button", { type: "button", class: "submit" }, "save");

  const root = _("div.popup", [
    _("div.panel", [
      _("strong", "Bot"),
      _("p", "name"),
      name.field,
      _("p", "self id"),
      self.field,
      _("p", "rule"),
      rule.field,
      _("footer", [cancel, save]),
    ]),
  ]);
  root.id = "persona-popup";

  const close = function () {
    document.removeEventListener("keydown", escape);
    root.remove();
  };
  const escape = function (e) {
    if (e.key === "Escape") {
      close();
    }
  };
  const submit = async function () {
    save.disabled = true;
    const done = await savePersona(self.box.value.trim(), name.box.value.trim(), rule.box.value);
    save.disabled = false;
    if (done) {
      close();
    }
  };

  for (const one of [name, self]) {
    one.box.addEventListener("keydown", function (e) {
      if (e.key !== "Enter" || e.shiftKey || e.isComposing) {
        return;
      }
      e.preventDefault();
      submit();
    });
  }
  cancel.addEventListener("click", close);
  save.addEventListener("click", submit);
  root.addEventListener("click", function (e) {
    if (e.target === root) {
      close();
    }
  });
  document.addEventListener("keydown", escape);

  document.body.appendChild(root);
  name.box.focus();
  name.box.select();
}

function personaField(value, placeholder, single) {
  const box = _("textarea", { placeholder: placeholder });
  box.value = value;

  const mirror = _("pre");
  mirror.textContent = box.value + "\n";

  box.addEventListener("input", function () {
    if (single) {
      box.value = box.value.replace(/\n/g, "");
    }
    mirror.textContent = box.value + "\n";
  });

  return { field: _("label.input", [box, mirror]), box: box };
}

async function savePersona(selfId, name, rule) {
  try {
    const response = await fetch(`${API}/v1/session/${encodeURIComponent(currentSessionId)}/persona`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ self_id: selfId, name: name, body: rule }),
    });
    const detail = await response.json().catch(() => ({}));
    if (!response.ok) {
      alert(`persona failed: ${detail.error || response.status}`);
      return false;
    }
  } catch (err) {
    console.error("savePersona", err);
    alert(`persona failed: ${err.message || err}`);
    return false;
  }
  return true;
}
