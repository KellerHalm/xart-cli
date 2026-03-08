package cmd

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)

func newDiscoverCommand() *cobra.Command {
	discoverCmd := &cobra.Command{
		Use:   "discover",
		Short: "Раздел discovery сайта",
	}

	discoverCmd.AddCommand(
		newDiscoverInterestingCommand(),
		newDiscoverDiscussingCommand(),
		newDiscoverWatchingCommand(),
		newDiscoverRecommendationsCommand(),
		newDiscoverCollectionsCommand(),
		newDiscoverScheduleCommand(),
		newDiscoverFilterTypesCommand(),
	)

	return discoverCmd
}

func newDiscoverInterestingCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "interesting",
		Short: "Интересное (GET /discover/interesting)",
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestGET("/discover/interesting", query, false, ""))
		},
	}
}

func newDiscoverDiscussingCommand() *cobra.Command {
	var page int

	cmd := &cobra.Command{
		Use:   "discussing",
		Short: "Обсуждаемое сейчас (GET /discover/discussing)",
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			path := "/discover/discussing"
			if page >= 0 {
				path = fmt.Sprintf("/discover/discussing/%d", page)
			}
			return doAndPrint(requestGET(path, query, false, ""))
		},
	}

	cmd.Flags().IntVar(&page, "page", -1, "Страница (если >=0, используется /discover/discussing/{page})")
	return cmd
}

func newDiscoverWatchingCommand() *cobra.Command {
	var page int
	cmd := &cobra.Command{
		Use:   "watching",
		Short: "Смотрят сейчас (GET /discover/watching/{page})",
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/discover/watching/%d", page),
				query,
				false,
				"",
			))
		},
	}
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	return cmd
}

func newDiscoverRecommendationsCommand() *cobra.Command {
	var page int
	var previousPage int

	cmd := &cobra.Command{
		Use:   "recommendations",
		Short: "Рекомендации (требуется авторизация)",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			query := url.Values{
				"previous_page": []string{fmt.Sprintf("%d", previousPage)},
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/discover/recommendations/%d", page),
				query,
				false,
				token,
			))
		},
	}

	cmd.Flags().IntVar(&page, "page", -1, "Страница")
	cmd.Flags().IntVar(&previousPage, "previous-page", -1, "Предыдущая страница")
	return cmd
}

func newDiscoverCollectionsCommand() *cobra.Command {
	var page int
	var previousPage int
	var where int
	var sortValue int

	cmd := &cobra.Command{
		Use:   "collections",
		Short: "Коллекции недели/популярные (GET /collection/all/{page})",
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{
				"where":         []string{fmt.Sprintf("%d", where)},
				"sort":          []string{fmt.Sprintf("%d", sortValue)},
				"previous_page": []string{fmt.Sprintf("%d", previousPage)},
			}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/collection/all/%d", page),
				query,
				false,
				"",
			))
		},
	}

	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	cmd.Flags().IntVar(&previousPage, "previous-page", 0, "Предыдущая страница")
	cmd.Flags().IntVar(&where, "where", 1, "Параметр where")
	cmd.Flags().IntVar(&sortValue, "sort", 4, "Сортировка 1..6")
	return cmd
}

func newDiscoverScheduleCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "schedule",
		Short: "Расписание релизов (GET /schedule)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return doAndPrint(requestGET("/schedule", nil, false, ""))
		},
	}
}

func newDiscoverFilterTypesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "filter-types",
		Short: "Типы фильтров (GET /type/all)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return doAndPrint(requestGET("/type/all", nil, false, ""))
		},
	}
}
