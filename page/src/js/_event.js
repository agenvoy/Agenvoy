function renderEvent(view, event) {
  const type = event.type;

  if (type === "EventAgentResult") {
    view.model.textContent = event.text || "";
    return;
  }

  if (type === "EventTextDelta") {
    view.streamed = true;
    view.text += event.text || "";
    render(view.answer, view.text);
    return;
  }

  if (type === "EventText") {
    if (view.streamed) {
      return;
    }
    view.text += (view.textStarted ? "\n" : "") + (event.text || "");
    view.textStarted = true;
    render(view.answer, view.text);
    return;
  }

  if (type === "EventReasoning") {
    renderReasoning(view, event.text || "");
    return;
  }

  if (type === "EventSuggest") {
    view.suggests = event.suggests || [];
    renameChat(currentSessionId, event.text || "");
    return;
  }

  if (type === "EventDone") {
    view.think.open = false;
    const usage = event.usage || {};
    const footer = assistantFooter({
      send_at: sendAt(),
      input: compactToken(usage.input_tokens),
      output: compactToken(usage.output_tokens),
    });
    view.footer.replaceWith(footer);
    view.footer = footer;
    renderSuggest(view);
    return;
  }

  renderReasoning(view, formatEvent(event));
}

function renderReasoning(view, line) {
  line = (line || "").trim();
  if (!line) {
    return;
  }
  view.trace += (view.trace ? "\n\n" : "") + line;
  view.think.hidden = false;
  render(view.reasoning, view.trace);
}

function renderSuggest(view) {
  if (!view.suggests || view.suggests.length === 0) {
    return;
  }

  const dom = _("section.suggests");
  for (const text of view.suggests) {
    const btn = _("button", { type: "button" }, text);
    btn.addEventListener("click", () => send(text));
    dom.appendChild(btn);
  }
  view.body.appendChild(dom);
  scrollToBottom();
}

function formatEvent(event) {
  const type = event.type;

  if (type === "EventToolCall") {
    const args = (event.tool_args || "").replace(/\s+/g, " ").slice(0, 120);
    return `⏵ \`${event.tool_name || "tool"}\`${args ? " " + args : ""}`;
  }

  if (type === "EventToolSkipped") {
    return `⏵ skipped \`${event.tool_name || ""}\``;
  }

  if (type === "EventCompact") {
    return "⏵ compacted history";
  }

  if (type === "EventExecError") {
    return `⚠ ${event.tool_name || ""} — ${event.text || "unknown error"}`;
  }

  if (type === "EventError") {
    return `⚠ ${event.text || "unknown error"}`;
  }

  if (type === "EventCanceled") {
    return "⚠ canceled";
  }

  return event.text || "";
}
