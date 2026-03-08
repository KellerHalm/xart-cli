package cmd

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newBookmarksCommand() *cobra.Command {
	bookmarksCmd := &cobra.Command{
		Use:   "bookmarks",
		Short: "Закладки/списки просмотра",
	}

	bookmarksCmd.AddCommand(
		newBookmarksListCommand(),
		newBookmarksSetCommand(),
		newBookmarksSearchCommand(),
	)
	return bookmarksCmd
}

func newBookmarksListCommand() *cobra.Command {
	var listName string
	var page int
	var sortValue int
	var profileID int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Список закладок: watching|planned|watched|delayed|abandoned|favorite",
		RunE: func(cmd *cobra.Command, args []string) error {
			listID, isFavorite, err := parseBookmarkList(listName)
			if err != nil {
				return err
			}

			if isFavorite {
				token, err := mustTokenOrError()
				if err != nil {
					return err
				}
				query := url.Values{
					"sort": []string{strconv.Itoa(sortValue)},
				}
				return doAndPrint(requestGET(
					fmt.Sprintf("/favorite/all/%d", page),
					query,
					false,
					token,
				))
			}

			if profileID == 0 {
				profileID = rt.cfg.UserID
			}
			if profileID == 0 {
				return fmt.Errorf("profile id is required for bookmark lists, use --profile-id or login to cache user_id")
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

	cmd.Flags().StringVar(&listName, "list", "watching", "Тип списка: watching|planned|watched|delayed|abandoned|favorite")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	cmd.Flags().IntVar(&sortValue, "sort", 1, "Сортировка 1..6")
	cmd.Flags().IntVar(&profileID, "profile-id", 0, "ID профиля")
	return cmd
}

func newBookmarksSetCommand() *cobra.Command {
	var listName string
	var releaseID int

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Поставить релиз в список (включая not_watching)",
		RunE: func(cmd *cobra.Command, args []string) error {
			listID, isFavorite, err := parseBookmarkList(listName)
			if err != nil {
				return err
			}
			if isFavorite {
				return fmt.Errorf("favorite is not valid here; use `xart favorites add/remove`")
			}
			if releaseID <= 0 {
				return fmt.Errorf("flag --release-id is required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/profile/list/add/%d/%d", listID, releaseID),
				nil,
				false,
				token,
			))
		},
	}

	cmd.Flags().StringVar(&listName, "list", "watching", "Тип списка: not_watching|watching|planned|watched|delayed|abandoned")
	cmd.Flags().IntVar(&releaseID, "release-id", 0, "ID релиза")
	return cmd
}

func newBookmarksSearchCommand() *cobra.Command {
	var listName string
	var queryText string
	var page int

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Поиск внутри закладок/избранного",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			listID, isFavorite, err := parseBookmarkList(listName)
			if err != nil {
				return err
			}

			query := url.Values{}
			path := ""
			searchBy := listID
			if isFavorite {
				path = fmt.Sprintf("/search/favorites/%d", page)
				searchBy = 0
			} else {
				path = fmt.Sprintf("/search/profile/list/%d/%d", listID, page)
			}

			body := map[string]any{
				"query":    queryText,
				"searchBy": searchBy,
			}
			return doAndPrint(requestPOST(path, query, body, true, token))
		},
	}

	cmd.Flags().StringVar(&listName, "list", "watching", "Тип списка: watching|planned|watched|delayed|abandoned|favorite")
	cmd.Flags().StringVar(&queryText, "query", "", "Строка поиска")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	return cmd
}

func parseBookmarkList(value string) (int, bool, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	mapping := map[string]int{
		"not_watching": 0,
		"watching":     1,
		"planned":      2,
		"watched":      3,
		"delayed":      4,
		"abandoned":    5,
		"favorite":     -1,
	}
	if id, ok := mapping[normalized]; ok {
		if id == -1 {
			return 0, true, nil
		}
		return id, false, nil
	}

	parsed, err := strconv.Atoi(normalized)
	if err != nil {
		return 0, false, fmt.Errorf("unknown list %q", value)
	}
	if parsed < 0 || parsed > 5 {
		return 0, false, fmt.Errorf("list out of range: %d", parsed)
	}
	return parsed, false, nil
}
