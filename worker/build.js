const fs = require("fs");
const path = require("path");
const { marked } = require("marked");

const PAGES_DIR = path.join(__dirname, "public/docs/pages");
const OUT_DIR = path.join(__dirname, "public/docs");
const TAGS_MANIFEST = path.join(__dirname, "public/docs/tags/manifest.json");

function semverSort(a, b) {
  const pa = a.replace(/^v/, "").split(".").map(Number);
  const pb = b.replace(/^v/, "").split(".").map(Number);
  for (let i = 0; i < 3; i++) {
    if ((pa[i] || 0) !== (pb[i] || 0)) return (pb[i] || 0) - (pa[i] || 0);
  }
  return 0;
}

let LATEST_VERSION = "";
if (fs.existsSync(TAGS_MANIFEST)) {
  const tags = Object.keys(JSON.parse(fs.readFileSync(TAGS_MANIFEST, "utf-8"))).sort(semverSort);
  LATEST_VERSION = tags[0] || "";
}

const NAV = [
  { section: "Overview", items: [
    { slug: "home", label: "Home" },
    { slug: "getting-started", label: "Getting Started" },
  ]},
  { section: "Concepts", items: [
    { slug: "sessions", label: "Sessions & Agents" },
    { slug: "execution-engine", label: "Execution Engine" },
    { slug: "providers", label: "Providers" },
  ]},
  { section: "User Guide", items: [
    { slug: "cli-commands", label: "CLI Commands" },
    { slug: "tui-guide", label: "TUI Guide" },
    { slug: "rest-api", label: "REST API" },
    { slug: "config-files", label: "Configuration" },
    { slug: "config-integrations", label: "Integration Config" },
  ]},
  { section: "Tools", items: [
    { slug: "built-in-tools", label: "Built-in Tools" },
    { slug: "tool-extension", label: "Tool Extension" },
    { slug: "tool-rules", label: "Tool Design & Rules" },
  ]},
  { section: "Features", items: [
    { slug: "memory-system", label: "Memory System" },
    { slug: "skill-basics", label: "Skill System" },
    { slug: "scheduler-skills", label: "Scheduler & Self-Improvement" },
    { slug: "mcp-server", label: "MCP Server" },
    { slug: "mcp-client", label: "MCP Client" },
    { slug: "kuradb-rag", label: "KuraDB & RAG" },
  ]},
  { section: "Security", items: [
    { slug: "sandbox", label: "Sandbox" },
    { slug: "security", label: "Security Model" },
  ]},
  { section: "Reference", items: [
    { slug: "architecture", label: "Architecture" },
    { slug: "comparison", label: "Comparison" },
  ]},
];

