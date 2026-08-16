document.addEventListener("DOMContentLoaded", function () {
  const config = readConfig();
  let params = praseURL();

  if (params.page == null) {
    params.page = "chat";
  }

  if (params.chat != null && !CHAT_ID.test(params.chat || "")) {
    window.location.href = getLink({ page: params.page });
    return;
  }
  params.chat = params.chat || "";

  console.log("config", config);
  console.log("params", params);

  function submit() {
    const dom = $("#chat-input");
    const content = dom.value;
    dom.value = "";
    dom.nextElementSibling.textContent = "\n";
    send(content);
  }

  function wheelDelta(e, target) {
    if (e.deltaMode === 1) return e.deltaY * 16;
    if (e.deltaMode === 2) return e.deltaY * target.clientHeight;
    return e.deltaY;
  }

  const app = new QUI({
    id: "app",
    data: {
      params: params,
      collapsed: config.left_tab_collapsed,
      left_tab: leftTab,
      feature: feature,
    },
    event: {
      show_tab: function () {
        const dom = $(".left-tab");
        const collapsed = dom.dataset.collapsed === "1" ? "0" : "1";
        dom.dataset.collapsed = collapsed;
        config.left_tab_collapsed = collapsed;
        writeConfig(config);
      },
      feature_tab_click: function (e) {
        const header = e.target.closest("header");
        const name = e.target.name;
        header.dataset.selected = name;
      },
      chat_input: function () {
        this.nextElementSibling.textContent = this.value + "\n";
      },
      chat_keydown: function (e) {
        if (e.key !== "Enter" || e.shiftKey || e.isComposing) {
          return;
        }
        e.preventDefault();
        submit();
      },
      send_click: function () {
        submit();
      },
      harness_click: function (e) {
        const dom = e.target.closest("button");
        if (!dom) return;

        const on = dom.dataset.selected == null;
        if (on) {
          dom.dataset.selected = "1";
        } else {
          delete dom.dataset.selected;
        }
        config.harness_enable = on;
        writeConfig(config);
        toggleVoice(on);
      },
      chat_wheel: function (e) {
        const dom = $("#right-content-chat-messages");
        if (dom.contains(e.target) || dom.scrollHeight <= dom.clientHeight) {
          return;
        }

        const box = e.target.closest("textarea");
        if (box && box.scrollHeight > box.clientHeight) {
          return;
        }
        dom.scrollTop += wheelDelta(e, dom);
        e.preventDefault();
      },
      rule_change: function (e) {
        selectRule(e.target.value);
      },
      model_change: function (e) {
        const model = e.target.value;
        if (!model || !currentSessionId) {
          return;
        }
        saveSessionModel(currentSessionId, model);
      },
    },
    when: {
      before_render: function () {
        // 停止渲染
      },
      rendered: function () {
        setTimeout(() => {
          document.body.dataset.rendered = "1";
        }, 0);

        window.addEventListener("resize", function (e) {
          const vw = document.documentElement.clientWidth;
          const dom = $(".left-tab");

          let timer;
          clearTimeout(timer);
          document.body.dataset.rendered = "0";
          timer = setTimeout(() => {
            document.body.dataset.rendered = "1";
            clearTimeout(timer);
          }, 500);

          if (dom.dataset.collapsed === "1" || vw >= 768) {
            return;
          }
          dom.dataset.collapsed = "1";
        });

        bindSelectPicker();
        bindInputDrop();
        renderChatList();

        if (params.page === "chat") {
          setSession(params.chat);
          subscribe(params.chat);
          getModelList(params.chat);
          getRuleList();
          renderChat(params.chat);
          loadPending(params.chat);

          const harness = $("section.chat button.harness");
          if (harness && config.harness_enable) {
            harness.dataset.selected = "1";
          }
          if (config.harness_enable) {
            initVoice();
          }
        }
      },
      before_update: function () {
        // 停止更新
      },
      updated: function () {
        // 已更新
      },
      before_destroy: function () {
        // 停止銷毀
      },
      destroyed: function () {
        // 已銷毀
      },
    },
  });
});
