let pendingTask = null;
let pendingAnswers = [];

async function loadPending(sessionId) {
  const dom = $("#right-content-chat-pending");
  if (!dom || !sessionId) {
    return;
  }

  clearPending();

  let list = [];
  try {
    const response = await fetch(`${API}/v1/session/${encodeURIComponent(sessionId)}/pending`);
    if (!response.ok) {
      return;
    }
    list = (await response.json()).pending || [];
  } catch (err) {
    console.error("loadPending", err);
    return;
  }

  const task = list.find((item) => item.has_questions);
  if (!task) {
    return;
  }

  let questions = [];
  try {
    const url = `${API}/v1/session/${encodeURIComponent(sessionId)}/pending/${encodeURIComponent(task.task_hash)}/questions`;
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

  pendingTask = { sessionId: sessionId, taskHash: task.task_hash, questions: questions };
  pendingAnswers = questions.map((q) => (q.multi_select ? [] : ""));

  for (let i = 0; i < questions.length; i++) {
    dom.appendChild(pendingCard(questions[i], i, questions.length));
  }
  dom.dataset.index = "0";
  scrollToBottom(true);
}

function clearPending() {
  const dom = $("#right-content-chat-pending");
  if (!dom) {
    return;
  }
  dom.innerHTML = "";
  delete dom.dataset.index;
  pendingTask = null;
  pendingAnswers = [];
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
    const dom = _("input", { type: "password", placeholder: "輸入回答…" });
    dom.addEventListener("input", () => (pendingAnswers[index] = dom.value));
    return dom;
  }

  const dom = _("textarea", { rows: "3", placeholder: "輸入回答…" });
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
  const dom = $("#right-content-chat-pending");
  if (!dom || !pendingTask) {
    return;
  }

  if (index < total - 1) {
    dom.dataset.index = String(index + 1);
    scrollToBottom(true);
    return;
  }

  const task = pendingTask;
  const answers = pendingAnswers;
  clearPending();
  resumePending(task, answers);
}

async function resumePending(task, answers) {
  const url = `${API}/v1/session/${encodeURIComponent(task.sessionId)}/pending/${encodeURIComponent(task.taskHash)}/resume`;

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
