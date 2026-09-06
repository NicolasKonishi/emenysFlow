package templates

import "html/template"

func iconSVG(name string) template.HTML {
	markup, ok := iconMarkup[name]
	if !ok {
		return ""
	}
	return template.HTML(markup)
}

func iconTag(name, path string) string {
	return `<svg class="icon icon-` + name + `" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">` + path + `</svg>`
}

const markSVG = `<img class="icon icon-mark" src="/static/icons/emenys-mark-ink.png" alt="" width="200" height="220">`

var iconMarkup = map[string]string{
	"mark":        markSVG,
	"home":        iconTag("home", `<path d="M4 11.2 12 4l8 7.2"/><path d="M6.5 10.5V20h11V10.5"/>`),
	"events":      iconTag("events", `<rect x="4" y="5" width="16" height="15" rx="2"/><path d="M8 3.5v3M16 3.5v3M4 10h16"/>`),
	"layouts":     iconTag("layouts", `<rect x="4" y="4" width="7" height="7" rx="1.2"/><rect x="13" y="4" width="7" height="7" rx="1.2"/><rect x="4" y="13" width="7" height="7" rx="1.2"/><rect x="13" y="13" width="7" height="7" rx="1.2"/>`),
	"models":      iconTag("models", `<path d="M4 8.5 12 4l8 4.5-8 4.5L4 8.5Z"/><path d="M4 12.5 12 17l8-4.5"/><path d="M4 16.5 12 21l8-4.5"/>`),
	"catalog":     iconTag("catalog", `<path d="M5 5.5h10.5A3.5 3.5 0 0 1 19 9v11H8.5A3.5 3.5 0 0 0 5 16.5V5.5Z"/><path d="M5 16.5h10.5"/>`),
	"inventory":   iconTag("inventory", `<path d="M4 8h16v11H4z"/><path d="M4 8 12 3.5 20 8"/><path d="M10 12h4"/>`),
	"rules":       iconTag("rules", `<path d="M7 4.5h10M7 9.5h10M7 14.5h6"/><path d="M5 4.5h.01M5 9.5h.01M5 14.5h.01"/><path d="M15 17.5 17 19.5 21 14.5"/>`),
	"settings":    iconTag("settings", `<circle cx="12" cy="12" r="3"/><path d="M12 3.5v2.2M12 18.3V20.5M4.8 6.4l1.6 1.6M17.6 16l1.6 1.6M3.5 12h2.2M18.3 12H20.5M4.8 17.6l1.6-1.6M17.6 8l1.6-1.6"/>`),
	"checklists":  iconTag("checklists", `<path d="M8.5 6.5h11M8.5 12h11M8.5 17.5h11"/><path d="M4.5 6.5 5.7 7.7 8 5.4M4.5 12 5.7 13.2 8 10.9"/>`),
	"plus":        iconTag("plus", `<path d="M12 5v14M5 12h14"/>`),
	"minus":       iconTag("minus", `<path d="M5 12h14"/>`),
	"search":      iconTag("search", `<circle cx="11" cy="11" r="6.5"/><path d="m16 16 4 4"/>`),
	"arrow":       iconTag("arrow", `<path d="M5 12h14M13 6l6 6-6 6"/>`),
	"back":        iconTag("back", `<path d="M15 6 9 12l6 6M9 12h11"/>`),
	"logout":      iconTag("logout", `<path d="M10 6H6.5A2.5 2.5 0 0 0 4 8.5v7A2.5 2.5 0 0 0 6.5 18H10"/><path d="M12 12h8M16.5 8.5 20 12l-3.5 3.5"/>`),
	"close":       iconTag("close", `<path d="m6 6 12 12M18 6 6 18"/>`),
	"warning":     iconTag("warning", `<path d="m12 4 9 16H3L12 4Z"/><path d="M12 10v4.5M12 17.2h.01"/>`),
	"check":       iconTag("check", `<path d="m5 12.5 4.5 4.5L19 7.5"/>`),
	"return":      iconTag("return", `<path d="M8 8H5.5A2.5 2.5 0 0 0 3 10.5v0A2.5 2.5 0 0 0 5.5 13H16"/><path d="m13 10 3 3-3 3"/>`),
	"refresh":     iconTag("refresh", `<path d="M20 12a8 8 0 1 1-2.2-5.5"/><path d="M20 4.5V8h-3.5"/>`),
	"download":    iconTag("download", `<path d="M12 4v11M8 11l4 4 4-4"/><path d="M5 19h14"/>`),
	"fullscreen":  iconTag("fullscreen", `<path d="M8 5H5v3M16 5h3v3M8 19H5v-3M16 19h3v-3"/>`),
	"select":      iconTag("select", `<path d="m5 4 5.5 14 2.3-6.2L19 9.5 5 4Z"/>`),
	"table-round": iconTag("table-round", `<circle cx="12" cy="12" r="7.5"/>`),
	"table-rect":  iconTag("table-rect", `<rect x="4.5" y="7" width="15" height="10" rx="1.5"/>`),
	"table-row":   iconTag("table-row", `<rect x="3.5" y="8" width="5" height="8" rx="1"/><rect x="9.5" y="8" width="5" height="8" rx="1"/><rect x="15.5" y="8" width="5" height="8" rx="1"/>`),
	"marker":      iconTag("marker", `<rect x="5" y="5" width="14" height="14" rx="1.5" stroke-dasharray="2.4 2"/>`),
	"edit":        iconTag("edit", `<path d="M4 20h4.2L19.5 8.7a2 2 0 0 0 0-2.8L18.1 4.5a2 2 0 0 0-2.8 0L4 15.8V20Z"/><path d="m13.8 6.2 4 4"/>`),
	"more":        iconTag("more", `<circle cx="6" cy="12" r="1.2" fill="currentColor" stroke="none"/><circle cx="12" cy="12" r="1.2" fill="currentColor" stroke="none"/><circle cx="18" cy="12" r="1.2" fill="currentColor" stroke="none"/>`),
	"zoom-in":     iconTag("zoom-in", `<circle cx="11" cy="11" r="6.2"/><path d="m16 16 4 4M11 8.4v5.2M8.4 11h5.2"/>`),
	"zoom-out":    iconTag("zoom-out", `<circle cx="11" cy="11" r="6.2"/><path d="m16 16 4 4M8.4 11h5.2"/>`),
	"zoom-reset":  iconTag("zoom-reset", `<circle cx="12" cy="12" r="3.2"/><circle cx="12" cy="12" r="7.5"/>`),
	"chevron":     iconTag("chevron", `<path d="m7 9 5 6 5-6"/>`),
	"live":        iconTag("live", `<path d="M12 4 20 8.5v7L12 20 4 15.5v-7L12 4Z"/>`),
	"pin":         iconTag("pin", `<path d="M12 21s6.5-5.4 6.5-10.2A6.5 6.5 0 0 0 5.5 10.8C5.5 15.6 12 21 12 21Z"/><circle cx="12" cy="10.6" r="2.1"/>`),
}
