document.addEventListener("DOMContentLoaded", function () {
  const config = ReadConfig();
  let params = PraseURL();

  if (params.page == null) {
    params.page = "chat";
  }

  if (params.chat == null) {
    params.chat = "";
  }

  console.log("config", config);
  console.log("params", params);

  const app = new QUI({
    id: "app",
    data: {
      params: params,
      collapsed: config.left_tab_collapsed,
      left_tab: LeftTab,
      feature: Feature,
    },
    event: {
      show_tab: function () {
        const dom = document.querySelector(".left-tab");
        const collapsed = dom.dataset.collapsed === "1" ? "0" : "1";
        dom.dataset.collapsed = collapsed;
        config.left_tab_collapsed = collapsed;
        WriteConfig(config);
      },
      chat_input: function () {
        this.nextElementSibling.textContent = this.value + "\n";
      },
      feature_tab_click: function (e) {
        const header = e.target.closest("header");
        const name = e.target.name;
        header.dataset.selected = name;
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
          const dom = document.querySelector(".left-tab");

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

        LoadChats();

        if (params.page == "chat" && params.chat != null) {
          LoadChat([
            {
              rule: "user",
              content: "asdfasfaf",
              meta: {
                send_at: "2026-08-08 11:02",
              },
            },
            {
              rule: "assistant",
              Reasoning: "我不確定你想表達什麼。請重新輸入問題或需求。",
              content: "我不確定你想表達什麼。請重新輸入問題或需求。",
              meta: {
                model: "auto",
                send_at: "2026-08-08 11:02",
                input: "12k",
                output: "12k",
              },
            },
          ]);
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
