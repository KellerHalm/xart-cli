package cmd

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"xart-cli/internal/xart"
)

func newProfileCommand() *cobra.Command {
	profileCmd := &cobra.Command{
		Use:   "profile",
		Short: "Профиль, друзья, блокировки, настройки",
	}

	profileCmd.AddCommand(
		newProfileInfoCommand(),
		newProfileBookmarksCommand(),
		newProfileCollectionsCommand(),
		newProfileRatingsCommand(),
		newProfileFriendsCommand(),
		newProfileFriendRequestsCommand(),
		newProfileFriendAddCommand(),
		newProfileFriendRemoveCommand(),
		newProfileFriendHideCommand(),
		newProfileBlocklistCommand(),
		newProfileBlockAddCommand(),
		newProfileBlockRemoveCommand(),
		newProfileSettingsMyCommand(),
		newProfileSettingsLoginInfoCommand(),
		newProfileSettingsLoginHistoryCommand(),
		newProfileSettingsLoginChangeCommand(),
		newProfileSettingsStatusCommand(),
		newProfileSettingsSocialInfoCommand(),
		newProfileSettingsSocialEditCommand(),
		newProfileSettingsPrivacyCommand(),
		newProfileSettingsAvatarCommand(),
	)

	return profileCmd
}

func newProfileInfoCommand() *cobra.Command {
	var id int
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Профиль пользователя",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 {
				return fmt.Errorf("flag --id is required")
			}
			query := url.Values{}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestGET(fmt.Sprintf("/profile/%d", id), query, false, ""))
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "ID профиля")
	return cmd
}

func newProfileBookmarksCommand() *cobra.Command {
	var profileID int
	var listName string
	var page int
	var sortValue int

	cmd := &cobra.Command{
		Use:   "bookmarks",
		Short: "Закладки профиля",
		RunE: func(cmd *cobra.Command, args []string) error {
			if profileID <= 0 {
				return fmt.Errorf("flag --profile-id is required")
			}
			listID, isFavorite, err := parseBookmarkList(listName)
			if err != nil {
				return err
			}
			if isFavorite {
				return fmt.Errorf("favorite list is not supported for profile bookmarks endpoint")
			}
			query := url.Values{
				"sort": []string{strconv.Itoa(sortValue)},
			}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/profile/list/all/%d/%d/%d", profileID, listID, page),
				query,
				false,
				"",
			))
		},
	}

	cmd.Flags().IntVar(&profileID, "profile-id", 0, "ID профиля")
	cmd.Flags().StringVar(&listName, "list", "watching", "Список: watching|planned|watched|delayed|abandoned")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	cmd.Flags().IntVar(&sortValue, "sort", 1, "Сортировка")
	return cmd
}

func newProfileCollectionsCommand() *cobra.Command {
	var profileID int
	var page int
	cmd := &cobra.Command{
		Use:   "collections",
		Short: "Коллекции профиля",
		RunE: func(cmd *cobra.Command, args []string) error {
			if profileID <= 0 {
				return fmt.Errorf("flag --profile-id is required")
			}
			query := url.Values{}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/collection/all/profile/%d/%d", profileID, page),
				query,
				false,
				"",
			))
		},
	}
	cmd.Flags().IntVar(&profileID, "profile-id", 0, "ID профиля")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	return cmd
}

func newProfileRatingsCommand() *cobra.Command {
	var profileID int
	var page int
	var sortValue int

	cmd := &cobra.Command{
		Use:   "ratings",
		Short: "Оценки релизов профиля",
		RunE: func(cmd *cobra.Command, args []string) error {
			if profileID <= 0 {
				return fmt.Errorf("flag --profile-id is required")
			}
			query := url.Values{
				"sort": []string{strconv.Itoa(sortValue)},
			}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/profile/vote/release/voted/%d/%d", profileID, page),
				query,
				false,
				"",
			))
		},
	}
	cmd.Flags().IntVar(&profileID, "profile-id", 0, "ID профиля")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	cmd.Flags().IntVar(&sortValue, "sort", 1, "Сортировка")
	return cmd
}

func newProfileFriendsCommand() *cobra.Command {
	var profileID int
	var page int
	cmd := &cobra.Command{
		Use:   "friends",
		Short: "Список друзей профиля",
		RunE: func(cmd *cobra.Command, args []string) error {
			if profileID <= 0 {
				return fmt.Errorf("flag --profile-id is required")
			}
			query := url.Values{}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/profile/friend/all/%d/%d", profileID, page),
				query,
				false,
				"",
			))
		},
	}
	cmd.Flags().IntVar(&profileID, "profile-id", 0, "ID профиля")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	return cmd
}

