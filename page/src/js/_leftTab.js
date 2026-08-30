const feature = {
  Rules: "deployed_code_account",
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
  Usage: "bar_chart",
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
  return path;
}
