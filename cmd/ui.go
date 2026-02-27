package cmd

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"xart-cli/internal/tui"
)

func newUICommand() *cobra.Command {
	var category string
	var page int

	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Interactive title cards UI",
		RunE: func(cmd *cobra.Command, args []string) error {
			callbacks := tui.AuthCallbacks{
				SaveLogin: func(token string, userID int) error {
					return setAuthInConfig(token, userID)
				},
				SaveLogout: func() error {
					return clearAuthInConfig()
				},
			}
			model := tui.NewModel(rt.client, rt.token, rt.cfg.UserID, category, page, callbacks)
			program := tea.NewProgram(model, tea.WithAltScreen())
			_, err := program.Run()
			return err
		},
	}

	cmd.Flags().StringVar(&category, "category", "last", "Category: last|ongoing|announce|finished|films|favorite|watching|planned|watched|delayed|abandoned")
	cmd.Flags().IntVar(&page, "page", 0, "Page")
	return cmd
}
