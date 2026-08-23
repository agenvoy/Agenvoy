const CONFIRM_OPTIONS = [
  { label: "yes", body: { approve: true }, style: "approve" },
  { label: "yes, don't ask again", body: { approve: true, remember: true }, style: "approve" },
  { label: "yes, allow this turn", body: { approve: true, allow_turn: true }, style: "approve" },
  { label: "no", body: { approve: false }, style: "reject" },
  { label: "abort task", body: { approve: false, abort: true }, style: "abort" },
];

async function resolveToolConfirm(requestId, body, dom) {
  dom.remove();
  try {
    await fetch(`${API}/v1/session/${encodeURIComponent(currentSessionId)}/confirm/${encodeURIComponent(requestId)}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  } catch (err) {
    console.error("resolveToolConfirm", err);
  }
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

  const action = _("div.action");
  const card = _("div.tool-confirm", { "data-id": requestId }, [...body, action]);
  for (const option of CONFIRM_OPTIONS) {
    const button = _("button", { type: "button", class: option.style }, [_("p", option.label)]);
    button.addEventListener("click", () => resolveToolConfirm(requestId, option.body, card));
    action.appendChild(button);
  }

  dom.appendChild(card);
  scrollToBottom(true);
}
