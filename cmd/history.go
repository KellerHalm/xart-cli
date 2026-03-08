package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newHistoryCommand() *cobra.Command {
	historyCmd := &cobra.Command{
		Use:   "history",
		Short: "История просмотра",
	}

	historyCmd.AddCommand(newHistoryListCommand())
	return historyCmd
}

func newHistoryListCommand() *cobra.Command {
	var page int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Показать историю просмотра",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/history/%d", page),
				nil,
				false,
				token,
			))
		},
	}
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	return cmd
}
