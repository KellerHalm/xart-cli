package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"xart-cli/internal/player"
)

func newWatchCommand() *cobra.Command {
	var releaseID int
	var voiceoverID int
	var sourceID int
	var episodePosition int
	var playerExecutable string
	var playerArgs []string
	var printURL bool
	var noProgress bool

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Смотреть тайтл в локальном плеере прямо из терминала",
		RunE: func(cmd *cobra.Command, args []string) error {
			if releaseID <= 0 {
				return fmt.Errorf("flag --id is required")
			}

			selection, err := player.ResolveSelection(
				withContext(),
				rt.client,
				releaseID,
				tokenOptional(),
				voiceoverID,
				sourceID,
				episodePosition,
			)
			if err != nil {
				return err
			}

			if printURL {
				fmt.Println(selection.Episode.URL)
				return nil
			}

			if !noProgress {
				if err := player.MarkEpisodeProgress(
					withContext(),
					rt.client,
					tokenOptional(),
					selection.ReleaseID,
					selection.Source.ID,
					selection.Episode.Position,
				); err != nil {
					// Non-fatal: playback can still continue.
					fmt.Fprintf(os.Stderr, "warning: failed to mark watch progress: %v\n", err)
				}
			}

			launch, err := player.BuildLaunchPlan(selection.Episode.URL, player.LaunchOptions{
				Player:    playerExecutable,
				ExtraArgs: playerArgs,
			})
			if err != nil {
				return err
			}

			fmt.Printf(
				"Release %d | Озвучка: %s (%d) | Источник: %s (%d) | Эпизод: %s (%d)\n",
				selection.ReleaseID,
				selection.Voiceover.Name,
				selection.Voiceover.ID,
				selection.Source.Name,
				selection.Source.ID,
				selection.Episode.Name,
				selection.Episode.Position,
			)
			fmt.Printf("Starting player: %s\n", launch.PlayerName)

			playerCmd := launch.Command()
			playerCmd.Stdin = os.Stdin
			playerCmd.Stdout = os.Stdout
			playerCmd.Stderr = os.Stderr
			return playerCmd.Run()
		},
	}

	cmd.Flags().IntVar(&releaseID, "id", 0, "ID релиза")
	cmd.Flags().IntVar(&voiceoverID, "voiceover-id", 0, "ID озвучки (по умолчанию первая доступная)")
	cmd.Flags().IntVar(&sourceID, "source-id", 0, "ID источника/плеера (по умолчанию первый доступный)")
	cmd.Flags().IntVar(&episodePosition, "episode", -1, "Позиция эпизода (по умолчанию последний доступный)")
	cmd.Flags().StringVar(&playerExecutable, "player", "", "Плеер: mpv|vlc|ffplay или путь к исполняемому файлу")
	cmd.Flags().StringArrayVar(&playerArgs, "player-arg", nil, "Дополнительный аргумент плеера (можно повторять)")
	cmd.Flags().BoolVar(&printURL, "print-url", false, "Только вывести URL выбранного эпизода")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "Не отмечать эпизод в истории/просмотренном")
	return cmd
}
