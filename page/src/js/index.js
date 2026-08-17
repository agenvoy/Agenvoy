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
        const mark = function (state) {
          if (state) {
            dom.dataset.selected = "1";
          } else {
            delete dom.dataset.selected;
          }
          config.harness_enable = state;
          writeConfig(config);
        };

        mark(on);
        toggleVoice(on).catch(function () {
          mark(false);
        });
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
      workdir_pick: function () {
        openWorkDirPrompt();
      },
      skill_pick: function () {
        openSkillPicker();
      },
      knowledge_pick: function () {
        openKnowledgePicker();
      },
      rule_save: function () {
        saveFeature("rule");
      },
      rule_reset: function () {
        resetFeature("rule");
      },
      rule_delete: function () {
        deleteEditing("rule");
      },
      knowledge_save: function () {
        saveFeature("knowledge");
      },
      knowledge_reset: function () {
        resetFeature("knowledge");
      },
      knowledge_delete: function () {
        deleteEditing("knowledge");
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
        bindChatMenu();
        renderChatList();

        if (params.page === "chat") {
          if (!params.chat) {
            clearChatDraft();
          }
          setSession(params.chat);
          subscribe(params.chat);
          getModelList(params.chat);
          getRuleList();
          renderKnowledgeMark();
          renderWorkDirMark();
          renderChat(params.chat);
          loadPending(params.chat);

          const harness = $("section.chat button.harness");
          if (config.harness_enable) {
            if (harness) {
              harness.dataset.selected = "1";
            }
            initVoice().catch(function (err) {
              console.error("voice: start deferred to the first gesture", err?.message || err);
              document.addEventListener(
                "pointerdown",
                function () {
                  initVoice().catch(function (retry) {
                    console.error("voice", retry?.message || retry);
                  });
                },
                { once: true },
              );
            });
          }
        }

        if (params.page === "features") {
          const kind = { Rules: "rule", Knowledge: "knowledge" }[params.tab];
          if (kind) {
            resetFeature(kind);
            renderFeature(kind);
          }

          for (const name of Object.keys(FEATURE_SPEC)) {
            if (name !== kind) {
              countFeature(name);
            }
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
