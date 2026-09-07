let pendingTask = null;
let pendingAnswers = [];

async function loadPending(sessionId, taskHash) {
  const dom = chatPart("pending", sessionId);
  if (!dom || !sessionId || !taskHash) {
    return;
  }

  clearPending(sessionId);

  let questions = [];
  try {
    const url = `${API}/v1/session/${encodeURIComponent(sessionId)}/task/${encodeURIComponent(taskHash)}/questions`;
    const response = await fetch(url);
    if (!response.ok) {
      return;
    }
    questions = (await response.json()).questions || [];
  } catch (err) {
    console.error("loadPending", err);
    return;
  }
  if (questions.length === 0) {
    return;
  }

  pendingTask = { sessionId: sessionId, taskHash: taskHash, questions: questions };
  pendingAnswers = questions.map((q) => (q.multi_select ? [] : ""));

  for (let i = 0; i < questions.length; i++) {
    dom.appendChild(pendingCard(questions[i], i, questions.length));
  }
  dom.dataset.index = "0";
  setInputTask(sessionId, taskHash);
  scrollToBottom(true, sessionId);
}

function clearPending(sessionId) {
  const dom = chatPart("pending", sessionId);
  if (!dom) {
    return;
  }
  dom.innerHTML = "";
  delete dom.dataset.index;
  if (!pendingTask || !sessionId || pendingTask.sessionId === sessionId) {
    pendingTask = null;
    pendingAnswers = [];
  }
}

function pendingCard(question, index, total) {
  const body = [_("strong", question.question || "")];

  if (question.detail) {
    body.push(_("p", { textContent: question.detail }));
  }

  const options = question.options || [];
  if (options.length === 0) {
    body.push(pendingInput(question, index));
  } else {
    for (const option of options) {
      body.push(pendingOption(question, index, option));
    }
  }

  const last = index === total - 1;
  const submit = _("button", { type: "button", class: "submit" }, last ? "Send" : "Next");
  submit.addEventListener("click", () => answerPending(index, total));

  const skip = _("button", { type: "button", class: "skip" }, "Skip");
  skip.addEventListener("click", () => {
    pendingAnswers[index] = question.multi_select ? [] : "";
    answerPending(index, total);
  });

  body.push(_("footer", [_("p", `${index + 1} / ${total}`), skip, submit]));
  return _("div", body);
}

function pendingInput(question, index) {
  if (question.secret) {
    const dom = _("input", { type: "password", placeholder: "Type your answer..." });
    dom.addEventListener("input", () => (pendingAnswers[index] = dom.value));
    return dom;
  }

  const dom = _("textarea", { rows: "3", placeholder: "Type your answer..." });
  dom.addEventListener("input", () => (pendingAnswers[index] = dom.value));
  dom.addEventListener("keydown", (e) => {
    if (e.key !== "Enter" || e.shiftKey || e.isComposing) {
      return;
    }
    e.preventDefault();
    answerPending(index, pendingTask ? pendingTask.questions.length : index + 1);
  });
  return dom;
}

function pendingOption(question, index, option) {
  const box = _("input", {
    type: question.multi_select ? "checkbox" : "radio",
    name: `pending-${index}`,
    value: option,
  });

  box.addEventListener("change", () => {
    if (!question.multi_select) {
      pendingAnswers[index] = option;
      return;
    }
    const picked = pendingAnswers[index].filter((item) => item !== option);
    if (box.checked) {
      picked.push(option);
    }
    pendingAnswers[index] = picked;
  });

  return _("label.option", [box, _("span", option)]);
}

function answerPending(index, total) {
  const sessionId = pendingTask ? pendingTask.sessionId : "";
  const dom = chatPart("pending", sessionId);
  if (!dom || !pendingTask) {
    return;
  }

  if (index < total - 1) {
    dom.dataset.index = String(index + 1);
    scrollToBottom(true, sessionId);
    return;
  }

  const task = pendingTask;
  const answers = pendingAnswers;
  clearPending(sessionId);
  resumePending(task, answers);
}

