package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

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
	var choosePlayer bool
	var chooseVoiceover bool
	var printURL bool
	var noProgress bool

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Смотреть тайтл в локальном плеере прямо из терминала",
		RunE: func(cmd *cobra.Command, args []string) error {
			if releaseID <= 0 {
				return fmt.Errorf("flag --id is required")
			}
			if choosePlayer && strings.TrimSpace(playerExecutable) != "" {
				return fmt.Errorf("flags --choose-player and --player are mutually exclusive")
			}
			if chooseVoiceover && voiceoverID > 0 {
				return fmt.Errorf("flags --choose-voiceover and --voiceover-id are mutually exclusive")
			}

			selectedVoiceoverID := voiceoverID
			if chooseVoiceover {
				var err error
				selectedVoiceoverID, err = chooseVoiceoverInteractively(releaseID)
				if err != nil {
					return err
				}
			}

			selection, err := player.ResolveSelection(
				withContext(),
				rt.client,
				releaseID,
				tokenOptional(),
				selectedVoiceoverID,
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

			selectedPlayer := playerExecutable
			if choosePlayer {
				selectedPlayer, err = choosePlayerInteractively()
				if err != nil {
					return err
				}
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
				Player:    selectedPlayer,
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
	cmd.Flags().BoolVar(&choosePlayer, "choose-player", false, "Выбрать плеер интерактивно перед запуском")
	cmd.Flags().BoolVar(&chooseVoiceover, "choose-voiceover", false, "Выбрать озвучку интерактивно перед запуском")
	cmd.Flags().StringArrayVar(&playerArgs, "player-arg", nil, "Дополнительный аргумент плеера (можно повторять)")
	cmd.Flags().BoolVar(&printURL, "print-url", false, "Только вывести URL выбранного эпизода")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "Не отмечать эпизод в истории/просмотренном")
	return cmd
}

func choosePlayerInteractively() (string, error) {
	available := player.DetectAvailablePlayers()
	if len(available) == 0 {
		return "", fmt.Errorf("player not found; install one of: mpv, vlc, ffplay")
	}

	fmt.Println("Choose player:")
	fmt.Println("  0) auto")
	for i, name := range available {
		fmt.Printf("  %d) %s\n", i+1, name)
	}

	choice, err := readInteractiveChoice(len(available))
	if err != nil {
		return "", err
	}
	if choice == 0 {
		return "", nil
	}
	return available[choice-1], nil
}

func chooseVoiceoverInteractively(releaseID int) (int, error) {
	voiceovers, err := player.ListVoiceovers(withContext(), rt.client, releaseID)
	if err != nil {
		return 0, err
	}
	if len(voiceovers) == 0 {
		return 0, fmt.Errorf("voiceovers not found")
	}

	fmt.Println("Choose voiceover:")
	fmt.Println("  0) auto (first available)")
	for i, voiceover := range voiceovers {
		fmt.Printf("  %d) %s (id=%d)\n", i+1, voiceover.Name, voiceover.ID)
	}

	choice, err := readInteractiveChoice(len(voiceovers))
	if err != nil {
		return 0, err
	}
	if choice == 0 {
		return 0, nil
	}
	return voiceovers[choice-1].ID, nil
}

func readInteractiveChoice(max int) (int, error) {
	if max <= 0 {
		return 0, fmt.Errorf("choice list is empty")
	}
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("Enter number (0..%d): ", max)
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return 0, fmt.Errorf("read choice: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			if errors.Is(err, io.EOF) {
				return 0, fmt.Errorf("empty choice")
			}
			continue
		}

		choice, convErr := strconv.Atoi(line)
		if convErr != nil {
			fmt.Printf("Invalid choice %q\n", line)
			if errors.Is(err, io.EOF) {
				return 0, fmt.Errorf("invalid choice %q", line)
			}
			continue
		}
		if choice >= 0 && choice <= max {
			return choice, nil
		}

		fmt.Printf("Choose number in range 0..%d\n", max)
		if errors.Is(err, io.EOF) {
			return 0, fmt.Errorf("invalid choice %q", line)
		}
	}
}
