const LeftTab = {
  "New Chat": {
    icon: "add_comment",
    href: GetLink({ page: "chat" }),
  },
  Nodes: {
    icon: "note_stack",
    href: GetLink({ page: "features", tab: "Nodes" }),
  },
  Skills: {
    icon: "docs",
    href: GetLink({ page: "features", tab: "Skills" }),
  },
  Schedule: {
    icon: "schedule",
    href: GetLink({ page: "features", tab: "Schedule" }),
  },
  Features: {
    icon: "stacks",
    href: GetLink({ page: "features", tab: "Rules" }),
  },
};

const Feature = {
  Rules: "deployed_code_account",
  Knowledge: "book_2",
  Notes: "note_stack",
  Skills: "docs",
  Schedule: "schedule",
};

function GetLink(params) {
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