const DESCRIPTIONS = {
  "home": "Agenvoy documentation — a personal AI agent that runs on your machine. Guides for setup, tools, MCP, memory, scheduling, and security.",
  "getting-started": "Install Agenvoy and run your first AI agent session in under 60 seconds. One command setup for macOS and Linux.",
  "sessions": "Manage sessions, agent personas, routing rules, and per-session concurrency in Agenvoy.",
  "execution-engine": "How the Agenvoy iteration loop, three-pass tool dispatch, and circuit breaker work under the hood.",
  "providers": "Configure 10 LLM providers — Claude, OpenAI, Gemini, Codex, Copilot, xAI Grok, DeepSeek, Nvidia NIM, OpenRouter, and Compat.",
  "cli-commands": "Agenvoy CLI commands, make shortcuts, input prefixes, and environment variables reference.",
  "tui-guide": "Agenvoy TUI keyboard shortcuts, slash commands, and interactive session management.",
  "rest-api": "OpenAI-compatible REST API endpoints for chat completions, sessions, and log replay.",
  "config-files": "Agenvoy configuration file layout — config.json, bot.md, permission modes, and runtime limits.",
  "config-integrations": "Configure MCP servers, LLM providers, KuraDB, Telegram bot, and Discord bot integrations.",
  "built-in-tools": "60+ built-in AI agent tools — file operations, web search, orchestration, memory, RAG, and media.",
  "tool-extension": "Auto-generate custom tools from natural language. Add script, API, or MCP tools to extend Agenvoy.",
  "tool-rules": "Tool design guidelines — concurrency markers, timeouts, credential auto-heal, and naming conventions.",
  "memory-system": "Three-tier conversation memory — rolling context window, semantic vector search, and FTS5 SQLite archive.",
  "skill-basics": "Create loadable markdown skill packs with YAML frontmatter, slash-command and natural-language triggers.",
  "scheduler-skills": "Cron and one-shot task scheduling with skill binding, hot-reload, and auto-fix on failure.",
  "mcp-server": "Run Agenvoy as an MCP server — expose sandboxed tools to Claude Code, Codex, Cursor, and OpenCode.",
  "mcp-client": "Connect Agenvoy to external MCP servers via stdio or HTTP/SSE with auto-discovery and hot-reload.",
  "kuradb-rag": "Enable KuraDB for keyword and semantic document search (RAG) over your personal knowledge base.",
  "sandbox": "OS-native command sandbox — bubblewrap on Linux, sandbox-exec on macOS. CPU, memory, and network limits.",
  "security": "Agenvoy security model — permission modes, macOS Keychain, system prompt protection, and MCP isolation.",
  "architecture": "Agenvoy architecture — system layers, daemon lifecycle, cross-cutting principles, and TUI design.",
  "comparison": "Compare Agenvoy vs Claude Code, Codex CLI, Cursor, Aider, and other AI agent platforms.",
};

const KEYWORDS = {
  "home": "agenvoy, ai agent, documentation, mcp server, tool builder, personal ai assistant",
  "getting-started": "agenvoy install, setup, getting started, curl install, macos, linux, ai agent setup",
  "sessions": "agenvoy sessions, agent personas, session routing, multi-session, concurrency",
  "execution-engine": "execution engine, tool dispatch, iteration loop, circuit breaker, three-pass dispatch",
  "providers": "llm providers, claude, openai, gemini, codex, copilot, grok, deepseek, nvidia nim, openrouter, multi-model",
  "cli-commands": "agenvoy cli, agen command, cli reference, terminal commands, make shortcuts",
  "tui-guide": "agenvoy tui, terminal ui, keyboard shortcuts, slash commands, interactive agent",
  "rest-api": "agenvoy api, rest api, chat completions, openai compatible, session api",
  "config-files": "agenvoy config, configuration, config.json, bot.md, permission modes, runtime limits",
  "config-integrations": "agenvoy integrations, mcp config, telegram bot setup, discord bot setup, kuradb config",
  "built-in-tools": "agenvoy tools, built-in tools, file tools, web search, orchestration, memory tools, ai tools",
  "tool-extension": "custom tools, tool generation, auto tool builder, script tools, api tools, mcp tools",
  "tool-rules": "tool design, tool naming, concurrency, timeouts, credential management, tool conventions",
  "memory-system": "ai memory, conversation memory, semantic search, vector search, fts5, sqlite, context window",
  "skill-basics": "agenvoy skills, skill system, markdown skills, yaml frontmatter, slash commands, skill triggers",
  "scheduler-skills": "agenvoy scheduler, cron jobs, one-shot tasks, skill binding, auto-fix, self-improvement",
  "mcp-server": "mcp server, model context protocol, claude code mcp, codex mcp, tool sharing, stdio server",
  "mcp-client": "mcp client, external mcp, stdio client, http sse, mcp tools, auto-discovery",
  "kuradb-rag": "kuradb, rag, retrieval augmented generation, document search, semantic search, knowledge base",
  "sandbox": "sandbox, bubblewrap, bwrap, sandbox-exec, macos sandbox, linux sandbox, command isolation",
  "security": "agenvoy security, permission modes, keychain, system prompt protection, mcp isolation, tool confirmation",
  "architecture": "agenvoy architecture, system design, daemon, go binary, tui design, internal structure",
  "comparison": "agenvoy vs, claude code, codex cli, cursor, aider, ai agent comparison, alternative",
};

