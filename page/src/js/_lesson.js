const LESSON_LIST_LIMIT = 200;
const LESSON_PAGE_SIZE = 10;
function lessonDom() {
  return {
    all: $("#lesson-all"),
    list: $("#lesson-list"),
    body: $("#lesson-body"),
    pager: $("#lesson-pager"),
  };
}

function lessonLink(tool, offset) {
  const params = { page: "monitor", tab: "Lessons" };
  if (tool) {
    params.target = tool;
  }
  if (offset > 0) {
    params.offset = offset;
  }
  return getLink(params);
}

async function fetchLessonRecords() {
  try {
    const response = await fetch(`${API}/v1/torii/error?limit=${LESSON_LIST_LIMIT}`);
    if (response.ok) {
      return (await response.json()).records || [];
    }
  } catch (err) {
    console.error("fetchLessonRecords", err);
  }
  return [];
}

function lessonClock(timestamp) {
  const at = Number(timestamp);
  if (!Number.isFinite(at) || at <= 0) {
    return "";
  }
  return historyClock(new Date(at * 1000));
}

function lessonGroups(records) {
  const dic = {};
  for (const one of records) {
    const tool = one.tool_name || "tool";
    if (!dic[tool]) {
      dic[tool] = [];
    }
    dic[tool].push(one);
  }

  return Object.keys(dic)
    .map((tool) => ({ tool: tool, records: dic[tool] }))
    .sort((a, b) => Number(b.records[0].timestamp) - Number(a.records[0].timestamp));
}

function renderLessonPager(dom, tool, offset, total) {
  dom.pager.innerHTML = "";
  if (total <= LESSON_PAGE_SIZE) {
    return;
  }

  const last = Math.floor((total - 1) / LESSON_PAGE_SIZE) * LESSON_PAGE_SIZE;
  const prev = _("a", { href: lessonLink(tool, Math.max(offset - LESSON_PAGE_SIZE, 0)) }, "prev");
  const next = _("a", { href: lessonLink(tool, Math.min(offset + LESSON_PAGE_SIZE, last)) }, "next");
  if (offset <= 0) {
    prev.dataset.disabled = "1";
  }
  if (offset >= last) {
    next.dataset.disabled = "1";
  }

  dom.pager.appendChild(prev);
  dom.pager.appendChild(textNode("p", `${offset + 1}-${Math.min(offset + LESSON_PAGE_SIZE, total)} / ${total}`));
  dom.pager.appendChild(next);
}

function lessonRecord(one) {
  const parts = [_("div.head", [textNode("strong", lessonClock(one.timestamp))])];

  for (const [label, text] of [
    ["Cause", one.cause],
    ["Action", one.action],
  ]) {
    if (text) {
      parts.push(_("div.block", [textNode("strong", label), textNode("p", text)]));
    }
  }

  const keywords = one.keywords || [];
  if (keywords.length > 0) {
    parts.push(_("div.pills", keywords.map((word) => textNode("span", word))));
  }

  return _("div.record", parts);
}

async function renderLessonPage(pickedTool, offset) {
  const dom = lessonDom();
  if (!dom.list || !dom.body) {
    return;
  }

  dom.list.innerHTML = "";
  dom.body.innerHTML = "";
  dom.pager.innerHTML = "";

  const records = (await fetchLessonRecords()).sort((a, b) => Number(b.timestamp) - Number(a.timestamp));

  const groups = lessonGroups(records);
  const tool = groups.some((group) => group.tool === pickedTool) ? pickedTool : "";

  if (dom.all) {
    dom.all.href = lessonLink("", 0);
    dom.all.dataset.selected = tool === "" ? "1" : "0";
  }

  if (records.length === 0) {
    dom.body.appendChild(textNode("p.empty", "no lesson recorded"));
    return;
  }

  for (const group of groups) {
    const count = group.records.length;
    const card = _("a.card", { href: lessonLink(group.tool, 0) }, [
      textNode("strong", group.tool),
      textNode("p", `${count} record${count === 1 ? "" : "s"} · ${lessonClock(group.records[0].timestamp)}`),
    ]);
    card.dataset.name = group.tool;
    card.dataset.selected = group.tool === tool ? "1" : "0";
    dom.list.appendChild(card);
  }

  const picked = tool ? records.filter((one) => (one.tool_name || "tool") === tool) : records;

  const start = Math.min(Math.max(offset, 0), Math.floor((picked.length - 1) / LESSON_PAGE_SIZE) * LESSON_PAGE_SIZE);
  for (const one of picked.slice(start, start + LESSON_PAGE_SIZE)) {
    dom.body.appendChild(lessonRecord(one));
  }
  renderLessonPager(dom, tool, start, picked.length);
}
