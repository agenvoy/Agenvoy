function renderEvent(view, event) {
  const type = event.type;

  if (type === "EventAgentResult") {
    view.model.textContent = event.text || "";
    return;
  }

  if (type === "EventTextDelta") {
    view.streamed = true;
    view.text += resumeMark(view) + (event.text || "");
    renderAnswer(view);
    return;
  }

  if (type === "EventText") {
    if (view.streamed) {
      return;
    }
    const join = view.textStarted ? "\n" : "";
    view.text += resumeMark(view) + join + (event.text || "");
    view.textStarted = true;
    renderAnswer(view);
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

  if (type === "EventCanceled") {
    view.stop?.remove();
    view.think.open = false;
    const canceled = assistantFooter({
      send_at: sendAt(),
      canceled: true,
      duration: compactDuration(event.duration),
    });
    view.footer.replaceWith(canceled);
    view.footer = canceled;
    return;
  }

  if (type === "EventFileChanged") {
    view.changedFiles = event.files || [];
    renderFileBox(view.files, view.changedFiles);
    return;
  }

  if (type === "EventDone") {
    view.stop?.remove();
    view.think.open = false;
    const usage = event.usage || {};
    const footer = assistantFooter({
      send_at: sendAt(),
      duration: compactDuration(event.duration),
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

function renderAnswer(view) {
  view.source.textContent = view.text;
  render(view.answer, view.text);
  if (view.answered) {
    return;
  }
  view.answered = true;
  scrollToBottom(true);
}

function resumeMark(view) {
  if (!view.resumed) {
    return "";
  }
  view.resumed = false;
  return "\n\n---\n\n";
}

function renderReasoning(view, line) {
  line = (line || "").trim();
  if (!line) {
    return;
  }
  view.trace += (view.trace ? "\n\n" : "") + line;
  view.think.hidden = false;
  view.resumed = Boolean(view.text);
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
    const label = (event.tool_display || "").replace(/\s+/g, " ").slice(0, 120);
    return label ? `⏵ \`${label}\`` : "";
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