const PRIORITIES = {
  "home": 0.9,
  "getting-started": 0.85,
  "built-in-tools": 0.8,
  "mcp-server": 0.8,
  "providers": 0.75,
  "tool-extension": 0.75,
  "cli-commands": 0.7,
  "sessions": 0.7,
  "memory-system": 0.7,
  "skill-basics": 0.7,
  "mcp-client": 0.7,
  "comparison": 0.7,
};

const NAV_ZH_SECTION = {
  "Overview": "總覽", "Concepts": "概念", "User Guide": "使用指南",
  "Tools": "工具", "Features": "功能", "Security": "安全", "Reference": "參考",
};

const NAV_ZH_LABEL = {
  "home": "首頁", "getting-started": "快速開始", "sessions": "Session 與 Agent",
  "execution-engine": "執行引擎", "providers": "供應商", "cli-commands": "CLI 指令",
  "tui-guide": "TUI 指南", "rest-api": "REST API", "config-files": "設定檔",
  "config-integrations": "整合設定", "built-in-tools": "內建工具", "tool-extension": "工具擴充",
  "tool-rules": "工具設計與規範", "memory-system": "記憶系統", "skill-basics": "Skill 系統",
  "scheduler-skills": "排程與自我修正", "mcp-server": "MCP 伺服器", "mcp-client": "MCP 客戶端",
  "kuradb-rag": "KuraDB 與 RAG", "sandbox": "沙箱", "security": "安全模型",
  "architecture": "架構", "comparison": "比較",
};

const DESCRIPTIONS_ZH = {
  "home": "Agenvoy 文件 —— 跑在你機器上的專屬 AI agent。涵蓋安裝、工具、MCP、記憶、排程與安全的指南。",
  "getting-started": "60 秒內安裝 Agenvoy 並啟動第一個 AI agent session。macOS 與 Linux 一行指令完成設定。",
  "sessions": "在 Agenvoy 管理 session、agent 人格、路由規則與每個 session 的並行度。",
  "execution-engine": "深入了解 Agenvoy 的迭代迴圈、三段式工具調度與斷路器如何運作。",
  "providers": "設定 10 家 LLM 供應商 —— Claude、OpenAI、Gemini、Codex、Copilot、xAI Grok、DeepSeek、Nvidia NIM、OpenRouter 與 Compat。",
  "cli-commands": "Agenvoy CLI 指令、make 捷徑、輸入前綴與環境變數參考。",
  "tui-guide": "Agenvoy TUI 鍵盤快捷鍵、斜線指令與互動式 session 管理。",
  "rest-api": "相容 OpenAI 的 REST API 端點：chat completions、session 與 log replay。",
  "config-files": "Agenvoy 設定檔結構 —— config.json、bot.md、權限模式與 runtime 限制。",
  "config-integrations": "設定 MCP 伺服器、LLM 供應商、KuraDB、Telegram 與 Discord 機器人整合。",
  "built-in-tools": "60+ 內建 AI agent 工具 —— 檔案操作、網頁搜尋、編排、記憶、RAG 與媒體。",
  "tool-extension": "用自然語言自動生成自訂工具。新增 script、API 或 MCP 工具來擴充 Agenvoy。",
  "tool-rules": "工具設計準則 —— 並行標記、逾時、憑證自動修復與命名慣例。",
  "memory-system": "三層對話記憶 —— 滾動 context window、語意向量搜尋與 FTS5 SQLite 封存。",
  "skill-basics": "建立可載入的 markdown skill 包，搭配 YAML frontmatter、斜線指令與自然語言觸發。",
  "scheduler-skills": "Cron 與一次性任務排程，可綁定 skill、熱重載並在失敗時自動修復。",
  "mcp-server": "把 Agenvoy 當成 MCP 伺服器 —— 將沙箱工具開放給 Claude Code、Codex、Cursor 與 OpenCode。",
  "mcp-client": "透過 stdio 或 HTTP/SSE 將 Agenvoy 連接到外部 MCP 伺服器，支援自動探索與熱重載。",
  "kuradb-rag": "啟用 KuraDB，在你的個人知識庫上進行關鍵字與語意文件搜尋（RAG）。",
  "sandbox": "OS 原生指令沙箱 —— Linux 用 bubblewrap、macOS 用 sandbox-exec。可限制 CPU、記憶體與網路。",
  "security": "Agenvoy 安全模型 —— 權限模式、macOS Keychain、system prompt 保護與 MCP 隔離。",
  "architecture": "Agenvoy 架構 —— 系統分層、daemon 生命週期、跨領域原則與 TUI 設計。",
  "comparison": "比較 Agenvoy 與 Claude Code、Codex CLI、Cursor、Aider 等 AI agent 平台。",
};

