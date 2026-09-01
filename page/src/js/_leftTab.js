const feature = {
  Rules: "contract",
  Knowledge: "book_2",
  Skills: "lightbulb_2",
  Cron: "schedule",
  Task: "alarm",
};

const configTab = {
  Model: "deployed_code",
  MCP: "electrical_services",
  Keychain: "key",
  Channel: "cell_tower",
};

const monitorTab = {
  Usage: "bar_chart",
  History: "history",
  Lessons: "cognition_2",
  Daemon: "terminal",
};

const usageTab = {
  "24h": "schedule",
  "7d": "schedule",
  "28d": "schedule",
};

const leftTab = {
  "New Chat": {
    icon: "add_comment",
    href: getLink({ page: "chat" }),
  },
};

const leftTabSkip = ["Cron", "Task"];

for (const name of Object.keys(feature)) {
  if (leftTabSkip.includes(name)) {
    continue;
  }
  leftTab[name] = {
    icon: feature[name],
    href: getLink({ page: "features", tab: name }),
  };
}

function getLink(params) {
  let path = "?";
  if (params.page) {
    path += `page=${params.page}`;
  }
  if (params.tab) {
    path += `&tab=${params.tab}`;
  }
  if (params.chat) {
    path += `&chat=${params.chat}`;
  }
  if (params.target) {
    path += `&target=${encodeURIComponent(params.target)}`;
  }
  if (params.offset) {
    path += `&offset=${params.offset}`;
  }
  if (params.hash) {
    path += `&hash=${encodeURIComponent(params.hash)}`;
  }
  if (params.item) {
    path += `&item=${encodeURIComponent(params.item)}`;
  }
  if (params.outcome) {
    path += `&outcome=${encodeURIComponent(params.outcome)}`;
  }
  if (params.keyword) {
    path += `&keyword=${encodeURIComponent(params.keyword)}`;
  }
  if (params.from) {
    path += `&from=${params.from}`;
  }
  if (params.to) {
    path += `&to=${params.to}`;
  }
  return path;
}