func newProfileFriendRequestsCommand() *cobra.Command {
	var requestType string
	var mode string
	var page int
	var count int

	cmd := &cobra.Command{
		Use:   "friend-requests",
		Short: "Входящие/исходящие заявки в друзья",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			reqType := strings.ToLower(strings.TrimSpace(requestType))
			if reqType != "in" && reqType != "out" {
				return fmt.Errorf("--type must be in|out")
			}
			mode = strings.ToLower(strings.TrimSpace(mode))
			path := ""
			query := url.Values{}
			if mode == "last" {
				path = fmt.Sprintf("/profile/friend/requests/%s/last", reqType)
				query.Set("count", strconv.Itoa(count))
			} else if mode == "all" {
				path = fmt.Sprintf("/profile/friend/requests/%s/all/%d", reqType, page)
			} else {
				return fmt.Errorf("--mode must be last|all")
			}
			return doAndPrint(requestGET(path, query, false, token))
		},
	}

	cmd.Flags().StringVar(&requestType, "type", "in", "Тип: in|out")
	cmd.Flags().StringVar(&mode, "mode", "last", "Режим: last|all")
	cmd.Flags().IntVar(&page, "page", 0, "Страница (для mode=all)")
	cmd.Flags().IntVar(&count, "count", 8, "Количество (для mode=last)")
	return cmd
}

func newProfileFriendAddCommand() *cobra.Command {
	var profileID int
	cmd := &cobra.Command{
		Use:   "friend-add",
		Short: "Отправить/принять заявку в друзья",
		RunE: func(cmd *cobra.Command, args []string) error {
			if profileID <= 0 {
				return fmt.Errorf("flag --profile-id is required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/profile/friend/request/send/%d", profileID),
				nil,
				false,
				token,
			))
		},
	}
	cmd.Flags().IntVar(&profileID, "profile-id", 0, "ID профиля")
	return cmd
}

func newProfileFriendRemoveCommand() *cobra.Command {
	var profileID int
	cmd := &cobra.Command{
		Use:   "friend-remove",
		Short: "Удалить из друзей/отклонить заявку",
		RunE: func(cmd *cobra.Command, args []string) error {
			if profileID <= 0 {
				return fmt.Errorf("flag --profile-id is required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/profile/friend/request/remove/%d", profileID),
				nil,
				false,
				token,
			))
		},
	}
	cmd.Flags().IntVar(&profileID, "profile-id", 0, "ID профиля")
	return cmd
}

func newProfileFriendHideCommand() *cobra.Command {
	var profileID int
	cmd := &cobra.Command{
		Use:   "friend-hide",
		Short: "Скрыть входящую заявку",
		RunE: func(cmd *cobra.Command, args []string) error {
			if profileID <= 0 {
				return fmt.Errorf("flag --profile-id is required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/profile/friend/request/hide/%d", profileID),
				nil,
				false,
				token,
			))
		},
	}
	cmd.Flags().IntVar(&profileID, "profile-id", 0, "ID профиля")
	return cmd
}

func newProfileBlocklistCommand() *cobra.Command {
	var page int
	cmd := &cobra.Command{
		Use:   "blocklist",
		Short: "Список заблокированных пользователей",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/profile/blocklist/all/%d", page),
				nil,
				false,
				token,
			))
		},
	}
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	return cmd
}

func newProfileBlockAddCommand() *cobra.Command {
	var profileID int
	cmd := &cobra.Command{
		Use:   "block-add",
		Short: "Заблокировать пользователя",
		RunE: func(cmd *cobra.Command, args []string) error {
			if profileID <= 0 {
				return fmt.Errorf("flag --profile-id is required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/profile/blocklist/add/%d", profileID),
				nil,
				false,
				token,
			))
		},
	}
	cmd.Flags().IntVar(&profileID, "profile-id", 0, "ID профиля")
	return cmd
}

func newProfileBlockRemoveCommand() *cobra.Command {
	var profileID int
	cmd := &cobra.Command{
		Use:   "block-remove",
		Short: "Разблокировать пользователя",
		RunE: func(cmd *cobra.Command, args []string) error {
			if profileID <= 0 {
				return fmt.Errorf("flag --profile-id is required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/profile/blocklist/remove/%d", profileID),
				nil,
				false,
				token,
			))
		},
	}
	cmd.Flags().IntVar(&profileID, "profile-id", 0, "ID профиля")
	return cmd
}

func newProfileSettingsMyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "settings-my",
		Short: "Мои настройки профиля",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET("/profile/preference/my", nil, false, token))
		},
	}
}

func newProfileSettingsLoginInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "settings-login-info",
		Short: "Текущий логин и лимиты смены",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET("/profile/preference/login/info", nil, false, token))
		},
	}
}

func newProfileSettingsLoginHistoryCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "settings-login-history",
		Short: "История логинов",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET("/profile/login/history/all", nil, false, token))
		},
	}
}

func newProfileSettingsLoginChangeCommand() *cobra.Command {
	var login string
	cmd := &cobra.Command{
		Use:   "settings-login-change",
		Short: "Сменить логин",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(login) == "" {
				return fmt.Errorf("flag --login is required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			query := url.Values{
				"login": []string{strings.TrimSpace(login)},
			}
			return doAndPrint(requestGET("/profile/preference/login/change", query, false, token))
		},
	}
	cmd.Flags().StringVar(&login, "login", "", "Новый логин")
	return cmd
}

func newProfileSettingsStatusCommand() *cobra.Command {
	var status string
	cmd := &cobra.Command{
		Use:   "settings-status",
		Short: "Обновить статус профиля",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			body := map[string]any{"status": status}
			return doAndPrint(requestPOST("/profile/preference/status/edit", nil, body, false, token))
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "Текст статуса")
	return cmd
}

func newProfileSettingsSocialInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "settings-social-info",
		Short: "Получить текущие соцсети",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET("/profile/preference/social", nil, false, token))
		},
	}
}

func newProfileSettingsSocialEditCommand() *cobra.Command {
	var vk string
	var tg string
	var discord string
	var inst string
	var tt string

	cmd := &cobra.Command{
		Use:   "settings-social-edit",
		Short: "Обновить соцсети профиля",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			body := map[string]any{
				"vkPage":      strings.TrimSpace(vk),
				"tgPage":      strings.TrimSpace(tg),
				"discordPage": strings.TrimSpace(discord),
				"instPage":    strings.TrimSpace(inst),
				"ttPage":      strings.TrimSpace(tt),
			}
			return doAndPrint(requestPOST("/profile/preference/social/edit", nil, body, false, token))
		},
	}

	cmd.Flags().StringVar(&vk, "vk", "", "VK")
	cmd.Flags().StringVar(&tg, "tg", "", "Telegram")
	cmd.Flags().StringVar(&discord, "discord", "", "Discord")
	cmd.Flags().StringVar(&inst, "inst", "", "Instagram")
	cmd.Flags().StringVar(&tt, "tt", "", "TikTok")
	return cmd
}

func newProfileSettingsPrivacyCommand() *cobra.Command {
	var kind string
	var permission int

	paths := map[string]string{
		"stats":           "/profile/preference/privacy/stats/edit",
		"counts":          "/profile/preference/privacy/counts/edit",
		"social":          "/profile/preference/privacy/social/edit",
		"friend_requests": "/profile/preference/privacy/friendRequests/edit",
	}

	cmd := &cobra.Command{
		Use:   "settings-privacy",
		Short: "Обновить privacy-параметр",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			path, ok := paths[strings.ToLower(strings.TrimSpace(kind))]
			if !ok {
				return fmt.Errorf("unknown --kind, expected one of: stats|counts|social|friend_requests")
			}
			body := map[string]any{"permission": permission}
			return doAndPrint(requestPOST(path, nil, body, false, token))
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "stats", "Тип настройки: stats|counts|social|friend_requests")
	cmd.Flags().IntVar(&permission, "permission", 0, "Значение permission")
	return cmd
}

func newProfileSettingsAvatarCommand() *cobra.Command {
	var filePath string

	cmd := &cobra.Command{
		Use:   "settings-avatar",
		Short: "Обновить аватар профиля",
		RunE: func(cmd *cobra.Command, args []string) error {
			if filePath == "" {
				return fmt.Errorf("flag --file is required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}

			fileData, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("read avatar file: %w", err)
			}

			var buffer bytes.Buffer
			writer := multipart.NewWriter(&buffer)
			part, err := writer.CreateFormFile("image", "avatar.jpg")
			if err != nil {
				return err
			}
			if _, err := part.Write(fileData); err != nil {
				return err
			}
			if err := writer.WriteField("name", "image"); err != nil {
				return err
			}
			if err := writer.Close(); err != nil {
				return err
			}

			payload, err := rt.client.Do(withContext(), xart.Request{
				Method:      "POST",
				Path:        "/profile/preference/avatar/edit",
				Body:        buffer.Bytes(),
				ContentType: writer.FormDataContentType(),
				Token:       token,
			})
			if err != nil {
				return err
			}
			return printPayload(payload)
		},
	}

	cmd.Flags().StringVar(&filePath, "file", "", "Путь к изображению")
	return cmd
}