function slugify(text) {
  // keep CJK ranges so Chinese headings produce usable anchor ids (empty ids break the TOC)
  return text.toLowerCase()
    .replace(/[^\w\s一-鿿㐀-䶿豈-﫿-]/g, "")
    .replace(/\s+/g, "-").replace(/-+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function buildSidebar(activeSlug, lang = "en") {
  const isZh = lang === "zh";
  const base = isZh ? "/zh/docs" : "/docs";
  let html = "";
  for (const group of NAV) {
    const section = isZh ? (NAV_ZH_SECTION[group.section] || group.section) : group.section;
    html += `<div class="nav-divider"></div>\n`;
    html += `<div class="nav-section">${section}</div>\n`;
    for (const item of group.items) {
      const cls = item.slug === activeSlug ? " active" : "";
      const href = item.slug === "home" ? `${base}/` : `${base}/${item.slug}`;
      const label = isZh ? (NAV_ZH_LABEL[item.slug] || item.label) : item.label;
      html += `<a class="nav-item${cls}" href="${href}">${label}</a>\n`;
    }
  }
  html += `<div class="nav-divider"></div>\n`;
  html += `<a class="nav-item" href="/docs/released/">${isZh ? "版本紀錄" : "Released"}</a>\n`;
  return html.replace(/^<div class="nav-divider"><\/div>\n/, "");
}

function buildTOC(html, lang = "en") {
  const tocTitle = lang === "zh" ? "本頁內容" : "On this page";
  const headings = [];
  const regex = /<h([23])[^>]*id="([^"]*)"[^>]*>(.*?)<\/h\1>/g;
  let m;
  while ((m = regex.exec(html)) !== null) {
    headings.push({ depth: parseInt(m[1]), id: m[2], text: m[3].replace(/<[^>]+>/g, "") });
  }
  if (!headings.length) return `<div class="toc-title">${tocTitle}</div>`;
  let toc = `<div class="toc-title">${tocTitle}</div>\n`;
  for (const h of headings) {
    const cls = h.depth === 3 ? " depth-3" : "";
    toc += `<a class="toc-link${cls}" href="#${h.id}">${h.text}</a>\n`;
  }
  return toc;
}

function addHeadingIds(html) {
  return html.replace(/<h([1-4])>(.*?)<\/h\1>/g, (match, level, text) => {
    const id = slugify(text.replace(/<[^>]+>/g, ""));
    return `<h${level} id="${id}">${text}</h${level}>`;
  });
}

// wrap every table so it scrolls horizontally instead of overflowing the viewport
function wrapTables(html) {
  return html
    .replace(/<table>/g, '<div class="table-scroll"><table>')
    .replace(/<\/table>/g, "</table></div>");
}