async function cancelPending(sessionId) {
  if (!pendingTask || pendingTask.sessionId !== sessionId) {
    return;
  }
  const taskHash = pendingTask.taskHash;
  if (!confirm("Cancel this task?")) {
    return;
  }
  clearPending(sessionId);
  setInputTask(sessionId, "");
  clearTaskCookie(sessionId);
  await deletePending(sessionId, taskHash);
}

async function resumePending(task, answers) {
  const url = `${API}/v1/session/${encodeURIComponent(task.sessionId)}/task/${encodeURIComponent(task.taskHash)}/resume`;
  writeTaskCookie(task.sessionId, task.taskHash);
  setInputTask(task.sessionId, task.taskHash);

  try {
    const response = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ answers: answers }),
    });
    if (!response.ok) {
      console.error("resumePending", response.status);
    }
  } catch (err) {
    console.error("resumePending", err);
  }
}

async function listResumable(sessionId) {
  try {
    const response = await fetch(`${API}/v1/session/${encodeURIComponent(sessionId)}/task`);
    if (!response.ok) {
      console.error("listResumable", response.status);
      return [];
    }
    return (await response.json()).pending || [];
  } catch (err) {
    console.error("listResumable", err);
    return [];
  }
}

async function deletePending(sessionId, taskHash) {
  const url = `${API}/v1/session/${encodeURIComponent(sessionId)}/task/${encodeURIComponent(taskHash)}`;
  try {
    const response = await fetch(url, { method: "DELETE" });
    if (!response.ok && response.status !== 404) {
      const detail = await response.json().catch(() => ({}));
      alert(detail.error || `HTTP ${response.status}`);
      return false;
    }
  } catch (err) {
    console.error("deletePending", err);
    alert(err.message || "failed");
    return false;
  }
  if (pendingTask && pendingTask.sessionId === sessionId && pendingTask.taskHash === taskHash) {
    clearPending(sessionId);
    pendingTask = null;
  }
  return true;
}

function clearPendingHint(sessionId) {
  const dom = chatPart("pending", sessionId);
  if (!dom) {
    return;
  }
  for (const old of dom.querySelectorAll(":scope > button.hint")) {
    old.remove();
  }
}

function hintBlocked(sessionId) {
  const dom = chatPart("pending", sessionId);
  const confirmDom = chatPart("confirm", sessionId);
  return !dom || dom.childElementCount > 0 || (confirmDom && confirmDom.childElementCount > 0);
}

async function renderPendingHint(sessionId) {
  if (!sessionId || hintBlocked(sessionId)) {
    return;
  }

  const tasks = await listResumable(sessionId);
  const mine = readTaskCookie();
  if (mine && mine.session === sessionId) {
    const own = tasks.find((one) => one.task_hash === mine.task);
    if (!own) {
      clearTaskCookie(sessionId);
    } else if (own.has_questions) {
      loadPending(sessionId, own.task_hash);
      return;
    }
  }
  if (tasks.length === 0 || hintBlocked(sessionId)) {
    return;
  }

  const dom = chatPart("pending", sessionId);
  for (const one of tasks) {
    const title = String(one.objective || "").replace(/\s+/g, " ").trim() || one.task_hash;
    const dot = _("button", { type: "button", class: "hint" }, [
      _("span.material-symbols-outlined", "live_help"),
      _("p", title),
    ]);
    dot.addEventListener("click", () => {
      writeTaskCookie(sessionId, one.task_hash);
      if (one.has_questions) {
        loadPending(sessionId, one.task_hash);
        return;
      }
      if (!confirm(`Resume in this chat?\n\n${title}`)) {
        return;
      }
      startResume(sessionId, one.task_hash);
    });
    dom.appendChild(dot);
  }
}

function startResume(sessionId, taskHash) {
  if (pendingTask && pendingTask.sessionId === sessionId && pendingTask.taskHash === taskHash) {
    clearPending(sessionId);
  }
  resumePending({ sessionId: sessionId, taskHash: taskHash }, []);
}
