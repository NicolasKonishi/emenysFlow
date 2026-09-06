(() => {
  const stroke = (name, path) =>
    `<svg class="icon icon-${name}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">${path}</svg>`;

  const icons = {
    check: stroke("check", `<path d="m5 12.5 4.5 4.5L19 7.5"/>`),
    plus: stroke("plus", `<path d="M12 5v14M5 12h14"/>`),
    close: stroke("close", `<path d="m6 6 12 12M18 6 6 18"/>`),
    arrow: stroke("arrow", `<path d="M5 12h14M13 6l6 6-6 6"/>`),
    events: stroke("events", `<rect x="4" y="5" width="16" height="15" rx="2"/><path d="M8 3.5v3M16 3.5v3M4 10h16"/>`),
    layouts: stroke("layouts", `<rect x="4" y="4" width="7" height="7" rx="1.2"/><rect x="13" y="4" width="7" height="7" rx="1.2"/><rect x="4" y="13" width="7" height="7" rx="1.2"/><rect x="13" y="13" width="7" height="7" rx="1.2"/>`),
    checklists: stroke("checklists", `<path d="M8.5 6.5h11M8.5 12h11M8.5 17.5h11"/><path d="M4.5 6.5 5.7 7.7 8 5.4M4.5 12 5.7 13.2 8 10.9"/>`),
    search: stroke("search", `<circle cx="11" cy="11" r="6.5"/><path d="m16 16 4 4"/>`),
  };

  window.emenysIcon = (name) => icons[name] || "";
})();