function renderPage(slug, title, description, keywords, sidebar, content, toc, lang = "en") {
  const isZh = lang === "zh";
  const isReleased = slug === "released" || slug.startsWith("released/");
  const base = isZh ? "https://agenvoy.com/zh" : "https://agenvoy.com";
  const canonical = slug === "home" ? `${base}/docs/` : `${base}/docs/${slug}`;
  const fullTitle = isZh ? `${title} - Agenvoy 文件` : `${title} - Agenvoy Docs`;

  // language toggle target (counterpart page); released has no zh copy → fall back to zh docs home
  let altHref;
  if (isZh) altHref = slug === "home" ? "/docs/" : `/docs/${slug}`;
  else if (isReleased) altHref = "/zh/docs/";
  else altHref = slug === "home" ? "/zh/docs/" : `/zh/docs/${slug}`;

  const enUrl = slug === "home" ? "https://agenvoy.com/docs/" : `https://agenvoy.com/docs/${slug}`;
  const zhUrl = slug === "home" ? "https://agenvoy.com/zh/docs/" : `https://agenvoy.com/zh/docs/${slug}`;
  const altLinks = isReleased ? "" :
    `<link rel="alternate" hreflang="en" href="${enUrl}" />
    <link rel="alternate" hreflang="zh-Hant" href="${zhUrl}" />
    <link rel="alternate" hreflang="x-default" href="${enUrl}" />
    `;

  const jsonLd = JSON.stringify({
    "@context": "https://schema.org",
    "@type": "TechArticle",
    "headline": fullTitle,
    "description": description,
    "url": canonical,
    "inLanguage": isZh ? "zh-Hant" : "en",
    "isPartOf": { "@type": "WebSite", "name": "Agenvoy", "url": "https://agenvoy.com/" },
    "publisher": { "@type": "Person", "name": "Pardn Chiu", "url": "https://pardn.io/" },
    "image": "https://agenvoy.com/logo-min.svg",
    "dateModified": new Date().toISOString().split("T")[0],
  });
  return `<!doctype html>
<html lang="${isZh ? "zh-Hant" : "en"}">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta name="robots" content="index, follow" />
    <title>${fullTitle}</title>
    <meta name="title" content="${fullTitle}" />
    <meta name="description" content="${description}" />
    <meta name="keywords" content="${keywords}" />
    <meta name="author" content="Pardn Chiu" />
    <link rel="author" href="https://pardn.io/" />
    <link rel="icon" href="/logo-min.svg" type="image/svg+xml" />
    <link rel="canonical" href="${canonical}" />
    ${altLinks}<meta property="og:title" content="${fullTitle}" />
    <meta property="og:description" content="${description}" />
    <meta property="og:image" content="https://agenvoy.com/logo-min.svg" />
    <meta property="og:url" content="${canonical}" />
    <meta property="og:type" content="article" />
    <meta property="og:site_name" content="Agenvoy" />
    <meta property="og:locale" content="${isZh ? "zh_TW" : "en_US"}" />
    <meta name="twitter:card" content="summary" />
    <meta name="twitter:title" content="${fullTitle}" />
    <meta name="twitter:description" content="${description}" />
    <meta name="twitter:image" content="https://agenvoy.com/logo-min.svg" />
    <script type="application/ld+json">${jsonLd}</script>
    <script async src="https://www.googletagmanager.com/gtag/js?id=G-L5VYEZPVXX"></script>
    <script>window.dataLayer=window.dataLayer||[];function gtag(){dataLayer.push(arguments)}gtag("js",new Date());gtag("config","G-L5VYEZPVXX");</script>
    <link rel="preconnect" href="https://fonts.googleapis.com" />
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet" />
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.2/css/all.min.css" referrerpolicy="no-referrer" />
    <link rel="stylesheet" href="/docs.css" />
  </head>
  <body>
    <header class="header">
      <button class="mobile-menu-btn" onclick="document.querySelector('.sidebar').classList.toggle('open')" aria-label="Menu"><i class="fa-solid fa-bars"></i></button>
      <a href="${isZh ? "/zh/" : "/"}" class="header-logo"><picture><source media="(max-width: 480px)" srcset="/logo-min.svg" /><img src="/logo-text.svg" alt="Agenvoy" /></picture></a>
      <span class="header-sep"></span>
      <span class="header-title">${isZh ? "文件" : "Documentation"}</span>
      ${LATEST_VERSION ? `<a class="header-version" href="https://github.com/pardnchiu/agenvoy/releases/tag/${LATEST_VERSION}" target="_blank" rel="noopener">${LATEST_VERSION}</a>` : ""}
      <div class="header-links">
        <a href="${isZh ? "/zh/" : "/"}">${isZh ? "首頁" : "Home"}</a>
        <a href="https://github.com/pardnchiu/agenvoy" target="_blank" rel="noopener">GitHub</a>
      </div>
    </header>
    <div class="layout">
      <nav class="sidebar">${sidebar}</nav>
      <main class="content">${content}</main>
      <aside class="toc">${toc}</aside>
    </div>
    <script>
      document.querySelectorAll('.sidebar .nav-item').forEach(function(el){
        el.addEventListener('click',function(){document.querySelector('.sidebar').classList.remove('open')})
      });
      var tocObs=new IntersectionObserver(function(entries){
        entries.forEach(function(e){
          if(e.isIntersecting){
            document.querySelectorAll('.toc-link').forEach(function(l){
              l.classList.toggle('active',l.getAttribute('href')==='#'+e.target.id)
            })
          }
        })
      },{rootMargin:'-80px 0px -70% 0px'});
      document.querySelectorAll('.content h2,.content h3').forEach(function(h){tocObs.observe(h)});
    </script>
    ${isReleased ? "" : `<a href="${altHref}" class="lang-fab" aria-label="${isZh ? "Switch to English" : "Switch to Chinese"}" hreflang="${isZh ? "en" : "zh-Hant"}"><i class="fa-solid fa-language"></i><span>${isZh ? "EN" : "中文"}</span></a>`}
  </body>
</html>`;
}

