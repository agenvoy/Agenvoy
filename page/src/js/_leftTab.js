const leftTab = {
  "New Chat": {
    icon: "add_comment",
    href: getLink({ page: "chat" }),
  },
  Nodes: {
    icon: "note_stack",
    href: getLink({ page: "features", tab: "Nodes" }),
  },
  Skills: {
    icon: "docs",
    href: getLink({ page: "features", tab: "Skills" }),
  },
  Schedule: {
    icon: "schedule",
    href: getLink({ page: "features", tab: "Schedule" }),
  },
  Features: {
    icon: "stacks",
    href: getLink({ page: "features", tab: "Rules" }),
  },
};

const feature = {
  Rules: "deployed_code_account",
  Knowledge: "book_2",
  Notes: "note_stack",
  Skills: "docs",
  Schedule: "schedule",
};

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
