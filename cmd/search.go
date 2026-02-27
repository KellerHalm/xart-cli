package cmd

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newSearchCommand() *cobra.Command {
	searchCmd := &cobra.Command{
		Use:   "search",
		Short: "Поиск по релизам, профилям, спискам и коллекциям",
	}

	searchCmd.AddCommand(
		newSearchReleasesCommand(),
		newSearchProfilesCommand(),
		newSearchListCommand(),
		newSearchHistoryCommand(),
		newSearchFavoritesCommand(),
		newSearchCollectionsCommand(),
		newSearchFavoriteCollectionsCommand(),
	)

	return searchCmd
}

func newSearchReleasesCommand() *cobra.Command {
	var query string
	var page int
	var by string

	byMap := map[string]int{
		"name":     0,
		"studio":   1,
		"director": 2,
		"author":   3,
		"tag":      4,
	}

	cmd := &cobra.Command{
		Use:   "releases",
		Short: "Поиск по релизам",
		RunE: func(cmd *cobra.Command, args []string) error {
			searchBy, err := parseSearchBy(by, byMap, 0)
			if err != nil {
				return err
			}
			return runSearch(
				fmt.Sprintf("/search/releases/%d", page),
				query,
				searchBy,
				false,
			)
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Строка поиска")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	cmd.Flags().StringVar(&by, "by", "name", "Поле поиска: name|studio|director|author|tag или число")
	return cmd
}

func newSearchProfilesCommand() *cobra.Command {
	var query string
	var page int

	cmd := &cobra.Command{
		Use:   "profiles",
		Short: "Поиск по профилям",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(
				fmt.Sprintf("/search/profiles/%d", page),
				query,
				0,
				false,
			)
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Строка поиска")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	return cmd
}

func newSearchListCommand() *cobra.Command {
	var query string
	var page int
	var by string

	byMap := map[string]int{
		"watching":  1,
		"planned":   2,
		"watched":   3,
		"delayed":   4,
		"abandoned": 5,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Поиск по своим спискам",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			searchBy, err := parseSearchBy(by, byMap, 1)
			if err != nil {
				return err
			}
			return runSearchWithToken(
				fmt.Sprintf("/search/profile/list/%d/%d", searchBy, page),
				query,
				searchBy,
				token,
			)
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Строка поиска")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	cmd.Flags().StringVar(&by, "by", "watching", "Список: watching|planned|watched|delayed|abandoned или число")
	return cmd
}

func newSearchHistoryCommand() *cobra.Command {
	var query string
	var page int

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Поиск в истории просмотра (требуется авторизация)",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return runSearchWithToken(
				fmt.Sprintf("/search/history/%d", page),
				query,
				0,
				token,
			)
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Строка поиска")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	return cmd
}

func newSearchFavoritesCommand() *cobra.Command {
	var query string
	var page int

	cmd := &cobra.Command{
		Use:   "favorites",
		Short: "Поиск по избранному (требуется авторизация)",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return runSearchWithToken(
				fmt.Sprintf("/search/favorites/%d", page),
				query,
				0,
				token,
			)
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Строка поиска")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	return cmd
}

func newSearchCollectionsCommand() *cobra.Command {
	var query string
	var page int

	cmd := &cobra.Command{
		Use:   "collections",
		Short: "Поиск по всем коллекциям",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(
				fmt.Sprintf("/search/collections/%d", page),
				query,
				0,
				false,
			)
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Строка поиска")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	return cmd
}

func newSearchFavoriteCollectionsCommand() *cobra.Command {
	var query string
	var page int

	cmd := &cobra.Command{
		Use:   "favorite-collections",
		Short: "Поиск по своим коллекциям (требуется авторизация)",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return runSearchWithToken(
				fmt.Sprintf("/search/favoriteCollections/%d", page),
				query,
				0,
				token,
			)
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Строка поиска")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	return cmd
}

func runSearch(path string, queryText string, searchBy int, requireAuth bool) error {
	token := ""
	var err error
	if requireAuth {
		token, err = mustTokenOrError()
		if err != nil {
			return err
		}
	} else {
		token = tokenOptional()
	}
	return runSearchWithToken(path, queryText, searchBy, token)
}

func runSearchWithToken(path string, queryText string, searchBy int, token string) error {
	query := url.Values{}
	if token != "" {
		query.Set("token", token)
	}
	body := map[string]any{
		"query":    queryText,
		"searchBy": searchBy,
	}
	return doAndPrint(requestPOST(path, query, body, true, ""))
}

func parseSearchBy(value string, mapping map[string]int, fallback int) (int, error) {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return fallback, nil
	}
	if mapped, ok := mapping[trimmed]; ok {
		return mapped, nil
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("invalid --by value %q", value)
	}
	return parsed, nil
}