marked.setOptions({ gfm: true, breaks: false });

const ZH_DOCS_DIR = path.join(__dirname, "public/zh/docs");
const allSlugs = NAV.flatMap(g => g.items.map(i => i.slug));
let built = 0;
let builtZh = 0;
const zhSlugs = [];

for (const slug of allSlugs) {
  const mdPath = path.join(PAGES_DIR, `${slug}.md`);
  if (!fs.existsSync(mdPath)) {
    console.warn(`SKIP: ${slug}.md not found`);
    continue;
  }

  const md = fs.readFileSync(mdPath, "utf-8");
  let html = marked.parse(md);
  html = wrapTables(addHeadingIds(html));

  const label = NAV.flatMap(g => g.items).find(i => i.slug === slug)?.label || slug;
  const desc = DESCRIPTIONS[slug] || `${label} — Agenvoy documentation.`;
  const kw = KEYWORDS[slug] || "agenvoy, ai agent, documentation";
  const sidebar = buildSidebar(slug, "en");
  const toc = buildTOC(html);
  const page = renderPage(slug, label, desc, kw, sidebar, html, toc, "en");

  const outPath = slug === "home"
    ? path.join(OUT_DIR, "index.html")
    : path.join(OUT_DIR, `${slug}.html`);

  fs.writeFileSync(outPath, page);
  built++;
  console.log(`OK: ${outPath}`);

  // zh variant — generated only when a translated source exists
  const zhMdPath = path.join(PAGES_DIR, `${slug}.zh.md`);
  if (fs.existsSync(zhMdPath)) {
    let zhHtml = wrapTables(addHeadingIds(marked.parse(fs.readFileSync(zhMdPath, "utf-8"))));
    const zhLabel = NAV_ZH_LABEL[slug] || label;
    const zhDesc = DESCRIPTIONS_ZH[slug] || desc;
    const zhSidebar = buildSidebar(slug, "zh");
    const zhToc = buildTOC(zhHtml, "zh");
    const zhPage = renderPage(slug, zhLabel, zhDesc, kw, zhSidebar, zhHtml, zhToc, "zh");
    const zhOut = slug === "home"
      ? path.join(ZH_DOCS_DIR, "index.html")
      : path.join(ZH_DOCS_DIR, `${slug}.html`);
    fs.mkdirSync(path.dirname(zhOut), { recursive: true });
    fs.writeFileSync(zhOut, zhPage);
    builtZh++;
    zhSlugs.push(slug);
    console.log(`OK: ${zhOut}`);
  }
}

