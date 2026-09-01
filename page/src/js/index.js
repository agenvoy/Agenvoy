const LEFT_TAB_WIDE = 1280;

document.addEventListener("DOMContentLoaded", function () {
  const config = readConfig();
  const isWide = () => document.documentElement.clientWidth >= LEFT_TAB_WIDE;
  let params = praseURL();

  if (params.page == null) {
    params.page = "chat";
  }

  if (params.page == "config") {
    const tabs = Object.keys(configTab);
    const matched = tabs.find((name) => name.toLowerCase() === String(params.tab || "").toLowerCase());
    params.tab = matched || tabs[0];
  }

  if (params.page == "monitor") {
    const tabs = Object.keys(monitorTab).concat(["Details"]);
    const matched = tabs.find((name) => name.toLowerCase() === String(params.tab || "").toLowerCase());
    params.tab = matched || tabs[0];

    const periods = Object.keys(usageTab);
    const period = periods.find((name) => name.toLowerCase() === String(params.period || "").toLowerCase());
    params.period = period || periods[0];
  }

  if (params.chat) {
    if (!SESSION_ID.test(params.chat)) {
      window.location.href = getLink({ page: params.page, tab: params.tab });
      return;
    }
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
      collapsed: isWide() ? config.left_tab_collapsed : "1",
      left_tab: leftTab,
      feature: feature,
      configTab: configTab,
      monitorTab: monitorTab,
      usageTab: usageTab,
    },
    event: {
      show_tab: function () {
        const dom = $(".left-tab");
        const collapsed = dom.dataset.collapsed === "1" ? "0" : "1";
        dom.dataset.collapsed = collapsed;
        if (!isWide()) {
          return;
        }
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
      persona_edit: function () {
        openPersonaPopup();
      },
      usage_open: function () {
        window.location.href = usageLink("24h", currentSessionId);
      },
      history_open: function () {
        window.location.href = historyLink(currentSessionId, 0);
      },
      history_range: function () {
        historySubmit();
      },
      history_keydown: function (e) {
        if (e.key === "Enter") {
          e.preventDefault();
          historySubmit();
        }
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
      cron_save: function () {
        commitSchedule("cron");
      },
      cron_reset: function () {
        resetSchedule("cron");
      },
      cron_delete: function () {
        deleteEditingSchedule("cron");
      },
      cron_test: function () {
        testSchedule("cron");
      },
      task_save: function () {
        commitSchedule("task");
      },
      task_reset: function () {
        resetSchedule("task");
      },
      task_delete: function () {
        deleteEditingSchedule("task");
      },
      task_test: function () {
        testSchedule("task");
      },
      skill_config: function () {
        openSkillConfig();
      },
      skill_delete: function () {
        deleteSkillTab();
      },
      skill_open: function () {
        openSkillFolder();
      },
      rule_pick: function () {
        openRulePicker();
      },
      model_pick: function () {
        openModelPicker();
      },
      reasoning_pick: function () {
        openReasoningPicker();
      },
      memory_pick: function () {
        openMemoryPicker();
      },
      resume_pick: function () {
        openResumePicker();
      },
      model_add: function () {
        selectProviderAdd();
      },
      model_routing: function () {
        selectModelRouting();
      },
      model_filter: function (e) {
        modelFilterChange(e);
      },
      daemon_range: function () {
        daemonSubmit();
      },
      daemon_keydown: function (e) {
        if (e.key === "Enter") {
          e.preventDefault();
          daemonSubmit();
        }
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
      mcp_transport: function (e) {
        mcpTransportChange(e);
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

          if (dom.dataset.collapsed === "1" || vw >= LEFT_TAB_WIDE) {
            return;
          }
          dom.dataset.collapsed = "1";
        });

        if (!(params.page === "chat" && params.chat)) {
          subscribe("");
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
          renderResumeMark(params.chat);

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

        if (params.page === "monitor") {
          if (params.tab === "Usage") {
            renderUsagePage(params.chat);
          }
          if (params.tab === "History") {
            renderHistoryPage(params.target || "", Number(params.offset) || 0);
          }
          if (params.tab === "Lessons") {
            renderLessonPage(params.target || "", Number(params.offset) || 0, params.outcome || "");
          }
          if (params.tab === "Details") {
            renderDetailsPage(params.target || "", params.hash || "", params.item || "");
          }
          if (params.tab === "Daemon") {
            renderDaemonPage();
          }
        }

        if (params.page === "features") {
          const kind = { Rules: "rule", Knowledge: "knowledge" }[params.tab];
          if (kind) {
            resetFeature(kind);
            renderFeature(kind);
          }
          if (params.tab === "Skills") {
            renderSkillTab();
            if (!params.target) {
              openSkillConfig();
            }
          }
          const schedule = { Cron: "cron", Task: "task" }[params.tab];
          if (schedule) {
            resetSchedule(schedule);
            renderSchedule(schedule);
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
