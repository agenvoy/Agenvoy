const TODO_ICON = {
  completed: "check_circle",
  in_progress: "progress_activity",
  pending: "radio_button_unchecked",
};

function renderTodo(view, list) {
  const dom = view.todo;
  dom.innerHTML = "";
  if (!list || list.length === 0) {
    return;
  }

  for (const item of list) {
    const status = TODO_ICON[item.status] ? item.status : "pending";
    const label = (status === "in_progress" && item.active_form) || item.content || "";
    if (!label) {
      continue;
    }
    dom.appendChild(
      _("div", { "data-status": status }, [_("span.material-symbols-outlined", TODO_ICON[status]), _("p", label)]),
    );
  }
  scrollToBottom(false, view.session);
}
