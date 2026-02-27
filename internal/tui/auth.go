package tui

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"xart-cli/internal/xart"
)

type AuthCallbacks struct {
	SaveLogin  func(token string, userID int) error
	SaveLogout func() error
}

type authMode int

const (
	authModeLogin authMode = iota
	authModeSignupCreate
	authModeSignupVerify
)

type authField struct {
	Label  string
	Value  string
	Secret bool
}

type signupPending struct {
	Email    string
	Login    string
	Password string
	Hash     string
}

type authForm struct {
	Mode      authMode
	Title     string
	Subtitle  string
	Fields    []authField
	Focused   int
	Working   bool
	ErrorText string
	InfoText  string
	Pending   signupPending
}

type authLoginMsg struct {
	Token  string
	UserID int
	Err    error
}

type authSignupCreateMsg struct {
	Pending signupPending
	Err     error
}

type authSignupVerifyMsg struct {
	Token  string
	UserID int
	Err    error
}

func newLoginForm() *authForm {
	return &authForm{
		Mode:     authModeLogin,
		Title:    "Вход в аккаунт",
		Subtitle: "Введите логин/email и пароль",
		Fields: []authField{
			{Label: "Логин или email"},
			{Label: "Пароль", Secret: true},
		},
	}
}

func newSignupCreateForm() *authForm {
	return &authForm{
		Mode:     authModeSignupCreate,
		Title:    "Регистрация",
		Subtitle: "Введите email, логин и пароль",
		Fields: []authField{
			{Label: "Email"},
			{Label: "Логин"},
			{Label: "Пароль", Secret: true},
		},
	}
}

func newSignupVerifyForm(pending signupPending) *authForm {
	return &authForm{
		Mode:     authModeSignupVerify,
		Title:    "Подтверждение регистрации",
		Subtitle: "Введите код из письма",
		Fields: []authField{
			{Label: "Код"},
		},
		Pending: pending,
		InfoText: fmt.Sprintf(
			"Логин: %s  |  Email: %s",
			pending.Login,
			pending.Email,
		),
	}
}

func (m *Model) openLoginForm() {
	m.auth = newLoginForm()
	m.errText = ""
}

func (m *Model) openSignupForm() {
	m.auth = newSignupCreateForm()
	m.errText = ""
}

func (m *Model) handleAuthKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.auth == nil {
		return m, nil
	}
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if m.auth.Working {
		if msg.String() == "esc" {
			return m, nil
		}
		return m, nil
	}

	switch msg.String() {
	case "esc", "backspace":
		// On verify form, backspace edits field; ESC closes form.
		if msg.String() == "esc" {
			m.auth = nil
			return m, nil
		}
		m.auth.Fields[m.auth.Focused].Value = trimLastRune(m.auth.Fields[m.auth.Focused].Value)
		return m, nil
	case "tab":
		m.auth.Focused = (m.auth.Focused + 1) % len(m.auth.Fields)
		return m, nil
	case "shift+tab":
		m.auth.Focused = (m.auth.Focused - 1 + len(m.auth.Fields)) % len(m.auth.Fields)
		return m, nil
	case "enter":
		return m, m.submitAuthCmd()
	}

	if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
		m.auth.Fields[m.auth.Focused].Value += string(msg.Runes)
	}
	return m, nil
}

func (m *Model) submitAuthCmd() tea.Cmd {
	if m.auth == nil || m.auth.Working {
		return nil
	}
	m.auth.ErrorText = ""
	m.auth.InfoText = ""

	switch m.auth.Mode {
	case authModeLogin:
		login := strings.TrimSpace(m.auth.Fields[0].Value)
		password := m.auth.Fields[1].Value
		if login == "" || password == "" {
			m.auth.ErrorText = "Заполните логин и пароль"
			return nil
		}
		m.auth.Working = true
		return func() tea.Msg {
			query := url.Values{
				"login":    []string{login},
				"password": []string{password},
			}
			payload, err := m.client.Do(context.Background(), xart.Request{
				Method: "POST",
				Path:   "/auth/signIn",
				Query:  query,
			})
			if err != nil {
				return authLoginMsg{Err: err}
			}
			token, userID, err := parseAuthPayload(payload)
			if err != nil {
				return authLoginMsg{Err: err}
			}
			return authLoginMsg{Token: token, UserID: userID}
		}

	case authModeSignupCreate:
		email := strings.TrimSpace(m.auth.Fields[0].Value)
		login := strings.TrimSpace(m.auth.Fields[1].Value)
		password := m.auth.Fields[2].Value
		if email == "" || login == "" || password == "" {
			m.auth.ErrorText = "Заполните email, логин и пароль"
			return nil
		}
		m.auth.Working = true
		return func() tea.Msg {
			query := url.Values{
				"email":    []string{email},
				"login":    []string{login},
				"password": []string{password},
			}
			payload, err := m.client.Do(context.Background(), xart.Request{
				Method: "POST",
				Path:   "/auth/signUp",
				Query:  query,
			})
			if err != nil {
				return authSignupCreateMsg{Err: err}
			}
			hash, err := parseHashPayload(payload)
			if err != nil {
				return authSignupCreateMsg{Err: err}
			}
			return authSignupCreateMsg{
				Pending: signupPending{
					Email:    email,
					Login:    login,
					Password: password,
					Hash:     hash,
				},
			}
		}

	case authModeSignupVerify:
		code := strings.TrimSpace(m.auth.Fields[0].Value)
		if code == "" {
			m.auth.ErrorText = "Введите код подтверждения"
			return nil
		}
		pending := m.auth.Pending
		m.auth.Working = true
		return func() tea.Msg {
			query := url.Values{
				"email":    []string{pending.Email},
				"login":    []string{pending.Login},
				"password": []string{pending.Password},
				"hash":     []string{pending.Hash},
				"code":     []string{code},
			}
			payload, err := m.client.Do(context.Background(), xart.Request{
				Method: "POST",
				Path:   "/auth/verify",
				Query:  query,
			})
			if err != nil {
				return authSignupVerifyMsg{Err: err}
			}
			token, userID, err := parseAuthPayload(payload)
			if err != nil {
				return authSignupVerifyMsg{Err: err}
			}
			return authSignupVerifyMsg{Token: token, UserID: userID}
		}
	}

	return nil
}