// === Release pages ===
const TAGS_SRC = path.join(__dirname, "public/docs/tags");
const RELEASED_DIR = path.join(OUT_DIR, "released");
const releaseTags = [];

function buildVersionSidebar(activeTag, tags, dates) {
  let html = '<a class="nav-item" href="/docs/">Documentation</a>\n';
  html += '<div class="nav-divider"></div>\n';
  const groups = new Map();
  for (const t of tags) {
    const p = t.replace(/^v/, "").split(".");
    const key = `v${p[0]}.${p[1]}`;
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(t);
  }
  for (const [minor, versions] of groups) {
    html += `<div class="nav-section">${minor}</div>\n`;
    for (const v of versions) {
      const cls = v === activeTag ? " active" : "";
      const date = dates[v] ? `<span class="nav-date">${dates[v]}</span>` : "";
      html += `<a class="nav-item${cls}" href="/docs/released/${v}">${v}${date}</a>\n`;
    }
  }
  return html;
}

if (fs.existsSync(TAGS_SRC)) {
  const tagFiles = fs.readdirSync(TAGS_SRC).filter(f => f.endsWith(".md"));
  const tags = tagFiles.map(f => f.replace(".md", "")).sort(semverSort);
  const manifestPath = path.join(TAGS_SRC, "manifest.json");
  const dates = fs.existsSync(manifestPath) ? JSON.parse(fs.readFileSync(manifestPath, "utf-8")) : {};

  if (tags.length) {
    fs.mkdirSync(RELEASED_DIR, { recursive: true });

    for (const tag of tags) {
      const md = fs.readFileSync(path.join(TAGS_SRC, `${tag}.md`), "utf-8");
      let html = marked.parse(md);
      html = wrapTables(addHeadingIds(html));
      const sidebar = buildVersionSidebar(tag, tags, dates);
      const toc = buildTOC(html);
      const desc = `Agenvoy ${tag} release notes — changelog, new features, and fixes.`;
      const kw = `agenvoy, release notes, changelog, ${tag}`;
      const page = renderPage(`released/${tag}`, `${tag} Release Notes`, desc, kw, sidebar, html, toc);
      fs.writeFileSync(path.join(RELEASED_DIR, `${tag}.html`), page);
      releaseTags.push(tag);
    }

    // Index page
    let listHtml = '<h1>Release Notes</h1>\n<p>All Agenvoy releases.</p>\n';
    const groups = new Map();
    for (const t of tags) {
      const p = t.replace(/^v/, "").split(".");
      const key = `v${p[0]}.${p[1]}`;
      if (!groups.has(key)) groups.set(key, []);
      groups.get(key).push(t);
    }
    for (const [minor, versions] of groups) {
      listHtml += `<h2>${minor}</h2>\n<ul>\n`;
      for (const v of versions) {
        const date = dates[v] ? ` <span style="color:var(--muted);font-size:13px">${dates[v]}</span>` : "";
        listHtml += `<li><a href="/docs/released/${v}">${v}</a>${date}</li>\n`;
      }
      listHtml += '</ul>\n';
    }
    listHtml = addHeadingIds(listHtml);
    const indexSidebar = buildVersionSidebar("", tags, dates);
    const indexToc = buildTOC(listHtml);
    const indexPage = renderPage("released", "Release Notes", "All Agenvoy release notes — changelogs, features, and fixes by version.", "agenvoy, releases, changelog, version history", indexSidebar, listHtml, indexToc);
    fs.writeFileSync(path.join(RELEASED_DIR, "index.html"), indexPage);

    console.log(`OK: ${releaseTags.length} release pages + index`);
  }
}

