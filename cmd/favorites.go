package cmd

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)

func newFavoritesCommand() *cobra.Command {
	favoritesCmd := &cobra.Command{
		Use:   "favorites",
		Short: "Избранные релизы",
	}

	favoritesCmd.AddCommand(
		newFavoritesListCommand(),
		newFavoritesAddCommand(),
		newFavoritesRemoveCommand(),
	)
	return favoritesCmd
}

func newFavoritesListCommand() *cobra.Command {
	var page int
	var sortValue int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Список избранного",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			query := url.Values{
				"sort": []string{fmt.Sprintf("%d", sortValue)},
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/favorite/all/%d", page),
				query,
				false,
				token,
			))
		},
	}

	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	cmd.Flags().IntVar(&sortValue, "sort", 1, "Сортировка 1..6")
	return cmd
}

func newFavoritesAddCommand() *cobra.Command {
	var releaseID int

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Добавить релиз в избранное",
		RunE: func(cmd *cobra.Command, args []string) error {
			if releaseID <= 0 {
				return fmt.Errorf("flag --release-id is required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/favorite/add/%d", releaseID),
				nil,
				false,
				token,
			))
		},
	}

	cmd.Flags().IntVar(&releaseID, "release-id", 0, "ID релиза")
	return cmd
}

func newFavoritesRemoveCommand() *cobra.Command {
	var releaseID int

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Удалить релиз из избранного",
		RunE: func(cmd *cobra.Command, args []string) error {
			if releaseID <= 0 {
				return fmt.Errorf("flag --release-id is required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/favorite/delete/%d", releaseID),
				nil,
				false,
				token,
			))
		},
	}

	cmd.Flags().IntVar(&releaseID, "release-id", 0, "ID релиза")
	return cmd
}
