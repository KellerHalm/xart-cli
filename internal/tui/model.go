package tui

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"xart-cli/internal/player"
	"xart-cli/internal/xart"
)

type viewMode int

const (
	modeGrid viewMode = iota
	modeDetail
)

type sectionMode int

const (
	sectionHome sectionMode = iota
	sectionBookmarks
)

type category struct {
	Key          string
	Title        string
	Apply        func(filter map[string]any)
	BookmarkList int
	IsFavorite   bool
}

type Card struct {
	ID             int
	Title          string
	Year           string
	Status         string
	Genres         string
	Rating         float64
	IsFavorite     bool
	ProfileList    int
	EpisodesTotal  int
	EpisodesReady  int
	OriginalTitle  string
	AgeRating      int
	CategoryName   string
	Country        string
	Studio         string
	Source         string
	Description    string
	CommentCount   int
	FavoritesCount int
}

type Detail struct {
	Card
}

type cardsLoadedMsg struct {
	Section     sectionMode
	CategoryKey string
	Page        int
	Cards       []Card
	Err         error
}

type detailLoadedMsg struct {
	ID     int
	Detail Detail
	Err    error
}

type favoriteToggledMsg struct {
	ID         int
	IsFavorite bool
	Err        error
}

type bookmarkSetMsg struct {
	ID      int
	List    int
	Err     error
	Message string
}

type watchResolvedMsg struct {
	Selection player.Selection
	Err       error
}

type watchFinishedMsg struct {
	Selection player.Selection
	Player    string
	Err       error
}

type Model struct {
	client *xart.Client
	token  string
	userID int

	authCallbacks AuthCallbacks
	auth          *authForm

	section            sectionMode
	homeCategories     []category
	bookmarkCategories []category
	homeCategory       int
	bookmarkCategory   int
	homePage           int
	bookmarkPage       int
	category           int
	page               int
	loading            bool
	errText            string
	statusText         string

	width  int
	height int
	cols   int

	cards     []Card
	selected  int
	rowOffset int
	mode      viewMode
	detail    *Detail
	detailPos int
}

func NewModel(client *xart.Client, token string, userID int, categoryKey string, page int, callbacks AuthCallbacks) *Model {
	homeCategories := []category{
		{Key: "last", Title: "Последнее", Apply: func(_ map[string]any) {}},
		{Key: "ongoing", Title: "Онгоинги", Apply: func(filter map[string]any) { filter["status_id"] = 2 }},
		{Key: "announce", Title: "Анонсы", Apply: func(filter map[string]any) { filter["status_id"] = 3 }},
		{Key: "finished", Title: "Завершенные", Apply: func(filter map[string]any) { filter["status_id"] = 1 }},
		{Key: "films", Title: "Фильмы", Apply: func(filter map[string]any) { filter["category_id"] = 2 }},
	}
	bookmarkCategories := []category{
		{Key: "favorite", Title: "Избранные", IsFavorite: true},
		{Key: "watching", Title: "Смотрю", BookmarkList: 1},
		{Key: "planned", Title: "Запланировано", BookmarkList: 2},
		{Key: "watched", Title: "Просмотрено", BookmarkList: 3},
		{Key: "delayed", Title: "Отложено", BookmarkList: 4},
		{Key: "abandoned", Title: "Брошено", BookmarkList: 5},
	}

	section := sectionHome
	homeCategory := 0
	bookmarkCategory := 0
	switch {
	case categoryIndexByKey(homeCategories, categoryKey) >= 0:
		homeCategory = categoryIndexByKey(homeCategories, categoryKey)
	case categoryIndexByKey(bookmarkCategories, categoryKey) >= 0:
		section = sectionBookmarks
		bookmarkCategory = categoryIndexByKey(bookmarkCategories, categoryKey)
	}

	safePage := max(0, page)
	activeCategory := homeCategory
	homePage := safePage
	bookmarkPage := 0
	if section == sectionBookmarks {
		activeCategory = bookmarkCategory
		homePage = 0
		bookmarkPage = safePage
	}

	return &Model{
		client:             client,
		token:              token,
		userID:             userID,
		authCallbacks:      callbacks,
		section:            section,
		homeCategories:     homeCategories,
		bookmarkCategories: bookmarkCategories,
		homeCategory:       homeCategory,
		bookmarkCategory:   bookmarkCategory,
		homePage:           homePage,
		bookmarkPage:       bookmarkPage,
		category:           activeCategory,
		page:               safePage,
		cols:               3,
		loading:            true,
		rowOffset:          0,
	}
}

