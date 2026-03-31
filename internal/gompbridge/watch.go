package gompbridge

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/KellerHalm/gomp"
	playermpv "github.com/KellerHalm/gomp/player/mpv"
	"github.com/KellerHalm/gomp/tui"

	"xart-cli/internal/player"
	"xart-cli/internal/xart"
)

type WatchOptions struct {
	ReleaseID       int
	Token           string
	VoiceoverID     int
	SourceID        int
	EpisodePosition int
	MarkProgress    bool
	MPVExecutable   string
	MPVArgs         []string
}

func RunWatchTUI(ctx context.Context, client *xart.Client, opts WatchOptions) error {
	if client == nil {
		return fmt.Errorf("xart client is required")
	}
	if opts.ReleaseID <= 0 {
		return fmt.Errorf("release id must be > 0")
	}

	releaseTitle := releaseName(ctx, client, opts.ReleaseID, opts.Token)

	voiceovers, err := player.ListVoiceovers(ctx, client, opts.ReleaseID)
	if err != nil {
		return err
	}
	voiceover, err := pickVoiceover(opts.VoiceoverID, releaseTitle, voiceovers)
	if err != nil {
		return err
	}

	sources, err := player.ListSources(ctx, client, opts.ReleaseID, voiceover.ID)
	if err != nil {
		return err
	}
	source, err := pickSource(opts.SourceID, releaseTitle, voiceover, sources)
	if err != nil {
		return err
	}

	episodes, err := player.ListEpisodes(ctx, client, opts.ReleaseID, voiceover.ID, source.ID, opts.Token)
	if err != nil {
		return err
	}
	episode, err := pickEpisode(opts.EpisodePosition, releaseTitle, voiceover, source, episodes)
	if err != nil {
		return err
	}

	selection, err := player.ResolveSelection(
		ctx,
		client,
		opts.ReleaseID,
		opts.Token,
		voiceover.ID,
		source.ID,
		episode.Position,
	)
	if err != nil {
		return err
	}

	if opts.MarkProgress {
		if err := player.MarkEpisodeProgress(
			ctx,
			client,
			opts.Token,
			selection.ReleaseID,
			selection.Source.ID,
			selection.Episode.Position,
		); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to mark watch progress: %v\n", err)
		}
	}

	playlist, err := buildPlaylist(ctx, releaseTitle, selection, episodes)
	if err != nil {
		return err
	}

	controller, err := playermpv.New(ctx, playermpv.Config{
		Binary:    strings.TrimSpace(opts.MPVExecutable),
		ExtraArgs: opts.MPVArgs,
	})
	if err != nil {
		return err
	}
	defer controller.Close()

	return tui.Run(ctx, tui.Options{
		Player:          controller,
		Playlist:        playlist,
		AlternateScreen: true,
	})
}

func buildPlaylist(ctx context.Context, releaseTitle string, selection player.Selection, episodes []player.Episode) (gomp.Playlist, error) {
	tracks := make([]gomp.Track, 0, len(episodes))
	startIndex := 0

	fmt.Fprintf(os.Stderr, "Resolving playlist for %s: %d episodes\n", releaseTitle, len(episodes))

	for _, episode := range episodes {
		playableURL := ""
		var err error
		switch episode.Position {
		case selection.Episode.Position:
			playableURL = selection.Episode.URL
		default:
			playableURL, err = player.ResolvePlayableURL(ctx, episode.URL)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to resolve episode %d: %v\n", episode.Position, err)
				continue
			}
		}

		title := strings.TrimSpace(episode.Name)
		if title == "" {
			title = fmt.Sprintf("Episode %d", episode.Position)
		}

		if episode.Position == selection.Episode.Position {
			startIndex = len(tracks)
		}
		tracks = append(tracks, gomp.Track{
			ID:       fmt.Sprintf("%d-%d-%d-%d", selection.ReleaseID, selection.Voiceover.ID, selection.Source.ID, episode.Position),
			Title:    title,
			Subtitle: fmt.Sprintf("%s | %s", fallbackText(selection.Voiceover.Name, fmt.Sprintf("voiceover %d", selection.Voiceover.ID)), fallbackText(selection.Source.Name, fmt.Sprintf("source %d", selection.Source.ID))),
			Location: playableURL,
			Tags:     []string{"xart-cli", "anixart", "gomp"},
			Meta: map[string]string{
				"release_id":       strconv.Itoa(selection.ReleaseID),
				"voiceover_id":     strconv.Itoa(selection.Voiceover.ID),
				"voiceover_name":   fallbackText(selection.Voiceover.Name, "-"),
				"source_id":        strconv.Itoa(selection.Source.ID),
				"source_name":      fallbackText(selection.Source.Name, "-"),
				"episode_position": strconv.Itoa(episode.Position),
			},
		})
	}

	if len(tracks) == 0 {
		return gomp.Playlist{}, fmt.Errorf("no playable episodes were resolved for release %d", selection.ReleaseID)
	}

	return gomp.Playlist{
		Title:       releaseTitle,
		Description: fmt.Sprintf("xart-cli watch via gomp TUI (release %d)", selection.ReleaseID),
		StartIndex:  startIndex,
		Tracks:      tracks,
	}, nil
}

