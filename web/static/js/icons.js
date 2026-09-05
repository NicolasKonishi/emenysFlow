(() => {
  const stroke = (name, path) =>
    `<svg class="icon icon-${name}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">${path}</svg>`;

  const icons = {
    check: stroke("check", `<path d="m5 12.5 4.5 4.5L19 7.5"/>`),
    plus: stroke("plus", `<path d="M12 5v14M5 12h14"/>`),
    close: stroke("close", `<path d="m6 6 12 12M18 6 6 18"/>`),
    arrow: stroke("arrow", `<path d="M5 12h14M13 6l6 6-6 6"/>`),
  };

  window.emenysIcon = (name) => icons[name] || "";
})();
