const TG_SESSION = /^tg-/i;

function channelText(text) {
  if (!TG_SESSION.test(currentSessionId || "")) {
    return text;
  }
  return tgHtmlToMarkdown(text);
}

function tgHtmlToMarkdown(text) {
  let out = String(text || "");

  out = out.replace(
    /<pre>\s*<code[^>]*class="language-([^"]*)"[^>]*>([\s\S]*?)<\/code>\s*<\/pre>/gi,
    (_m, lang, body) => "\n```" + lang + "\n" + body.trim() + "\n```\n",
  );
  out = out.replace(/<pre>([\s\S]*?)<\/pre>/gi, (_m, body) => "\n```\n" + stripTags(body).trim() + "\n```\n");
  out = out.replace(/<code[^>]*>([\s\S]*?)<\/code>/gi, (_m, body) => "`" + body + "`");
  out = out.replace(
    /<a\s[^>]*href="([^"]*)"[^>]*>([\s\S]*?)<\/a>/gi,
    (_m, href, body) => "[" + body + "](" + href + ")",
  );
  out = out.replace(/<(b|strong)>([\s\S]*?)<\/\1>/gi, (_m, _tag, body) => "**" + body + "**");
  out = out.replace(/<(i|em)>([\s\S]*?)<\/\1>/gi, (_m, _tag, body) => "*" + body + "*");
  out = out.replace(/<(s|strike|del)>([\s\S]*?)<\/\1>/gi, (_m, _tag, body) => "~~" + body + "~~");
  out = out.replace(
    /<blockquote[^>]*>([\s\S]*?)<\/blockquote>/gi,
    (_m, body) =>
      "\n" +
      stripTags(body)
        .trim()
        .split("\n")
        .map((line) => "> " + line)
        .join("\n") +
      "\n",
  );
  out = out.replace(/<\/?(u|ins|tg-spoiler|span)[^>]*>/gi, "");

  return decodeEntities(out);
}

function stripTags(text) {
  return String(text || "").replace(/<[^>]*>/g, "");
}

function decodeEntities(text) {
  return String(text || "")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/&#0?39;/g, "'")
    .replace(/&amp;/g, "&");
}
