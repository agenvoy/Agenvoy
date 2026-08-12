const ACTION_LINE = /^\[([^\]]+)\]\[([^\]]+)\]\[([^\]]+)\]\s?([\s\S]*)$/;
const ACTION_NEWLINE = "\u001f";

function parseActionLog(content) {
  const list = [];
  let pending = null;

  const close = () => {
    if (pending && (pending.content || pending.Reasoning)) {
      pending.pending = !pending.finished;
      list.push(pending);
    }
    pending = null;
  };

  for (const line of content.split("\n")) {
    const match = ACTION_LINE.exec(line);
    if (!match) {
      continue;
    }

    const sendAt = match[1].slice(0, 16);
    const kind = match[3];
    const body = match[4].split(ACTION_NEWLINE).join("\n").trim();

    switch (kind) {
      case "user":
        if (body.startsWith("[Resumed Task")) {
          break;
        }

        close();
        if (body) {
          list.push({ rule: "user", content: body, meta: { send_at: sendAt } });
        }
        break;

      case "agent_result":
        pending = pending || logItem(sendAt);
        pending.meta.model = body;
        break;

      case "thinking":
        pending = pending || logItem(sendAt);
        pending.Reasoning += (pending.Reasoning ? "\n\n" : "") + body;
        break;

      case "tool_call":
        pending = pending || logItem(sendAt);
        pending.Reasoning += (pending.Reasoning ? "\n\n" : "") + formatTool(body);
        break;

      case "skill_result":
        pending = pending || logItem(sendAt);
        pending.Reasoning += (pending.Reasoning ? "\n\n" : "") + "⏵ skill `" + body + "`";
        break;

      case "assistant":
        pending = pending || logItem(sendAt);
        pending.content += (pending.content ? "\n\n" : "") + body;
        pending.meta.send_at = sendAt;
        break;

      case "done": {
        pending = pending || logItem(sendAt);

        const meta = formatDone(body);
        pending.meta.model = meta.model || pending.meta.model;
        pending.meta.input = meta.input;
        pending.meta.output = meta.output;
        pending.meta.send_at = sendAt;
        pending.finished = true;
        close();
        break;
      }
    }
  }
  close();

  return list;
}

function logItem(sendAt) {
  return { rule: "assistant", content: "", Reasoning: "", finished: false, meta: { model: "", send_at: sendAt } };
}

function formatTool(body) {
  const cut = body.indexOf(" ");
  const name = cut === -1 ? body : body.slice(0, cut);
  const args =
    cut === -1
      ? ""
      : body
          .slice(cut + 1)
          .replace(/\s+/g, " ")
          .slice(0, 120);
  return `⏵ \`${name}\`${args ? " " + args : ""}`;
}

function formatDone(body) {
  const model = (body.split(/\s+/)[0] || "").includes("=") ? "" : body.split(/\s+/)[0] || "";
  const input = /\bin=(\d+)/.exec(body);
  const output = /\bout=(\d+)/.exec(body);
  return {
    model: model,
    input: input ? compactToken(input[1]) : "",
    output: output ? compactToken(output[1]) : "",
  };
}

function compactToken(value) {
  if (value === null || value === undefined || value === "") {
    return "";
  }

  const number = Number(value);
  if (!Number.isFinite(number)) {
    return "";
  }

  if (number < 1_000) {
    return String(number);
  }
  if (number < 1_000_000) {
    return `${(number / 1_000).toFixed(1)}k`;
  }
  return `${(number / 1_000_000).toFixed(1)}m`;
}