func (m *Model) Init() tea.Cmd {
	return m.fetchCardsCmd(m.section, m.currentCategory().Key, m.page)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateColumns()
		m.ensureSelectionVisible()
		return m, nil
	case cardsLoadedMsg:
		if msg.Section != m.section || msg.CategoryKey != m.currentCategory().Key || msg.Page != m.page {
			return m, nil
		}
		m.loading = false
		if msg.Err != nil {
			m.errText = msg.Err.Error()
			m.cards = nil
			m.selected = 0
			m.rowOffset = 0
			return m, nil
		}
		m.errText = ""
		m.cards = msg.Cards
		if len(m.cards) == 0 {
			m.selected = 0
			m.rowOffset = 0
			m.statusText = "Ничего не найдено на этой странице"
			return m, nil
		}
		if m.selected >= len(m.cards) {
			m.selected = len(m.cards) - 1
		}
		m.ensureSelectionVisible()
		m.statusText = fmt.Sprintf("Загружено карточек: %d", len(m.cards))
		return m, nil
	case detailLoadedMsg:
		if msg.Err != nil {
			m.errText = msg.Err.Error()
			return m, nil
		}
		if !m.hasCard(msg.ID) {
			return m, nil
		}
		m.errText = ""
		m.detail = &msg.Detail
		m.detailPos = 0
		return m, nil
	case favoriteToggledMsg:
		if msg.Err != nil {
			m.errText = msg.Err.Error()
			return m, nil
		}
		m.errText = ""
		removedFromCurrentView := false
		for i := range m.cards {
			if m.cards[i].ID == msg.ID {
				m.cards[i].IsFavorite = msg.IsFavorite
				break
			}
		}
		if m.detail != nil && m.detail.ID == msg.ID {
			m.detail.IsFavorite = msg.IsFavorite
		}
		if m.section == sectionBookmarks && m.currentCategory().IsFavorite && !msg.IsFavorite {
			removedFromCurrentView = m.removeCardByID(msg.ID)
		}
		if msg.IsFavorite {
			m.statusText = "Добавлено в избранное"
		} else {
			m.statusText = "Удалено из избранного"
			if removedFromCurrentView {
				m.statusText = "Удалено из избранного и из текущего списка"
			}
		}
		return m, nil
	case bookmarkSetMsg:
		if msg.Err != nil {
			m.errText = msg.Err.Error()
			return m, nil
		}
		m.errText = ""
		for i := range m.cards {
			if m.cards[i].ID == msg.ID {
				m.cards[i].ProfileList = msg.List
				break
			}
		}
		if m.detail != nil && m.detail.ID == msg.ID {
			m.detail.ProfileList = msg.List
		}
		if m.section == sectionBookmarks {
			currentCategory := m.currentCategory()
			if !currentCategory.IsFavorite && currentCategory.BookmarkList != msg.List {
				m.removeCardByID(msg.ID)
			}
		}
		m.statusText = msg.Message
		return m, nil
	case watchResolvedMsg:
		if msg.Err != nil {
			m.errText = msg.Err.Error()
			return m, nil
		}
		launch, err := player.BuildLaunchPlan(msg.Selection.Episode.URL, player.LaunchOptions{})
		if err != nil {
			m.errText = err.Error()
			return m, nil
		}
		m.errText = ""
		m.statusText = fmt.Sprintf(
			"Запуск %s | %s | %s",
			launch.PlayerName,
			emptyFallback(msg.Selection.Source.Name, "источник"),
			emptyFallback(msg.Selection.Episode.Name, fmt.Sprintf("эпизод %d", msg.Selection.Episode.Position)),
		)

		proc := launch.Command()
		proc.Stdin = os.Stdin
		proc.Stdout = os.Stdout
		proc.Stderr = os.Stderr

		selection := msg.Selection
		playerName := launch.PlayerName
		return m, tea.ExecProcess(proc, func(execErr error) tea.Msg {
			return watchFinishedMsg{
				Selection: selection,
				Player:    playerName,
				Err:       execErr,
			}
		})
	case watchFinishedMsg:
		if msg.Err != nil {
			m.errText = "Ошибка плеера: " + msg.Err.Error()
			return m, nil
		}
		m.errText = ""
		m.statusText = fmt.Sprintf(
			"Просмотр завершен: %s (%s)",
			emptyFallback(msg.Selection.Episode.Name, fmt.Sprintf("эпизод %d", msg.Selection.Episode.Position)),
			emptyFallback(msg.Player, "player"),
		)
		return m, nil
	case authLoginMsg:
		if m.auth != nil {
			m.auth.Working = false
		}
		if msg.Err != nil {
			if m.auth != nil {
				m.auth.ErrorText = msg.Err.Error()
			} else {
				m.errText = msg.Err.Error()
			}
			return m, nil
		}
		if err := m.applyLogin(msg.Token, msg.UserID); err != nil {
			m.errText = err.Error()
			return m, nil
		}
		return m, m.fetchCardsCmd(m.section, m.currentCategory().Key, m.page)
	case authSignupCreateMsg:
		if m.auth != nil {
			m.auth.Working = false
		}
		if msg.Err != nil {
			if m.auth != nil {
				m.auth.ErrorText = msg.Err.Error()
			} else {
				m.errText = msg.Err.Error()
			}
			return m, nil
		}
		m.auth = newSignupVerifyForm(msg.Pending)
		m.statusText = "Код подтверждения отправлен. Завершите регистрацию."
		return m, nil
	case authSignupVerifyMsg:
		if m.auth != nil {
			m.auth.Working = false
		}
		if msg.Err != nil {
			if m.auth != nil {
				m.auth.ErrorText = msg.Err.Error()
			} else {
				m.errText = msg.Err.Error()
			}
			return m, nil
		}
		if err := m.applyLogin(msg.Token, msg.UserID); err != nil {
			m.errText = err.Error()
			return m, nil
		}
		m.statusText = "Регистрация завершена, вход выполнен"
		return m, m.fetchCardsCmd(m.section, m.currentCategory().Key, m.page)
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.auth != nil {
		return m.handleAuthKey(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	}

	if m.mode == modeDetail {
		switch msg.String() {
		case "esc", "backspace":
			m.mode = modeGrid
			m.detailPos = 0
			return m, nil
		case "up", "k":
			m.detailPos = max(0, m.detailPos-1)
			return m, nil
		case "down", "j":
			m.detailPos++
			return m, nil
		case "pgup":
			m.detailPos = max(0, m.detailPos-8)
			return m, nil
		case "pgdown", " ":
			m.detailPos += 8
			return m, nil
		case "home":
			m.detailPos = 0
			return m, nil
		case "b":
			return m, m.switchSection(sectionBookmarks)
		case "g":
			return m, m.switchSection(sectionHome)
		case "f":
			return m, m.toggleFavoriteCmd()
		case "w":
			return m, m.watchSelectedCmd()
		case "0", "1", "2", "3", "4", "5":
			list := int(msg.Runes[0] - '0')
			return m, m.setBookmarkCmd(list)
		case "i":
			m.openLoginForm()
			return m, nil
		case "u":
			m.openSignupForm()
			return m, nil
		case "o":
			if m.token == "" {
				m.statusText = "Вы не вошли в аккаунт"
				return m, nil
			}
			if err := m.logout(); err != nil {
				m.errText = err.Error()
				return m, nil
			}
			return m, m.fetchCardsCmd(m.section, m.currentCategory().Key, m.page)
		}
		return m, nil
	}

	switch msg.String() {
	case "b":
		return m, m.switchSection(sectionBookmarks)
	case "g":
		return m, m.switchSection(sectionHome)
	case "enter":
		if !m.hasSelection() {
			return m, nil
		}
		m.mode = modeDetail
		m.detail = nil
		m.detailPos = 0
		return m, m.fetchDetailCmd(m.selectedCard().ID)
	case "tab", "shift+tab", "]", "[":
		categories := m.activeCategories()
		if len(categories) == 0 {
			return m, nil
		}
		delta := 1
		if msg.String() == "shift+tab" || msg.String() == "[" {
			delta = -1
		}
		m.category = (m.category + delta + len(categories)) % len(categories)
		m.page = 0
		m.syncSectionCursor()
		m.beginCardsReload(true)
		m.statusText = fmt.Sprintf("Категория: %s", m.currentCategory().Title)
		return m, m.fetchCardsCmd(m.section, m.currentCategory().Key, m.page)
	case "n":
		m.page++
		m.syncSectionCursor()
		m.beginCardsReload(true)
		return m, m.fetchCardsCmd(m.section, m.currentCategory().Key, m.page)
	case "p":
		if m.page > 0 {
			m.page--
			m.syncSectionCursor()
			m.beginCardsReload(true)
			return m, m.fetchCardsCmd(m.section, m.currentCategory().Key, m.page)
		}
		return m, nil
	case "r":
		m.beginCardsReload(false)
		return m, m.fetchCardsCmd(m.section, m.currentCategory().Key, m.page)
	case "left", "h":
		if m.selected > 0 {
			m.selected--
			m.ensureSelectionVisible()
		}
		return m, nil
	case "right", "l":
		if m.selected < len(m.cards)-1 {
			m.selected++
			m.ensureSelectionVisible()
		}
		return m, nil
	case "up", "k":
		if m.selected-m.cols >= 0 {
			m.selected -= m.cols
			m.ensureSelectionVisible()
		}
		return m, nil
	case "down", "j":
		if m.selected+m.cols < len(m.cards) {
			m.selected += m.cols
			m.ensureSelectionVisible()
		}
		return m, nil
	case "f":
		return m, m.toggleFavoriteCmd()
	case "w":
		return m, m.watchSelectedCmd()
	case "0", "1", "2", "3", "4", "5":
		list := int(msg.Runes[0] - '0')
		return m, m.setBookmarkCmd(list)
	case "i":
		m.openLoginForm()
		return m, nil
	case "u":
		m.openSignupForm()
		return m, nil
	case "o":
		if m.token == "" {
			m.statusText = "Вы не вошли в аккаунт"
			return m, nil
		}
		if err := m.logout(); err != nil {
			m.errText = err.Error()
			return m, nil
		}
		return m, m.fetchCardsCmd(m.section, m.currentCategory().Key, m.page)
	}

	return m, nil
}

func (m *Model) View() string {
	if m.width == 0 {
		return "Инициализация интерфейса..."
	}

	header := m.renderHeader()
	authForm := m.renderAuthForm()
	footer := m.renderFooter()
	headerLines := countLines(header)
	authLines := countLines(authForm)
	footerLines := countLines(footer)
	bodyLines := m.bodyLines(headerLines, authLines, footerLines)
	if m.mode == modeDetail {
		body := m.renderDetail(bodyLines)
		if authForm != "" {
			return lipgloss.JoinVertical(lipgloss.Left, header, body, authForm, footer)
		}
		return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	}

	body := m.renderGrid(bodyLines)
	if authForm != "" {
		return lipgloss.JoinVertical(lipgloss.Left, header, body, authForm, footer)
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m *Model) renderHeader() string {
	authState := "guest"
	if m.token != "" {
		authState = "auth"
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("77"))
	textWidth := m.contentWidth()

	header := titleStyle.Render(trimRunes("Xart Terminal UI", textWidth))
	meta := metaStyle.Render(wrapForWidth(fmt.Sprintf("Раздел: %s  |  Категория: %s  |  Страница: %d  |  Режим: %s", m.sectionName(), m.currentCategory().Title, m.page, authState), textWidth))

	lines := []string{header, meta}
	if m.loading {
		lines = append(lines, statusStyle.Render(wrapForWidth("Загрузка карточек...", textWidth)))
	}
	if m.errText != "" {
		lines = append(lines, errStyle.Render(wrapForWidth("Ошибка: "+m.errText, textWidth)))
	} else if m.statusText != "" {
		lines = append(lines, statusStyle.Render(wrapForWidth(m.statusText, textWidth)))
	}
	return lipgloss.NewStyle().
		Padding(0, 1).
		MaxWidth(max(1, m.width)).
		Render(strings.Join(lines, "\n"))
}

func (m *Model) renderGrid(bodyLines int) string {
	if m.loading {
		return lipgloss.NewStyle().Padding(1, 1).Render("Подождите, получаем список...")
	}
	if len(m.cards) == 0 {
		return lipgloss.NewStyle().Padding(1, 1).Render("Карточек нет")
	}

	cardHeight := min(m.cardContentLines(), max(3, bodyLines-2))
	cardWidth := m.cardWidth()
	normal := lipgloss.NewStyle().
		Width(cardWidth).
		Height(cardHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)
	selected := normal.Copy().BorderForeground(lipgloss.Color("205")).Bold(true)

	totalRows := m.totalRows()
	visibleRows := m.visibleRows(bodyLines)
	m.ensureSelectionVisibleFor(visibleRows)

	startRow := m.rowOffset
	maxStart := max(0, totalRows-visibleRows)
	if startRow > maxStart {
		startRow = maxStart
		m.rowOffset = startRow
	}
	endRow := min(totalRows, startRow+visibleRows)

	rows := make([]string, 0, endRow-startRow+2)
	if startRow > 0 {
		rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("↑ выше есть карточки"))
	}

	for row := startRow; row < endRow; row++ {
		i := row * m.cols
		columns := make([]string, 0, m.cols)
		for j := 0; j < m.cols; j++ {
			idx := i + j
			if idx >= len(m.cards) {
				break
			}
			card := m.cards[idx]
			content := m.renderCardContent(card, cardWidth-2, cardHeight)
			style := normal
			if idx == m.selected {
				style = selected
			}
			columns = append(columns, style.Render(content))
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, columns...))
	}

	if endRow < totalRows {
		rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("↓ ниже есть карточки"))
	}

	return lipgloss.NewStyle().
		Padding(0, 1).
		MaxWidth(max(1, m.width)).
		Render(strings.Join(rows, "\n"))
}

