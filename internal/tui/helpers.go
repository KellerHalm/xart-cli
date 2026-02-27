package tui

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

func defaultFilter() map[string]any {
	return map[string]any{
		"country":                        nil,
		"category_id":                    nil,
		"genres":                         []string{},
		"is_genres_exclude_mode_enabled": false,
		"profile_list_exclusions":        []int{},
		"types":                          []int{},
		"studio":                         nil,
		"source":                         nil,
		"start_year":                     nil,
		"end_year":                       nil,
		"season":                         nil,
		"episodes_from":                  nil,
		"episodes_to":                    nil,
		"episode_duration_from":          nil,
		"episode_duration_to":            nil,
		"status_id":                      nil,
		"age_ratings":                    []int{},
		"sort":                           0,
	}
}

func mustJSON(value any) []byte {
	data, _ := json.Marshal(value)
	return data
}

func parseCards(payload any) ([]Card, error) {
	root, ok := payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected cards payload type: %T", payload)
	}
	rawItems, ok := root["content"].([]any)
	if !ok {
		return nil, nil
	}

	out := make([]Card, 0, len(rawItems))
	for _, rawItem := range rawItems {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		card := parseCard(item)
		if card.ID == 0 {
			continue
		}
		out = append(out, card)
	}
	return out, nil
}

func parseDetail(payload any) (Detail, error) {
	root, ok := payload.(map[string]any)
	if !ok {
		return Detail{}, fmt.Errorf("unexpected detail payload type: %T", payload)
	}
	release, ok := root["release"].(map[string]any)
	if !ok {
		// fallback for direct release payloads
		release = root
	}
	card := parseCard(release)
	return Detail{Card: card}, nil
}

func parseCard(item map[string]any) Card {
	status := ""
	if rawStatus, ok := item["status"].(map[string]any); ok {
		status = stringFromAny(rawStatus["name"])
	}
	categoryName := ""
	if rawCategory, ok := item["category"].(map[string]any); ok {
		categoryName = stringFromAny(rawCategory["name"])
	}

	titleRu := stringFromAny(item["title_ru"])
	titleOriginal := stringFromAny(item["title_original"])
	title := firstNonEmpty(titleRu, titleOriginal)
	if title == "" {
		title = "Без названия"
	}

	return Card{
		ID:             intFromAny(item["id"]),
		Title:          title,
		OriginalTitle:  titleOriginal,
		Year:           stringFromAny(item["year"]),
		Status:         status,
		Genres:         stringFromAny(item["genres"]),
		Rating:         floatFromAny(item["grade"]),
		IsFavorite:     boolFromAny(item["is_favorite"]),
		ProfileList:    intFromAny(item["profile_list_status"]),
		EpisodesTotal:  intFromAny(item["episodes_total"]),
		EpisodesReady:  intFromAny(item["episodes_released"]),
		AgeRating:      intFromAny(item["age_rating"]),
		CategoryName:   categoryName,
		Country:        stringFromAny(item["country"]),
		Studio:         stringFromAny(item["studio"]),
		Source:         stringFromAny(item["source"]),
		Description:    stringFromAny(item["description"]),
		CommentCount:   intFromAny(item["comment_count"]),
		FavoritesCount: intFromAny(item["favorites_count"]),
	}
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0
		}
		return int(n)
	case string:
		if strings.TrimSpace(v) == "" {
			return 0
		}
		var n int
		_, err := fmt.Sscanf(v, "%d", &n)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}

func floatFromAny(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		n, err := v.Float64()
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}

func boolFromAny(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "true") || v == "1"
	case int:
		return v != 0
	case float64:
		return v != 0
	default:
		return false
	}
}

func stringFromAny(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func trimRunes(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= width {
		return string(runes)
	}
	if width <= 1 {
		return "…"
	}
	return string(runes[:width-1]) + "…"
}

func wrapLine(value string, width int) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return []string{""}
	}
	if width <= 1 {
		return []string{trimRunes(trimmed, 1)}
	}

	words := strings.Fields(trimmed)
	if len(words) == 0 {
		return []string{trimRunes(trimmed, width)}
	}

	lines := make([]string, 0, 4)
	current := ""
	for _, word := range words {
		word = trimRunes(word, width)
		if current == "" {
			current = word
			continue
		}
		if utf8.RuneCountInString(current)+1+utf8.RuneCountInString(word) <= width {
			current += " " + word
			continue
		}
		lines = append(lines, current)
		current = word
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func wrapForWidth(value string, width int) string {
	return strings.Join(wrapLine(value, width), "\n")
}

func wrapText(value string, width int) string {
	if strings.TrimSpace(value) == "" {
		return "Описание отсутствует"
	}
	if width < 10 {
		return value
	}

	words := strings.Fields(value)
	if len(words) == 0 {
		return value
	}

	lines := make([]string, 0, int(math.Ceil(float64(len(words))/8)))
	current := ""
	for _, word := range words {
		if len([]rune(current))+len([]rune(word))+1 > width {
			lines = append(lines, strings.TrimSpace(current))
			current = word
			continue
		}
		if current == "" {
			current = word
		} else {
			current += " " + word
		}
	}
	if current != "" {
		lines = append(lines, strings.TrimSpace(current))
	}
	return strings.Join(lines, "\n")
}

func listName(value int) string {
	switch value {
	case 0:
		return "Не смотрю"
	case 1:
		return "Смотрю"
	case 2:
		return "В планах"
	case 3:
		return "Просмотрено"
	case 4:
		return "Отложено"
	case 5:
		return "Брошено"
	default:
		return "—"
	}
}

func emptyFallback(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
