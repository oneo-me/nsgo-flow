package flowfile

import (
	"slices"
	"strings"
)

// CommonIcons are icons most interfaces use, which the model is shown;
// an icon takes the name of any Lucide icon.
var CommonIcons = []string{"house", "search", "bell", "settings", "user", "users", "circle-user", "mail", "message-circle", "message-square",
	"phone", "send", "inbox", "heart", "star", "bookmark", "share-2", "link", "paperclip", "image", "camera", "video", "mic", "music",
	"play", "pause", "calendar", "clock", "map-pin", "navigation", "compass", "globe", "shopping-cart", "shopping-bag", "store",
	"credit-card", "wallet", "gift", "tag", "package", "truck", "file", "file-text", "folder", "download", "upload", "cloud", "plus",
	"minus", "x", "check", "pencil", "trash", "copy", "funnel", "sliders-horizontal", "list", "layout-grid", "layout-dashboard",
	"menu", "ellipsis", "ellipsis-vertical", "chevron-right", "chevron-left", "chevron-down", "chevron-up", "arrow-right",
	"arrow-left", "arrow-up", "arrow-down", "external-link", "refresh-cw", "rotate-ccw", "log-in", "log-out", "lock", "key",
	"shield", "eye", "eye-off", "info", "circle-alert", "triangle-alert", "circle-check", "circle-x", "circle-question-mark",
	"chart-bar", "chart-line", "chart-pie", "activity", "trending-up", "trending-down", "zap", "sparkles", "flame", "award",
	"trophy", "target", "sun", "moon", "wifi", "battery", "smartphone", "monitor", "qr-code", "scan", "book", "briefcase",
	"building", "coffee", "utensils", "car", "plane", "bike", "dumbbell", "headphones", "graduation-cap"}

// iconAliases are names models still use for icons Lucide renamed, and a
// few common words, by the icon's name.
var iconAliases = map[string]string{
	"home": "house", "cart": "shopping-cart", "profile": "user", "account": "user", "edit": "square-pen", "edit-2": "pencil",
	"edit-3": "pen-line", "trash-2": "trash", "delete": "trash", "more-horizontal": "ellipsis", "more-vertical": "ellipsis-vertical",
	"alert-circle": "circle-alert", "alert-triangle": "triangle-alert", "check-circle": "circle-check", "check-circle-2": "circle-check",
	"x-circle": "circle-x", "help-circle": "circle-question-mark", "help": "circle-question-mark", "plus-circle": "circle-plus",
	"minus-circle": "circle-minus", "user-circle": "circle-user", "play-circle": "circle-play", "bar-chart": "chart-column",
	"bar-chart-2": "chart-column", "bar-chart-3": "chart-column", "bar-chart-4": "chart-column", "line-chart": "chart-line",
	"pie-chart": "chart-pie", "filter": "funnel", "grid": "layout-grid", "check-square": "square-check", "settings-2": "settings-2",
	"loader-2": "loader-circle", "unlock": "lock-open", "logout": "log-out", "login": "log-in", "close": "x", "cancel": "x",
	"add": "plus", "remove": "minus", "notification": "bell", "notifications": "bell", "chat": "message-circle", "message": "message-circle",
	"location": "map-pin", "pin": "map-pin", "photo": "image", "picture": "image", "favorite": "heart", "like": "heart",
	"share": "share-2", "money": "wallet", "payment": "credit-card", "time": "clock", "date": "calendar", "email": "mail",
	"sort-asc": "arrow-up-narrow-wide", "sort-desc": "arrow-down-wide-narrow", "dashboard": "layout-dashboard",
}

// IconName returns the Lucide name of the icon name names, its own or
// an alias's, false when it names none.
func IconName(name string) (string, bool) {
	if alias, ok := iconAliases[name]; ok {
		name = alias
	}
	_, found := slices.BinarySearch(lucideNames, name)
	return name, found
}

// similarIcons are up to n icon names that share words with name, the
// common ones first.
func similarIcons(name string, n int) []string {
	words := strings.FieldsFunc(name, func(r rune) bool { return r == '-' || r == '_' || r == ' ' })
	var out []string
	add := func(list []string) {
		for _, icon := range list {
			if len(out) == n {
				return
			}
			for _, w := range words {
				if len(w) > 2 && strings.Contains(icon, w) && !slices.Contains(out, icon) {
					out = append(out, icon)
					break
				}
			}
		}
	}
	add(CommonIcons)
	add(lucideNames)
	return out
}