func (m *Model) applyLogin(token string, userID int) error {
	resolvedUserID := userID
	if resolvedUserID == 0 {
		resolvedUserID = m.userID
	}
	if m.authCallbacks.SaveLogin != nil {
		if err := m.authCallbacks.SaveLogin(token, resolvedUserID); err != nil {
			return err
		}
	}
	m.token = token
	m.userID = resolvedUserID
	m.statusText = "Вход выполнен"
	m.auth = nil
	m.loading = true
	return nil
}

func (m *Model) logout() error {
	if m.authCallbacks.SaveLogout != nil {
		if err := m.authCallbacks.SaveLogout(); err != nil {
			return err
		}
	}
	m.syncSectionCursor()
	m.token = ""
	m.userID = 0
	if m.section == sectionBookmarks {
		m.section = sectionHome
		m.category = clamp(m.homeCategory, 0, len(m.homeCategories)-1)
		m.page = max(0, m.homePage)
	}
	m.statusText = "Вы вышли из аккаунта"
	m.auth = nil
	m.loading = true
	return nil
}

func (m *Model) renderAuthForm() string {
	if m.auth == nil {
		return ""
	}

	boxWidth := m.width - 2
	if boxWidth > 90 {
		boxWidth = 90
	}
	if boxWidth < 18 {
		boxWidth = max(10, m.width)
	}
	contentWidth := max(10, boxWidth-4)
	box := lipgloss.NewStyle().
		Width(boxWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(0, 1)
	title := lipgloss.NewStyle().Bold(true).Render(trimRunes(m.auth.Title, contentWidth))
	subtitle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(wrapForWidth(m.auth.Subtitle, contentWidth))

	lines := []string{title, subtitle, ""}
	if m.auth.InfoText != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("77")).Render(wrapForWidth(m.auth.InfoText, contentWidth)), "")
	}
	for i := range m.auth.Fields {
		field := m.auth.Fields[i]
		prefix := "  "
		if i == m.auth.Focused {
			prefix = "> "
		}
		value := field.Value
		if field.Secret {
			value = strings.Repeat("*", utf8.RuneCountInString(field.Value))
		}
		if value == "" {
			value = "..."
		}
		lines = append(lines, wrapLine(fmt.Sprintf("%s%s: %s", prefix, field.Label, value), contentWidth)...)
	}
	lines = append(lines, "")
	if m.auth.Working {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("77")).Render("Отправка..."))
	} else {
		lines = append(lines, wrapLine("Enter: отправить | Tab: следующее поле | Esc: закрыть", contentWidth)...)
	}
	if m.auth.ErrorText != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(wrapForWidth("Ошибка: "+m.auth.ErrorText, contentWidth)))
	}

	return lipgloss.NewStyle().
		Padding(0, 1).
		MaxWidth(max(1, m.width)).
		Render(box.Render(strings.Join(lines, "\n")))
}

func parseHashPayload(payload any) (string, error) {
	root, ok := payload.(map[string]any)
	if !ok {
		return "", errors.New("invalid sign up response")
	}
	hash := strings.TrimSpace(fmt.Sprint(root["hash"]))
	if hash == "" || hash == "<nil>" {
		return "", errors.New("hash was not returned by API")
	}
	return hash, nil
}

func parseAuthPayload(payload any) (string, int, error) {
	root, ok := payload.(map[string]any)
	if !ok {
		return "", 0, errors.New("invalid auth response")
	}

	token := ""
	userID := 0

	if tokenRaw, ok := root["token"].(string); ok && tokenRaw != "" {
		token = tokenRaw
	}

	if profileToken, ok := root["profileToken"].(map[string]any); ok {
		if token == "" {
			token = strings.TrimSpace(fmt.Sprint(profileToken["token"]))
		}
		if profile, ok := profileToken["profile"].(map[string]any); ok {
			userID = intFromAny(profile["id"])
		}
	}

	if userID == 0 {
		if profile, ok := root["profile"].(map[string]any); ok {
			userID = intFromAny(profile["id"])
		}
	}

	if token == "" || token == "<nil>" {
		return "", 0, errors.New("token was not returned by API")
	}
	return token, userID, nil
}

func trimLastRune(value string) string {
	if value == "" {
		return value
	}
	runes := []rune(value)
	if len(runes) <= 1 {
		return ""
	}
	return string(runes[:len(runes)-1])
}