func (m *Model) renderCardContent(card Card, width int, maxLines int) string {
	heart := "♡"
	if card.IsFavorite {
		heart = "♥"
	}
	lines := []string{
		trimRunes(card.Title, width),
		trimRunes(fmt.Sprintf("ID %d  %s", card.ID, heart), width),
		trimRunes(fmt.Sprintf("★ %.2f  %s", card.Rating, emptyFallback(card.Year, "?")), width),
		trimRunes(emptyFallback(card.Status, "—"), width),
		trimRunes(fmt.Sprintf("Список: %s", listName(card.ProfileList)), width),
		trimRunes(fmt.Sprintf("Эп: %d/%d", card.EpisodesReady, card.EpisodesTotal), width),
	}
	if maxLines < len(lines) {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderDetail(bodyLines int) string {
	if m.loading {
		return lipgloss.NewStyle().Padding(1, 1).Render("Загрузка...")
	}
	if m.detail == nil {
		return lipgloss.NewStyle().Padding(1, 1).Render("Загружаю детали тайтла...")
	}

	panelWidth := max(16, m.width-2)
	innerWidth := max(10, panelWidth-4)

	heart := "нет"
	if m.detail.IsFavorite {
		heart = "да"
	}

	lines := make([]string, 0, 24)
	addWrappedLine(&lines, fmt.Sprintf("%s (%s)", m.detail.Title, emptyFallback(m.detail.Year, "?")), innerWidth)
	addWrappedLine(&lines, fmt.Sprintf("ID: %d | Рейтинг: %.2f | Статус: %s", m.detail.ID, m.detail.Rating, emptyFallback(m.detail.Status, "—")), innerWidth)
	addWrappedLine(&lines, fmt.Sprintf("Избранное: %s | Список: %s", heart, listName(m.detail.ProfileList)), innerWidth)
	addWrappedLine(&lines, fmt.Sprintf("Эпизоды: %d/%d | Студия: %s", m.detail.EpisodesReady, m.detail.EpisodesTotal, emptyFallback(m.detail.Studio, "—")), innerWidth)
	addWrappedLine(&lines, fmt.Sprintf("Жанры: %s", emptyFallback(m.detail.Genres, "—")), innerWidth)
	lines = append(lines, "")
	description := strings.TrimSpace(m.detail.Description)
	if description == "" {
		description = "Описание отсутствует"
	}
	for _, part := range strings.Split(description, "\n") {
		if strings.TrimSpace(part) == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, wrapLine(part, innerWidth)...)
	}

	// Keep detail readable on small terminals and allow scrolling.
	showHints := bodyLines >= 8
	viewPort := max(1, bodyLines-2)
	if showHints {
		viewPort = max(1, bodyLines-4)
	}
	maxStart := max(0, len(lines)-viewPort)
	if m.detailPos > maxStart {
		m.detailPos = maxStart
	}
	if m.detailPos < 0 {
		m.detailPos = 0
	}
	start := m.detailPos
	end := min(len(lines), start+viewPort)

	visible := make([]string, 0, (end-start)+2)
	if showHints && start > 0 {
		visible = append(visible, trimRunes("↑ k/up: выше", innerWidth))
	}
	visible = append(visible, lines[start:end]...)
	if showHints && end < len(lines) {
		visible = append(visible, trimRunes("↓ j/down: ниже", innerWidth))
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(0, 1).
		Width(panelWidth)

	return lipgloss.NewStyle().
		Padding(0, 1).
		MaxWidth(max(1, m.width)).
		Render(box.Render(strings.Join(visible, "\n")))
}

func (m *Model) renderFooter() string {
	help := "Grid: ←→↑↓/h j k l, Enter=детали, w=смотреть, f=избранное, 0..5=список, b=закладки, g=главная, i=вход, u=регистрация, o=выход, Tab/[ ]=категория, n/p=страница, r=reload, q=выход"
	if m.auth != nil {
		help = "Auth: ввод текста, Tab/Shift+Tab = поле, Enter = отправить, Esc = закрыть"
	}
	if m.mode == modeDetail {
		help = "Detail: esc=назад, j/k или ↑/↓=скролл, w=смотреть, f=избранное, 0..5=список, b=закладки, g=главная, i=вход, u=регистрация, o=выход, q=выход"
		if m.auth != nil {
			help = "Auth: ввод текста, Tab/Shift+Tab = поле, Enter = отправить, Esc = закрыть"
		}
	}
	if m.width < 100 && m.auth == nil {
		if m.mode == modeDetail {
			help = "Detail: esc назад | j/k скролл | w | f | 0..5 | b/g | i/u/o | q"
		} else {
			help = "Grid: move ←→↑↓/hjkl | Enter | w | Tab | n/p | f | 0..5 | b/g | i/u/o | q"
		}
	}
	if m.width < 72 && m.auth == nil {
		if m.mode == modeDetail {
			help = "Detail: esc | j/k | w | f | 0..5 | b/g | q"
		} else {
			help = "Grid: arrows/hjkl | Enter | w | Tab | n/p | b/g | q"
		}
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 1).
		MaxWidth(max(1, m.width)).
		Render(wrapForWidth(help, m.contentWidth()))
}

func (m *Model) fetchCardsCmd(section sectionMode, categoryKey string, page int) tea.Cmd {
	return func() tea.Msg {
		cat, found := categoryByKey(m.categoriesForSection(section), categoryKey)
		if !found {
			return cardsLoadedMsg{
				Section:     section,
				CategoryKey: categoryKey,
				Page:        page,
				Err:         fmt.Errorf("unknown category: %s", categoryKey),
			}
		}

		var (
			payload any
			err     error
		)
		if section == sectionHome {
			filter := defaultFilter()
			if cat.Apply != nil {
				cat.Apply(filter)
			}
			payload, err = m.client.Do(context.Background(), xart.Request{
				Method: "POST",
				Path:   fmt.Sprintf("/filter/%d", page),
				Body:   mustJSON(filter),
				Token:  m.token,
				Headers: map[string]string{
					"Content-Type": "application/json; charset=UTF-8",
				},
			})
		} else {
			if strings.TrimSpace(m.token) == "" {
				err = fmt.Errorf("для раздела «Закладки» нужно войти в аккаунт")
			} else if m.userID == 0 && !cat.IsFavorite {
				err = fmt.Errorf("не удалось определить профиль, войдите через UI (клавиша i)")
			} else {
				query := url.Values{"sort": []string{"1"}}
				path := fmt.Sprintf("/profile/list/all/%d/%d/%d", m.userID, cat.BookmarkList, page)
				if cat.IsFavorite {
					path = fmt.Sprintf("/favorite/all/%d", page)
				}
				payload, err = m.client.Do(context.Background(), xart.Request{
					Method: "GET",
					Path:   path,
					Query:  query,
					Token:  m.token,
				})
			}
		}
		if err != nil {
			return cardsLoadedMsg{
				Section:     section,
				CategoryKey: categoryKey,
				Page:        page,
				Err:         err,
			}
		}

		cards, parseErr := parseCards(payload)
		return cardsLoadedMsg{
			Section:     section,
			CategoryKey: categoryKey,
			Page:        page,
			Cards:       cards,
			Err:         parseErr,
		}
	}
}

func (m *Model) fetchDetailCmd(id int) tea.Cmd {
	return func() tea.Msg {
		query := url.Values{}
		payload, err := m.client.Do(context.Background(), xart.Request{
			Method: "GET",
			Path:   fmt.Sprintf("/release/%d", id),
			Query:  query,
			Token:  m.token,
		})
		if err != nil {
			return detailLoadedMsg{ID: id, Err: err}
		}

		detail, parseErr := parseDetail(payload)
		return detailLoadedMsg{
			ID:     id,
			Detail: detail,
			Err:    parseErr,
		}
	}
}

func (m *Model) toggleFavoriteCmd() tea.Cmd {
	if !m.hasSelection() {
		return nil
	}
	if m.token == "" {
		m.statusText = "Для избранного нужна авторизация: xart auth login"
		return nil
	}
	card := m.selectedCard()
	return func() tea.Msg {
		path := fmt.Sprintf("/favorite/add/%d", card.ID)
		nextValue := true
		if card.IsFavorite {
			path = fmt.Sprintf("/favorite/delete/%d", card.ID)
			nextValue = false
		}
		_, err := m.client.Do(context.Background(), xart.Request{
			Method: "GET",
			Path:   path,
			Token:  m.token,
		})
		return favoriteToggledMsg{
			ID:         card.ID,
			IsFavorite: nextValue,
			Err:        err,
		}
	}
}

func (m *Model) setBookmarkCmd(list int) tea.Cmd {
	if !m.hasSelection() {
		return nil
	}
	if m.token == "" {
		m.statusText = "Для списков нужна авторизация: xart auth login"
		return nil
	}
	card := m.selectedCard()
	return func() tea.Msg {
		_, err := m.client.Do(context.Background(), xart.Request{
			Method: "GET",
			Path:   fmt.Sprintf("/profile/list/add/%d/%d", list, card.ID),
			Token:  m.token,
		})
		return bookmarkSetMsg{
			ID:      card.ID,
			List:    list,
			Err:     err,
			Message: fmt.Sprintf("Установлен список: %s", listName(list)),
		}
	}
}

func (m *Model) watchSelectedCmd() tea.Cmd {
	releaseID := 0
	if m.mode == modeDetail && m.detail != nil && m.detail.ID > 0 {
		releaseID = m.detail.ID
	} else if m.hasSelection() {
		releaseID = m.selectedCard().ID
	}
	if releaseID <= 0 {
		m.statusText = "Сначала выберите тайтл для просмотра"
		return nil
	}

	m.errText = ""
	m.statusText = "Ищу доступный эпизод и запускаю плеер..."
	return func() tea.Msg {
		selection, err := player.ResolveSelection(
			context.Background(),
			m.client,
			releaseID,
			m.token,
			0,
			0,
			-1,
		)
		if err != nil {
			return watchResolvedMsg{Err: err}
		}
		return watchResolvedMsg{Selection: selection}
	}
}

func (m *Model) currentCategory() category {
	categories := m.activeCategories()
	if len(categories) == 0 {
		return category{}
	}
	if m.category < 0 || m.category >= len(categories) {
		m.category = 0
		m.syncSectionCursor()
	}
	return categories[m.category]
}

func (m *Model) activeCategories() []category {
	return m.categoriesForSection(m.section)
}

func (m *Model) categoriesForSection(section sectionMode) []category {
	if section == sectionBookmarks {
		return m.bookmarkCategories
	}
	return m.homeCategories
}

func (m *Model) sectionName() string {
	if m.section == sectionBookmarks {
		return "Закладки"
	}
	return "Главная"
}

func (m *Model) switchSection(section sectionMode) tea.Cmd {
	if section == m.section {
		return nil
	}
	if section == sectionBookmarks {
		if strings.TrimSpace(m.token) == "" {
			m.statusText = "Для закладок нужна авторизация"
			return nil
		}
	}

	m.syncSectionCursor()
	m.section = section
	if section == sectionHome {
		m.category = clamp(m.homeCategory, 0, len(m.homeCategories)-1)
		m.page = max(0, m.homePage)
	} else {
		m.category = clamp(m.bookmarkCategory, 0, len(m.bookmarkCategories)-1)
		m.page = max(0, m.bookmarkPage)
		if m.userID == 0 && !m.currentCategory().IsFavorite {
			favoriteIndex := categoryIndexByKey(m.bookmarkCategories, "favorite")
			if favoriteIndex < 0 {
				favoriteIndex = 0
			}
			m.category = favoriteIndex
			m.page = 0
		}
	}
	m.mode = modeGrid
	m.detail = nil
	m.detailPos = 0
	m.beginCardsReload(true)
	if section == sectionBookmarks && m.userID == 0 {
		m.statusText = "Раздел: Закладки (без ID профиля доступно только «Избранные»)"
	} else {
		m.statusText = fmt.Sprintf("Раздел: %s", m.sectionName())
	}
	return m.fetchCardsCmd(m.section, m.currentCategory().Key, m.page)
}

func (m *Model) beginCardsReload(resetSelection bool) {
	if resetSelection {
		m.selected = 0
		m.rowOffset = 0
	}
	m.cards = nil
	m.loading = true
	m.errText = ""
}

func (m *Model) syncSectionCursor() {
	if m.section == sectionBookmarks {
		m.bookmarkCategory = m.category
		m.bookmarkPage = m.page
		return
	}
	m.homeCategory = m.category
	m.homePage = m.page
}

func (m *Model) updateColumns() {
	const minCardWithGap = 21
	if m.width <= minCardWithGap {
		m.cols = 1
		return
	}
	m.cols = max(1, m.width/minCardWithGap)
	m.ensureSelectionVisible()
}

func (m *Model) cardWidth() int {
	if m.cols <= 0 {
		return 30
	}
	width := (m.width / m.cols) - 2
	if width < 8 {
		return 8
	}
	if width > 42 {
		return 42
	}
	return width
}

func (m *Model) cardContentLines() int {
	if m.height < 24 || m.width < 70 {
		return 5
	}
	if m.height < 32 || m.width < 95 {
		return 6
	}
	return 7
}

func (m *Model) contentWidth() int {
	return max(10, m.width-2)
}

func (m *Model) bodyLines(headerLines, authLines, footerLines int) int {
	available := m.height - headerLines - authLines - footerLines
	if available < 3 {
		return 3
	}
	return available
}

func (m *Model) hasSelection() bool {
	return len(m.cards) > 0 && m.selected >= 0 && m.selected < len(m.cards)
}

func (m *Model) selectedCard() Card {
	if !m.hasSelection() {
		return Card{}
	}
	return m.cards[m.selected]
}

func (m *Model) hasCard(id int) bool {
	for i := range m.cards {
		if m.cards[i].ID == id {
			return true
		}
	}
	return false
}

func (m *Model) removeCardByID(id int) bool {
	index := -1
	for i := range m.cards {
		if m.cards[i].ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return false
	}

	m.cards = append(m.cards[:index], m.cards[index+1:]...)
	if len(m.cards) == 0 {
		m.selected = 0
		m.rowOffset = 0
	} else {
		if m.selected >= len(m.cards) {
			m.selected = len(m.cards) - 1
		}
		m.ensureSelectionVisible()
	}

	if m.detail != nil && m.detail.ID == id {
		m.detail = nil
		m.detailPos = 0
		if m.mode == modeDetail {
			m.mode = modeGrid
		}
	}
	return true
}

func (m *Model) totalRows() int {
	if m.cols <= 0 {
		return 0
	}
	return (len(m.cards) + m.cols - 1) / m.cols
}

func (m *Model) ensureSelectionVisible() {
	m.ensureSelectionVisibleFor(m.visibleRows(max(1, m.height-6)))
}

func (m *Model) ensureSelectionVisibleFor(visibleRows int) {
	if !m.hasSelection() || m.cols <= 0 {
		m.rowOffset = 0
		return
	}
	if visibleRows <= 0 {
		visibleRows = 1
	}

	selectedRow := m.selected / m.cols
	if selectedRow < m.rowOffset {
		m.rowOffset = selectedRow
	}
	if selectedRow >= m.rowOffset+visibleRows {
		m.rowOffset = selectedRow - visibleRows + 1
	}

	maxStart := max(0, m.totalRows()-visibleRows)
	if m.rowOffset > maxStart {
		m.rowOffset = maxStart
	}
	if m.rowOffset < 0 {
		m.rowOffset = 0
	}
}

func (m *Model) visibleRows(bodyLines int) int {
	rowHeight := m.cardContentLines() + 2
	if bodyLines < rowHeight {
		return 1
	}
	return max(1, bodyLines/rowHeight)
}

func addWrappedLine(lines *[]string, value string, width int) {
	*lines = append(*lines, wrapLine(value, width)...)
}

func categoryIndexByKey(categories []category, key string) int {
	for i := range categories {
		if categories[i].Key == key {
			return i
		}
	}
	return -1
}

func categoryByKey(categories []category, key string) (category, bool) {
	index := categoryIndexByKey(categories, key)
	if index < 0 {
		return category{}, false
	}
	return categories[index], true
}

func clamp(value, low, high int) int {
	if high < low {
		return low
	}
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func countLines(value string) int {
	if strings.TrimSpace(value) == "" {
		return 0
	}
	return strings.Count(value, "\n") + 1
}
