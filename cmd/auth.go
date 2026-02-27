package cmd

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)

func newAuthCommand() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Авторизация и управление сессией",
	}

	authCmd.AddCommand(
		newAuthLoginCommand(),
		newAuthSignupCommand(),
		newAuthVerifySignupCommand(),
		newAuthRestoreCommand(),
		newAuthVerifyRestoreCommand(),
		newAuthUseTokenCommand(),
		newAuthWhoamiCommand(),
		newAuthStatusCommand(),
		newAuthLogoutCommand(),
	)
	return authCmd
}

func newAuthLoginCommand() *cobra.Command {
	var login string
	var password string
	var save bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Вход по логину/email и паролю",
		RunE: func(cmd *cobra.Command, args []string) error {
			if login == "" || password == "" {
				return fmt.Errorf("flags --login and --password are required")
			}

			payload, err := rt.client.Do(withContext(), requestPOST(
				"/auth/signIn",
				url.Values{
					"login":    []string{login},
					"password": []string{password},
				},
				nil,
				false,
				"",
			))
			if err != nil {
				return err
			}

			if save {
				token, userID, ok := extractAuth(payload)
				if ok {
					if err := setAuthInConfig(token, userID); err != nil {
						return err
					}
				}
			}

			return printPayload(payload)
		},
	}

	cmd.Flags().StringVar(&login, "login", "", "Логин или email")
	cmd.Flags().StringVar(&password, "password", "", "Пароль")
	cmd.Flags().BoolVar(&save, "save", true, "Сохранить токен в конфиг")
	return cmd
}

func newAuthSignupCommand() *cobra.Command {
	var email string
	var login string
	var password string

	cmd := &cobra.Command{
		Use:   "signup",
		Short: "Регистрация аккаунта (этап запроса кода)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" || login == "" || password == "" {
				return fmt.Errorf("flags --email, --login and --password are required")
			}
			payload, err := rt.client.Do(withContext(), requestPOST(
				"/auth/signUp",
				url.Values{
					"email":    []string{email},
					"login":    []string{login},
					"password": []string{password},
				},
				nil,
				false,
				"",
			))
			if err != nil {
				return err
			}
			return printPayload(payload)
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "Email")
	cmd.Flags().StringVar(&login, "login", "", "Логин")
	cmd.Flags().StringVar(&password, "password", "", "Пароль")
	return cmd
}

func newAuthVerifySignupCommand() *cobra.Command {
	var email string
	var login string
	var password string
	var hash string
	var code string
	var save bool

	cmd := &cobra.Command{
		Use:   "verify-signup",
		Short: "Подтверждение регистрации кодом",
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" || login == "" || password == "" || hash == "" || code == "" {
				return fmt.Errorf("flags --email, --login, --password, --hash and --code are required")
			}
			payload, err := rt.client.Do(withContext(), requestPOST(
				"/auth/verify",
				url.Values{
					"email":    []string{email},
					"login":    []string{login},
					"password": []string{password},
					"hash":     []string{hash},
					"code":     []string{code},
				},
				nil,
				false,
				"",
			))
			if err != nil {
				return err
			}

			if save {
				token, userID, ok := extractAuth(payload)
				if ok {
					if err := setAuthInConfig(token, userID); err != nil {
						return err
					}
				}
			}

			return printPayload(payload)
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "Email")
	cmd.Flags().StringVar(&login, "login", "", "Логин")
	cmd.Flags().StringVar(&password, "password", "", "Пароль")
	cmd.Flags().StringVar(&hash, "hash", "", "Hash подтверждения")
	cmd.Flags().StringVar(&code, "code", "", "Код подтверждения")
	cmd.Flags().BoolVar(&save, "save", true, "Сохранить токен в конфиг")
	return cmd
}

func newAuthRestoreCommand() *cobra.Command {
	var login string

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Запрос кода восстановления пароля",
		RunE: func(cmd *cobra.Command, args []string) error {
			if login == "" {
				return fmt.Errorf("flag --login is required")
			}
			payload, err := rt.client.Do(withContext(), requestPOST(
				"/auth/restore",
				url.Values{
					"login": []string{login},
				},
				nil,
				false,
				"",
			))
			if err != nil {
				return err
			}
			return printPayload(payload)
		},
	}

	cmd.Flags().StringVar(&login, "login", "", "Логин или email")
	return cmd
}