// Generate sitemap.xml
const today = new Date().toISOString().split("T")[0];
let sitemap = `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n`;
sitemap += `  <url><loc>https://agenvoy.com/</loc><changefreq>weekly</changefreq><priority>1.0</priority><lastmod>${today}</lastmod></url>\n`;
sitemap += `  <url><loc>https://agenvoy.com/docs/</loc><changefreq>weekly</changefreq><priority>0.9</priority><lastmod>${today}</lastmod></url>\n`;
for (const slug of allSlugs) {
  if (slug === "home") continue;
  const mdPath = path.join(PAGES_DIR, `${slug}.md`);
  if (!fs.existsSync(mdPath)) continue;
  const pri = PRIORITIES[slug] || 0.6;
  sitemap += `  <url><loc>https://agenvoy.com/docs/${slug}</loc><changefreq>monthly</changefreq><priority>${pri}</priority><lastmod>${today}</lastmod></url>\n`;
}
// zh mirror
sitemap += `  <url><loc>https://agenvoy.com/zh/</loc><changefreq>weekly</changefreq><priority>0.9</priority><lastmod>${today}</lastmod></url>\n`;
if (zhSlugs.length) {
  sitemap += `  <url><loc>https://agenvoy.com/zh/docs/</loc><changefreq>weekly</changefreq><priority>0.8</priority><lastmod>${today}</lastmod></url>\n`;
  for (const slug of zhSlugs) {
    if (slug === "home") continue;
    const pri = (PRIORITIES[slug] || 0.6) - 0.1;
    sitemap += `  <url><loc>https://agenvoy.com/zh/docs/${slug}</loc><changefreq>monthly</changefreq><priority>${pri.toFixed(2)}</priority><lastmod>${today}</lastmod></url>\n`;
  }
}
if (releaseTags.length) {
  sitemap += `  <url><loc>https://agenvoy.com/docs/released/</loc><changefreq>weekly</changefreq><priority>0.6</priority><lastmod>${today}</lastmod></url>\n`;
  for (let i = 0; i < releaseTags.length; i++) {
    const pri = i < 5 ? 0.5 : 0.3;
    sitemap += `  <url><loc>https://agenvoy.com/docs/released/${releaseTags[i]}</loc><changefreq>yearly</changefreq><priority>${pri}</priority><lastmod>${today}</lastmod></url>\n`;
  }
}
sitemap += `</urlset>\n`;
const zhDocUrls = zhSlugs.filter(s => s !== "home").length;
const zhCount = 1 + (zhSlugs.length ? 1 + zhDocUrls : 0);
const sitemapCount = allSlugs.length + 1 + (releaseTags.length ? releaseTags.length + 1 : 0) + zhCount;
fs.writeFileSync(path.join(__dirname, "public/sitemap.xml"), sitemap);
console.log(`OK: sitemap.xml (${sitemapCount} URLs)`);

// Generate robots.txt
const robots = `User-agent: *
Allow: /

Sitemap: https://agenvoy.com/sitemap.xml
`;
fs.writeFileSync(path.join(__dirname, "public/robots.txt"), robots);
console.log("OK: robots.txt");

console.log(`\nBuilt ${built} doc pages (${builtZh} zh), ${releaseTags.length} release pages.`);
