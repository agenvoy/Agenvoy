document.addEventListener("DOMContentLoaded", function () {
  const config = readConfig();
  let params = praseURL();

  if (params.page == null) {
    params.page = "chat";
  }

  if (params.page == "config") {
    const tabs = Object.keys(configTab);
    const matched = tabs.find((name) => name.toLowerCase() === String(params.tab || "").toLowerCase());
    params.tab = matched || tabs[0];
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
      configTab: configTab,
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
      reasoning_change: function (e) {
        saveSessionReasoning(currentSessionId, e.target.value);
      },
      memory_summary: function () {
        memorySummary();
      },
      memory_compact: function () {
        memoryCompact();
      },
      memory_reset: function () {
        memoryReset();
      },
      model_add: function () {
        selectProviderAdd();
      },
      model_routing: function () {
        selectModelRouting();
      },
      mcp_reset: function () {
        resetMcp();
      },
      mcp_save: function () {
        saveMcp();
      },
      mcp_delete: function () {
        deleteEditingMcp();
      },
      mcp_reconnect: function () {
        reconnectMcp();
      },
      mcp_login: function () {
        startMcpLogin(mcpEditing);
      },
      mcp_logout: function () {
        clearMcpOAuth(mcpEditing);
      },
      mcp_transport: function () {
        mcpTransportChange();
      },
      mcp_auth: function () {
        mcpAuthChange();
      },
      keychain_reset: function () {
        resetKeychain();
      },
      keychain_save: function () {
        saveKeychain();
      },
      keychain_delete: function () {
        deleteEditingKeychain();
      },
      channel_telegram: function () {
        selectChannel("telegram");
      },
      channel_discord: function () {
        selectChannel("discord");
      },
      channel_line: function () {
        selectChannel("line");
      },
      channel_admin: function () {
        selectChannel("admin");
      },
      channel_enable: function () {
        enableChannel();
      },
      channel_disable: function () {
        disableChannel();
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

        if (!(params.page === "chat" && params.chat)) {
          subscribeDaemonLog();
        }
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
          getReasoningList(params.chat);
          getRuleList();
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

        if (params.page === "config") {
          if (params.tab === "Model") {
            renderModel();
          }
          if (params.tab === "MCP") {
            resetMcp();
            renderMcp();
          }
          if (params.tab === "Keychain") {
            resetKeychain();
            renderKeychain();
          }
          if (params.tab === "Channel") {
            renderChannel();
          }
        }

        if (params.page === "features") {
          const kind = { Rules: "rule", Knowledge: "knowledge" }[params.tab];
          if (kind) {
            resetFeature(kind);
            renderFeature(kind);
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
