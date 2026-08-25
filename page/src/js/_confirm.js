const CONFIRM_OPTIONS = [
  { label: "yes", body: { approve: true }, style: "approve" },
  { label: "yes, don't ask again", body: { approve: true, remember: true }, style: "approve" },
  { label: "yes, allow this turn", body: { approve: true, allow_turn: true }, style: "approve" },
  { label: "no", body: { approve: false }, style: "reject" },
  { label: "abort task", body: { approve: false, abort: true }, style: "abort" },
];

async function resolveToolConfirm(requestId, body, dom) {
  const note = dom.querySelector("p.note");
  try {
    const response = await fetch(
      `${API}/v1/session/${encodeURIComponent(currentSessionId)}/confirm/${encodeURIComponent(requestId)}`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      },
    );
    if (response.status === 401) {
      const detail = await response.json().catch(() => ({}));
      if (note) {
        note.textContent = detail.error || "password rejected";
      }
      return;
    }
  } catch (err) {
    console.error("resolveToolConfirm", err);
  }
  dom.remove();
}

function renderToolConfirm(event) {
  const requestId = event.tool_id || "";
  const dom = $("#right-content-chat-confirm");
  if (!requestId || !dom) {
    return;
  }
  if (dom.querySelector(`div.tool-confirm[data-id="${requestId}"]`)) {
    return;
  }

  const body = [_("p.name", event.tool_name || "tool")];
  const args = (event.tool_args || "").trim();
  if (args && args !== "{}") {
    body.push(_("pre.args", args));
  }

  const restricted = event.restricted || [];
  let password = null;
  let note = null;
  if (restricted.length > 0) {
    body.push(_("p.restricted", `needs your system password · ${restricted.join(", ")}`));
    password = _("input", {
      type: "password",
      autocomplete: "off",
      "data-1p-ignore": "true",
      "data-lpignore": "true",
      placeholder: "System password",
    });
    note = _("p.note", "");
    password.addEventListener("input", () => {
      note.textContent = "";
    });
    body.push(password);
    body.push(note);
  }

  const action = _("div.action");
  let primary = null;
  const card = _("div.tool-confirm", { "data-id": requestId }, [...body, action]);
  for (const option of CONFIRM_OPTIONS) {
    if (restricted.length > 0 && option.body.remember) {
      continue;
    }
    const button = _("button", { type: "button", class: option.style }, [_("p", option.label)]);
    button.addEventListener("click", () => {
      const payload = Object.assign({}, option.body);
      if (password && option.body.approve) {
        const field = card.querySelector('input[type="password"]') || password;
        if (field.value === "") {
          note.textContent = "type your system password first";
          field.focus();
          return;
        }
        payload.password = field.value;
        note.textContent = "";
      }
      resolveToolConfirm(requestId, payload, card);
    });
    if (password && option.body.approve && !primary) {
      primary = button;
    }
    action.appendChild(button);
  }

  if (password && primary) {
    password.addEventListener("keydown", (e) => {
      if (e.key === "Enter") {
        e.preventDefault();
        primary.click();
      }
    });
  }

  dom.appendChild(card);
  scrollToBottom(true);
  if (password) {
    password.focus();
  }
}