func pickVoiceover(selectedID int, releaseTitle string, voiceovers []player.Voiceover) (player.Voiceover, error) {
	if len(voiceovers) == 0 {
		return player.Voiceover{}, fmt.Errorf("voiceovers not found")
	}
	if selectedID > 0 {
		for _, voiceover := range voiceovers {
			if voiceover.ID == selectedID {
				return voiceover, nil
			}
		}
		return player.Voiceover{}, fmt.Errorf("voiceover id %d is not available", selectedID)
	}

	options := make([]string, 0, len(voiceovers))
	for _, voiceover := range voiceovers {
		options = append(options, fmt.Sprintf("%s (id=%d)", fallbackText(voiceover.Name, "untitled"), voiceover.ID))
	}
	index, err := promptChoice(
		fmt.Sprintf("Choose voiceover for %s", releaseTitle),
		options,
		0,
	)
	if err != nil {
		return player.Voiceover{}, err
	}
	return voiceovers[index], nil
}

func pickSource(selectedID int, releaseTitle string, voiceover player.Voiceover, sources []player.Source) (player.Source, error) {
	if len(sources) == 0 {
		return player.Source{}, fmt.Errorf("sources not found")
	}
	if selectedID > 0 {
		for _, source := range sources {
			if source.ID == selectedID {
				return source, nil
			}
		}
		return player.Source{}, fmt.Errorf("source id %d is not available", selectedID)
	}

	options := make([]string, 0, len(sources))
	for _, source := range sources {
		options = append(options, fmt.Sprintf("%s (id=%d)", fallbackText(source.Name, "untitled"), source.ID))
	}
	index, err := promptChoice(
		fmt.Sprintf("Choose source for %s / %s", releaseTitle, fallbackText(voiceover.Name, "voiceover")),
		options,
		0,
	)
	if err != nil {
		return player.Source{}, err
	}
	return sources[index], nil
}

func pickEpisode(selectedPosition int, releaseTitle string, voiceover player.Voiceover, source player.Source, episodes []player.Episode) (player.Episode, error) {
	if len(episodes) == 0 {
		return player.Episode{}, fmt.Errorf("episodes not found")
	}
	if selectedPosition > 0 {
		for _, episode := range episodes {
			if episode.Position == selectedPosition {
				return episode, nil
			}
		}
		return player.Episode{}, fmt.Errorf("episode %d is not available", selectedPosition)
	}

	options := make([]string, 0, len(episodes))
	for _, episode := range episodes {
		title := strings.TrimSpace(episode.Name)
		if title == "" {
			title = fmt.Sprintf("Episode %d", episode.Position)
		}
		options = append(options, fmt.Sprintf("%s (episode %d)", title, episode.Position))
	}
	index, err := promptChoice(
		fmt.Sprintf("Choose episode for %s / %s / %s", releaseTitle, fallbackText(voiceover.Name, "voiceover"), fallbackText(source.Name, "source")),
		options,
		len(episodes)-1,
	)
	if err != nil {
		return player.Episode{}, err
	}
	return episodes[index], nil
}

func promptChoice(label string, options []string, defaultIndex int) (int, error) {
	if len(options) == 0 {
		return 0, fmt.Errorf("choice list is empty")
	}
	if defaultIndex < 0 || defaultIndex >= len(options) {
		defaultIndex = 0
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s:\n", label)
	for index, option := range options {
		fmt.Printf("  %d) %s\n", index+1, option)
	}

	for {
		fmt.Printf("Enter number [default %d]: ", defaultIndex+1)
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return 0, fmt.Errorf("read choice: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			return defaultIndex, nil
		}

		choice, convErr := strconv.Atoi(line)
		if convErr != nil {
			fmt.Printf("Invalid choice %q\n", line)
			if errors.Is(err, io.EOF) {
				return 0, fmt.Errorf("invalid choice %q", line)
			}
			continue
		}
		if choice >= 1 && choice <= len(options) {
			return choice - 1, nil
		}

		fmt.Printf("Choose number in range 1..%d\n", len(options))
		if errors.Is(err, io.EOF) {
			return 0, fmt.Errorf("invalid choice %q", line)
		}
	}
}

func releaseName(ctx context.Context, client *xart.Client, releaseID int, token string) string {
	payload, err := client.Do(ctx, xart.Request{
		Method: "GET",
		Path:   fmt.Sprintf("/release/%d", releaseID),
		Token:  token,
	})
	if err != nil {
		return fmt.Sprintf("Release %d", releaseID)
	}

	root, ok := payload.(map[string]any)
	if !ok {
		return fmt.Sprintf("Release %d", releaseID)
	}

	release, _ := root["release"].(map[string]any)
	if release == nil {
		release = root
	}

	return firstNonEmpty(
		stringFromAny(release["title_ru"]),
		stringFromAny(release["title_original"]),
		fmt.Sprintf("Release %d", releaseID),
	)
}

func stringFromAny(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func fallbackText(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