func newAuthVerifyRestoreCommand() *cobra.Command {
	var login string
	var password string
	var hash string
	var code string
	var save bool

	cmd := &cobra.Command{
		Use:   "verify-restore",
		Short: "Подтверждение восстановления пароля кодом",
		RunE: func(cmd *cobra.Command, args []string) error {
			if login == "" || password == "" || code == "" {
				return fmt.Errorf("flags --login, --password and --code are required")
			}

			query := url.Values{
				"login":    []string{login},
				"password": []string{password},
				"code":     []string{code},
			}
			if hash != "" {
				query.Set("hash", hash)
			}

			payload, err := rt.client.Do(withContext(), requestPOST(
				"/auth/restore/verify",
				query,
				nil,
				false,
				"",
			))
			if err != nil {
				return err
			}

			if save {
				token, userID, ok := extractAuth(payload)
				if ok {
					if err := setAuthInConfig(token, userID); err != nil {
						return err
					}
				}
			}

			return printPayload(payload)
		},
	}

	cmd.Flags().StringVar(&login, "login", "", "Логин или email")
	cmd.Flags().StringVar(&password, "password", "", "Новый пароль")
	cmd.Flags().StringVar(&hash, "hash", "", "Hash подтверждения (опционально)")
	cmd.Flags().StringVar(&code, "code", "", "Код подтверждения")
	cmd.Flags().BoolVar(&save, "save", true, "Сохранить токен в конфиг")
	return cmd
}

func newAuthUseTokenCommand() *cobra.Command {
	var token string
	var userID int

	cmd := &cobra.Command{
		Use:   "use-token",
		Short: "Сохранить токен вручную",
		RunE: func(cmd *cobra.Command, args []string) error {
			if token == "" {
				return fmt.Errorf("flag --token is required")
			}
			if err := setAuthInConfig(token, userID); err != nil {
				return err
			}
			return printPayload(map[string]any{
				"status":  "ok",
				"message": "token saved",
				"user_id": userID,
			})
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "JWT токен")
	cmd.Flags().IntVar(&userID, "user-id", 0, "ID пользователя (опционально)")
	return cmd
}

func newAuthWhoamiCommand() *cobra.Command {
	var userID int

	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Проверить текущий токен и получить профиль",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := tokenRequired()
			if err != nil {
				return err
			}
			if userID == 0 {
				userID = rt.cfg.UserID
			}
			if userID == 0 {
				return fmt.Errorf("user id is unknown; pass --user-id or login again to store profile id")
			}
			payload, err := rt.client.Do(withContext(), requestGET(
				fmt.Sprintf("/profile/%d", userID),
				nil,
				false,
				token,
			))
			if err != nil {
				return err
			}
			return printPayload(payload)
		},
	}

	cmd.Flags().IntVar(&userID, "user-id", 0, "ID профиля (если не задан, берется из конфига)")
	return cmd
}

func newAuthStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Показать состояние текущей сессии",
		RunE: func(cmd *cobra.Command, args []string) error {
			return printPayload(map[string]any{
				"api_base_url": rt.baseURL,
				"user_agent":   rt.userAgent,
				"is_auth":      rt.token != "",
				"user_id":      rt.cfg.UserID,
				"token_saved":  rt.cfg.Token != "",
			})
		},
	}
	return cmd
}

func newAuthLogoutCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Удалить токен из конфига",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := clearAuthInConfig(); err != nil {
				return err
			}
			return printPayload(map[string]any{
				"status":  "ok",
				"message": "token removed",
			})
		},
	}
	return cmd
}

func extractAuth(payload any) (string, int, bool) {
	root, ok := payload.(map[string]any)
	if !ok {
		return "", 0, false
	}

	token := ""
	userID := 0

	if profileToken, ok := root["profileToken"].(map[string]any); ok {
		if tokenValue, ok := profileToken["token"].(string); ok {
			token = tokenValue
		}
		if profile, ok := profileToken["profile"].(map[string]any); ok {
			userID = intFromAny(profile["id"])
		}
	}

	if token == "" {
		if tokenValue, ok := root["token"].(string); ok {
			token = tokenValue
		}
	}
	if userID == 0 {
		if profile, ok := root["profile"].(map[string]any); ok {
			userID = intFromAny(profile["id"])
		}
	}
	if userID == 0 {
		userID = intFromAny(root["user_id"])
	}

	if token == "" {
		return "", 0, false
	}
	return token, userID, true
}
