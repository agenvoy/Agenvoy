const ACTION_LINE = /^\[([^\]]+)\]\[([^\]]+)\]\[([^\]]+)\]\s?([\s\S]*)$/;
const ACTION_NEWLINE = "\u001f";
const DURATION_UNIT = { ns: 1e-6, us: 1e-3, "\u00b5s": 1e-3, ms: 1, s: 1000, m: 60000, h: 3600000 };

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

      case "steer": {
        if (!body) {
          break;
        }
        const last = list[list.length - 1];
        if (last && last.rule === "user") {
          last.content += steerMark(sendAt, body, !last.steered);
          last.steered = true;
          break;
        }
        list.push({ rule: "user", content: body, meta: { send_at: sendAt } });
        break;
      }

      case "agent_result":
        pending = pending || logItem(sendAt);
        pending.meta.model = body;
        break;

      case "thinking":
        pending = pending || logItem(sendAt);
        pending.Reasoning += (pending.Reasoning ? "\n\n" : "") + body;
        pending.resumed = Boolean(pending.content);
        break;

      case "tool_call": {
        pending = pending || logItem(sendAt);
        const todos = parseTodoArgs(body);
        if (todos) {
          pending.todos = todos;
          break;
        }
        pending.Reasoning += (pending.Reasoning ? "\n\n" : "") + formatTool(body);
        pending.resumed = Boolean(pending.content);
        break;
      }

      case "todo": {
        pending = pending || logItem(sendAt);
        const list = parseTodoLine(body);
        if (list) {
          pending.todos = list;
        }
        break;
      }

      case "skill_result":
        pending = pending || logItem(sendAt);
        pending.Reasoning += (pending.Reasoning ? "\n\n" : "") + "⏵ skill `" + body + "`";
        pending.resumed = Boolean(pending.content);
        break;

      case "assistant":
        pending = pending || logItem(sendAt);
        pending.content += (pending.resumed ? "\n\n---\n\n" : pending.content ? "\n\n" : "") + body;
        pending.resumed = false;
        pending.meta.send_at = sendAt;
        break;

      case "canceled": {
        pending = pending || logItem(sendAt);

        const meta = formatDone(body);
        pending.meta.model = meta.model || pending.meta.model;
        pending.meta.duration = meta.duration;
        pending.meta.send_at = sendAt;
        pending.meta.canceled = sendAt;
        pending.finished = true;
        close();
        break;
      }

      case "done": {
        pending = pending || logItem(sendAt);

        const meta = formatDone(body);
        pending.meta.model = meta.model || pending.meta.model;
        pending.meta.duration = meta.duration;
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
  return {
    rule: "assistant",
    content: "",
    Reasoning: "",
    resumed: false,
    finished: false,
    meta: { model: "", send_at: sendAt },
  };
}

function steerMark(sendAt, body, first) {
  return (first ? "\n===\n" : "\n") + sendAt + " - " + body;
}

function parseTodoLine(body) {
  try {
    const list = JSON.parse(body);
    return Array.isArray(list) ? list : null;
  } catch (err) {
    console.error("parseTodoLine", err);
    return null;
  }
}

function parseTodoArgs(body) {
  if (!body.startsWith("write_todo ")) {
    return null;
  }
  try {
    const todos = JSON.parse(body.slice("write_todo ".length)).todos;
    return Array.isArray(todos) ? todos : null;
  } catch (err) {
    console.error("parseTodoArgs", err);
    return null;
  }
}

function formatTool(body) {
  const label = body.replace(/\s+/g, " ").slice(0, 120);
  return label ? `⏵ \`${label}\`` : "";
}

function formatDone(body) {
  const model = (body.split(/\s+/)[0] || "").includes("=") ? "" : body.split(/\s+/)[0] || "";
  const duration = /\bdur=(\S+)/.exec(body);
  const input = /\bin=(\d+)/.exec(body);
  const output = /\bout=(\d+)/.exec(body);
  return {
    model: model,
    duration: duration ? compactDuration(duration[1]) : "",
    input: input ? compactToken(input[1]) : "",
    output: output ? compactToken(output[1]) : "",
  };
}

function compactDuration(value) {
  let ms = 0;
  if (typeof value === "number") {
    ms = value / 1e6;
  } else if (typeof value === "string") {
    for (const match of value.matchAll(/([\d.]+)(ms|us|\u00b5s|ns|s|m|h)/g)) {
      ms += parseFloat(match[1]) * DURATION_UNIT[match[2]];
    }
  }

  if (!Number.isFinite(ms) || ms <= 0) {
    return "";
  }
  if (ms < 1000) {
    return `${Math.round(ms)}ms`;
  }

  const sec = ms / 1000;
  if (sec < 60) {
    return `${sec.toFixed(1)}s`;
  }
  return `${Math.floor(sec / 60)}m ${Math.round(sec % 60)}s`;
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
