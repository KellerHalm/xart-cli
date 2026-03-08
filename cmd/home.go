package cmd

import (
	"fmt"
	"net/url"
	"sort"

	"github.com/spf13/cobra"
)

func newHomeCommand() *cobra.Command {
	homeCmd := &cobra.Command{
		Use:   "home",
		Short: "Главные категории Xart (как на /home/*)",
	}

	homeCmd.AddCommand(
		newHomeListCommand(),
		newHomeFilterCommand(),
	)
	return homeCmd
}

func newHomeListCommand() *cobra.Command {
	var page int

	categories := map[string]func(map[string]any){
		"last":     func(_ map[string]any) {},
		"ongoing":  func(filter map[string]any) { filter["status_id"] = 2 },
		"announce": func(filter map[string]any) { filter["status_id"] = 3 },
		"finished": func(filter map[string]any) { filter["status_id"] = 1 },
		"films":    func(filter map[string]any) { filter["category_id"] = 2 },
	}

	cmd := &cobra.Command{
		Use:   "list <category>",
		Short: "Категории: last|ongoing|announce|finished|films",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			modifier, ok := categories[args[0]]
			if !ok {
				return fmt.Errorf("unknown category %q, available: %v", args[0], mapKeysInt(categories))
			}

			filter := defaultFilter()
			modifier(filter)

			query := url.Values{}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}

			return doAndPrint(requestPOST(
				fmt.Sprintf("/filter/%d", page),
				query,
				filter,
				false,
				"",
			))
		},
	}

	cmd.Flags().IntVar(&page, "page", 0, "Номер страницы")
	return cmd
}

func newHomeFilterCommand() *cobra.Command {
	var page int
	var body string
	var bodyFile string

	cmd := &cobra.Command{
		Use:   "filter",
		Short: "Кастомный фильтр (POST /filter/{page})",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := parseJSONBody(body, bodyFile)
			if err != nil {
				return err
			}
			if payload == nil {
				payload = defaultFilter()
			}

			query := url.Values{}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestPOST(
				fmt.Sprintf("/filter/%d", page),
				query,
				payload,
				false,
				"",
			))
		},
	}

	cmd.Flags().IntVar(&page, "page", 0, "Номер страницы")
	cmd.Flags().StringVar(&body, "body", "", "JSON-тело фильтра")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Файл с JSON-телом фильтра")
	return cmd
}

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

func mapKeysInt[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
