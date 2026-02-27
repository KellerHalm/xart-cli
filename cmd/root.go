package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"xart-cli/internal/config"
	"xart-cli/internal/xart"
)

type Runtime struct {
	cfg       config.Config
	client    *xart.Client
	token     string
	baseURL   string
	userAgent string
	rawOutput bool
}

var (
	rootCmd = &cobra.Command{
		Use:   "xart",
		Short: "CLI-клиент Anixart/Xart",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			baseURL, _ := cmd.Flags().GetString("base-url")
			userAgent, _ := cmd.Flags().GetString("user-agent")
			tokenOverride, _ := cmd.Flags().GetString("token")
			rawOutput, _ := cmd.Flags().GetBool("raw")

			if baseURL == "" {
				baseURL = cfg.APIBaseURL
			}
			if userAgent == "" {
				userAgent = cfg.UserAgent
			}
			if baseURL == "" {
				baseURL = config.DefaultAPIBaseURL
			}
			if userAgent == "" {
				userAgent = config.DefaultUserAgent
			}

			token := cfg.Token
			if tokenOverride != "" {
				token = tokenOverride
			}

			rt = &Runtime{
				cfg:       cfg,
				client:    xart.NewClient(baseURL, userAgent),
				token:     token,
				baseURL:   baseURL,
				userAgent: userAgent,
				rawOutput: rawOutput,
			}
			return nil
		},
	}
	rt *Runtime
)

func Execute() error {
	rootCmd.SetErrPrefix("")
	return rootCmd.Execute()
}

func init() {
	defaultCfg := config.Default()
	rootCmd.PersistentFlags().String("base-url", defaultCfg.APIBaseURL, "Base URL API")
	rootCmd.PersistentFlags().String("user-agent", defaultCfg.UserAgent, "User-Agent для запросов")
	rootCmd.PersistentFlags().String("token", "", "Переопределить токен (иначе берется из конфига)")
	rootCmd.PersistentFlags().Bool("raw", false, "Выводить без pretty JSON")

	rootCmd.AddCommand(
		newUICommand(),
		newWatchCommand(),
		newAuthCommand(),
		newHomeCommand(),
		newDiscoverCommand(),
		newSearchCommand(),
		newReleaseCommand(),
		newBookmarksCommand(),
		newFavoritesCommand(),
		newHistoryCommand(),
		newCollectionsCommand(),
		newProfileCommand(),
		newAPICommand(),
	)
}

func withContext() context.Context {
	return context.Background()
}

func printPayload(value any) error {
	if rt == nil {
		return errors.New("runtime is not initialized")
	}
	if rt.rawOutput {
		fmt.Printf("%v\n", value)
		return nil
	}

	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode output: %w", err)
	}
	fmt.Println(string(encoded))
	return nil
}

func tokenRequired() (string, error) {
	if rt == nil {
		return "", errors.New("runtime is not initialized")
	}
	if rt.token == "" {
		return "", errors.New("нужна авторизация: выполните `xart auth login` или передайте `--token`")
	}
	return rt.token, nil
}

func tokenOptional() string {
	if rt == nil {
		return ""
	}
	return rt.token
}

func saveCurrentConfig() error {
	if rt == nil {
		return errors.New("runtime is not initialized")
	}
	rt.cfg.APIBaseURL = rt.baseURL
	rt.cfg.UserAgent = rt.userAgent
	if rt.token != "" {
		rt.cfg.Token = rt.token
	}
	return config.Save(rt.cfg)
}

func clearAuthInConfig() error {
	if rt == nil {
		return errors.New("runtime is not initialized")
	}
	rt.token = ""
	rt.cfg.Token = ""
	rt.cfg.UserID = 0
	return config.Save(rt.cfg)
}

func setAuthInConfig(token string, userID int) error {
	if rt == nil {
		return errors.New("runtime is not initialized")
	}
	rt.token = token
	rt.cfg.Token = token
	rt.cfg.UserID = userID
	return config.Save(rt.cfg)
}

func parseQueryValues(values []string) (url.Values, error) {
	result := url.Values{}
	for _, kv := range values {
		key, value, ok := strings.Cut(kv, "=")
		if !ok {
			return nil, fmt.Errorf("invalid query format %q, expected key=value", kv)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return nil, fmt.Errorf("empty query key in %q", kv)
		}
		result.Add(key, value)
	}
	return result, nil
}

func parseJSONBody(body string, bodyFile string) (any, error) {
	switch {
	case body != "" && bodyFile != "":
		return nil, errors.New("используйте только один источник тела: --body или --body-file")
	case body == "" && bodyFile == "":
		return nil, nil
	}

	var raw []byte
	var err error
	if body != "" {
		raw = []byte(body)
	} else {
		raw, err = osReadFile(bodyFile)
		if err != nil {
			return nil, err
		}
	}

	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("parse json body: %w", err)
	}
	return payload, nil
}

func osReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func parseCommaIntList(value string) ([]int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parts := strings.Split(trimmed, ",")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid integer %q: %w", part, err)
		}
		out = append(out, n)
	}
	return out, nil
}

func mapKeys(value map[string]int) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
