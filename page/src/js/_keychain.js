let keychainEditing = "";

function keychainDom() {
  return {
    form: $("#keychain-form"),
    list: $("#keychain-list"),
    name: $("#keychain-name"),
    value: $("#keychain-value"),
  };
}

function keychainError(text) {
  alert(text);
}

async function keychainKeys() {
  try {
    const response = await fetch(`${API}/v1/keys`);
    if (response.ok) {
      return (await response.json()).keys || [];
    }
  } catch (err) {
    console.error("keychainKeys", err);
  }
  return [];
}

async function renderKeychain() {
  const dom = keychainDom();
  if (!dom.list) {
    return;
  }

  const keys = await keychainKeys();

  dom.list.innerHTML = "";

  for (const key of keys) {
    if (!key) continue;

    const remove = _("button", { type: "button" }, [_("span.material-symbols-outlined", "delete")]);
    remove.addEventListener("click", (e) => {
      e.stopPropagation();
      deleteKeychain(key);
    });

    const card = _("div.card", [_("strong", key), _("p", "stored"), remove]);
    card.dataset.name = key;
    card.addEventListener("click", () => openKeychain(key));
    dom.list.appendChild(card);
  }
}

function openKeychain(key) {
  const dom = keychainDom();
  if (!dom.form) {
    return;
  }

  dom.name.value = key;
  dom.name.readOnly = true;
  dom.value.value = "";
  dom.value.placeholder = "New value · replaces the stored one";
  dom.form.dataset.editing = "1";
  keychainEditing = key;
}

function resetKeychain() {
  const dom = keychainDom();
  if (!dom.form) {
    return;
  }

  dom.name.value = "";
  dom.name.readOnly = false;
  dom.value.value = "";
  dom.value.placeholder = "Value · stored in the OS keychain, never shown again";
  delete dom.form.dataset.editing;
  keychainEditing = "";
}

async function saveKeychain() {
  const dom = keychainDom();
  if (!dom.form) {
    return;
  }

  const key = dom.name.value.trim();
  const value = dom.value.value.trim();
  if (!key) {
    keychainError("key is required");
    return;
  }
  if (!value) {
    keychainError(keychainEditing ? "enter a new value to replace the stored one" : "value is required");
    return;
  }

  try {
    const response = await fetch(`${API}/v1/keys`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ key: key, value: value }),
    });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      keychainError(detail.error || `HTTP ${response.status}`);
      return;
    }
  } catch (err) {
    console.error("saveKeychain", err);
    keychainError(err.message || "failed");
    return;
  }

  openKeychain(key);
  renderKeychain();
}

async function deleteKeychain(key) {
  if (!key || !confirm(`Delete "${key}"?`)) {
    return;
  }

  try {
    const response = await fetch(`${API}/v1/key?key=${encodeURIComponent(key)}`, { method: "DELETE" });
    if (!response.ok) {
      const detail = await response.json().catch(() => ({}));
      keychainError(detail.error || `HTTP ${response.status}`);
      return;
    }
  } catch (err) {
    console.error("deleteKeychain", err);
    keychainError(err.message || "failed");
    return;
  }

  if (keychainEditing === key) {
    resetKeychain();
  }
  renderKeychain();
}

function deleteEditingKeychain() {
  if (keychainEditing) {
    deleteKeychain(keychainEditing);
  }
}
