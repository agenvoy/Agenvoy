function LoadChats() {
  const chat_list = [
    {
      id: "123",
      title: "Chat 1",
    },
  ];

  for (let i = 0; i < chat_list.length; i++) {
    document
      .getElementById("left-tab-chat-list")
      .appendChild(
        _("a", { href: GetLink({ page: "chat", chat: chat_list[i].id }) }, [
          _("p", chat_list[i].title),
          _("span.material-symbols-outlined", "more_horiz"),
        ]),
      );
  }
}

function LoadChat(list) {
  const dom = document.getElementById("right-content-chat-messages");

  for (let i = 0; i < list.length; i++) {
    if (list[i].rule === "user") {
      dom.appendChild(chatRuleUser(list[i]));
    } else if (list[i].rule === "assistant") {
      dom.appendChild(chatRuleAssistant(list[i]));
    }
  }
}

function chatRuleUser(item) {
  return _("div.user", [
    _("p", item.content),
    _("footer", [
      _("p", item.meta.send_at),
      _("span.material-symbols-outlined", "content_copy"),
      _("span.material-symbols-outlined", "cached"),
    ]),
  ]);
}

function chatRuleAssistant(item) {
  return _("div.assistant", [
    _("img", "public/logo-min.svg"),
    _("section", [
      _("p", item.meta.model),
      _("details", [
        _("summary", ["Reasoning", _("span.material-symbols-outlined", "keyboard_arrow_down")]),
        _("section.md-render", `<span class="md-line">我不確定你想表達什麼。請重新輸入問題或需求。</span>`),
      ]),
      _("section.md-render", `<span class="md-line">我不確定你想表達什麼。請重新輸入問題或需求。</span>`),
      _("footer", [
        _("span.material-symbols-outlined", "content_copy"),
        _("span.material-symbols-outlined", "cached"),
        _("div", [_("span.material-symbols-outlined", "arrow_upward_alt"), _("p", item.meta.input)]),
        _("div", [_("span.material-symbols-outlined", "arrow_downward_alt"), _("p", item.meta.output)]),
        _("p", item.meta.send_at),
      ]),
    ]),
  ]);
}
