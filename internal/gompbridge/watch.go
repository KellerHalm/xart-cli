package gompbridge

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/KellerHalm/gomp"
	"github.com/KellerHalm/gomp/backend/mpv"
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

type WatchResolver struct {
	client       *xart.Client
	opts         WatchOptions
	releaseTitle string
}

func RunWatchTUI(ctx context.Context, client *xart.Client, opts WatchOptions) error {
	if client == nil {
		return fmt.Errorf("xart client is required")
	}
	if opts.ReleaseID <= 0 {
		return fmt.Errorf("release id must be > 0")
	}

	resolver := &WatchResolver{
		client: client,
		opts:   opts,
	}

	return tui.Run(ctx, tui.Config{
		Resolver: resolver,
		Backend: mpv.New(mpv.Config{
			Executable: opts.MPVExecutable,
			ExtraArgs:  opts.MPVArgs,
		}),
		AutoStart: true,
	})
}

func (r *WatchResolver) NextStep(ctx context.Context, values gomp.SelectionValues) (gomp.SelectionStep, error) {
	voiceoverID, ok, err := r.voiceoverID(values)
	if err != nil {
		return gomp.SelectionStep{}, err
	}
	if !ok {
		return r.voiceoverStep(ctx)
	}

	sourceID, ok, err := r.sourceID(values)
	if err != nil {
		return gomp.SelectionStep{}, err
	}
	if !ok {
		return r.sourceStep(ctx, voiceoverID)
	}

	_, ok, err = r.episodePosition(values)
	if err != nil {
		return gomp.SelectionStep{}, err
	}
	if !ok {
		return r.episodeStep(ctx, voiceoverID, sourceID)
	}

	return gomp.SelectionStep{}, gomp.ErrSelectionComplete
}

func (r *WatchResolver) Resolve(ctx context.Context, values gomp.SelectionValues) (gomp.Playlist, error) {
	voiceoverID, _, err := r.voiceoverID(values)
	if err != nil {
		return gomp.Playlist{}, err
	}
	sourceID, _, err := r.sourceID(values)
	if err != nil {
		return gomp.Playlist{}, err
	}
	episodePosition, _, err := r.episodePosition(values)
	if err != nil {
		return gomp.Playlist{}, err
	}

	selection, err := player.ResolveSelection(
		ctx,
		r.client,
		r.opts.ReleaseID,
		r.opts.Token,
		voiceoverID,
		sourceID,
		episodePosition,
	)
	if err != nil {
		return gomp.Playlist{}, err
	}

	if r.opts.MarkProgress {
		if err := player.MarkEpisodeProgress(
			ctx,
			r.client,
			r.opts.Token,
			selection.ReleaseID,
			selection.Source.ID,
			selection.Episode.Position,
		); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to mark watch progress: %v\n", err)
		}
	}

	releaseTitle := r.releaseName(ctx)
	episodes, err := player.ListEpisodes(ctx, r.client, r.opts.ReleaseID, voiceoverID, sourceID, r.opts.Token)
	if err != nil {
		return gomp.Playlist{}, err
	}

	tracks := make([]gomp.Track, 0, len(episodes))
	startIndex := 0
	for _, episode := range episodes {
		playableURL := ""
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
		return gomp.Playlist{}, fmt.Errorf("no playable episodes were resolved for release %d", r.opts.ReleaseID)
	}

	return gomp.Playlist{
		Title:       releaseTitle,
		Description: fmt.Sprintf("xart-cli watch via gomp TUI (release %d)", r.opts.ReleaseID),
		StartIndex:  startIndex,
		Tracks:      tracks,
	}, nil
}

func (r *WatchResolver) voiceoverStep(ctx context.Context) (gomp.SelectionStep, error) {
	voiceovers, err := player.ListVoiceovers(ctx, r.client, r.opts.ReleaseID)
	if err != nil {
		return gomp.SelectionStep{}, err
	}
	options := make([]gomp.SelectionOption, 0, len(voiceovers))
	for _, voiceover := range voiceovers {
		options = append(options, gomp.SelectionOption{
			ID:    strconv.Itoa(voiceover.ID),
			Title: fallbackText(voiceover.Name, fmt.Sprintf("Voiceover %d", voiceover.ID)),
		})
	}
	return gomp.SelectionStep{
		Key:         "voiceover",
		Title:       "Voiceover",
		Description: "Choose the dub/voiceover before source and episode selection.",
		Options:     options,
	}, nil
}

func (r *WatchResolver) sourceStep(ctx context.Context, voiceoverID int) (gomp.SelectionStep, error) {
	sources, err := player.ListSources(ctx, r.client, r.opts.ReleaseID, voiceoverID)
	if err != nil {
		return gomp.SelectionStep{}, err
	}
	options := make([]gomp.SelectionOption, 0, len(sources))
	for _, source := range sources {
		options = append(options, gomp.SelectionOption{
			ID:    strconv.Itoa(source.ID),
			Title: fallbackText(source.Name, fmt.Sprintf("Source %d", source.ID)),
		})
	}
	return gomp.SelectionStep{
		Key:         "source",
		Title:       "Source",
		Description: "Choose the player source for the selected voiceover.",
		Options:     options,
	}, nil
}

func (r *WatchResolver) episodeStep(ctx context.Context, voiceoverID, sourceID int) (gomp.SelectionStep, error) {
	episodes, err := player.ListEpisodes(ctx, r.client, r.opts.ReleaseID, voiceoverID, sourceID, r.opts.Token)
	if err != nil {
		return gomp.SelectionStep{}, err
	}
	options := make([]gomp.SelectionOption, 0, len(episodes))
	initialIndex := 0
	for i, episode := range episodes {
		title := strings.TrimSpace(episode.Name)
		if title == "" {
			title = fmt.Sprintf("Episode %d", episode.Position)
		}
		options = append(options, gomp.SelectionOption{
			ID:       strconv.Itoa(episode.Position),
			Title:    title,
			Subtitle: fmt.Sprintf("position %d", episode.Position),
		})
		initialIndex = i
	}
	return gomp.SelectionStep{
		Key:          "episode",
		Title:        "Episode",
		Description:  "Choose which episode to resolve and start in the gomp TUI player.",
		InitialIndex: initialIndex,
		Options:      options,
	}, nil
}

func (r *WatchResolver) voiceoverID(values gomp.SelectionValues) (int, bool, error) {
	if r.opts.VoiceoverID > 0 {
		return r.opts.VoiceoverID, true, nil
	}
	return parseOptionalSelection(values["voiceover"], "voiceover")
}

func (r *WatchResolver) sourceID(values gomp.SelectionValues) (int, bool, error) {
	if r.opts.SourceID > 0 {
		return r.opts.SourceID, true, nil
	}
	return parseOptionalSelection(values["source"], "source")
}

func (r *WatchResolver) episodePosition(values gomp.SelectionValues) (int, bool, error) {
	if r.opts.EpisodePosition > 0 {
		return r.opts.EpisodePosition, true, nil
	}
	return parseOptionalSelection(values["episode"], "episode")
}

func (r *WatchResolver) releaseName(ctx context.Context) string {
	if strings.TrimSpace(r.releaseTitle) != "" {
		return r.releaseTitle
	}

	payload, err := r.client.Do(ctx, xart.Request{
		Method: "GET",
		Path:   fmt.Sprintf("/release/%d", r.opts.ReleaseID),
		Token:  r.opts.Token,
	})
	if err != nil {
		r.releaseTitle = fmt.Sprintf("Release %d", r.opts.ReleaseID)
		return r.releaseTitle
	}

	root, ok := payload.(map[string]any)
	if !ok {
		r.releaseTitle = fmt.Sprintf("Release %d", r.opts.ReleaseID)
		return r.releaseTitle
	}

	release, _ := root["release"].(map[string]any)
	if release == nil {
		release = root
	}

	r.releaseTitle = firstNonEmpty(
		stringFromAny(release["title_ru"]),
		stringFromAny(release["title_original"]),
		fmt.Sprintf("Release %d", r.opts.ReleaseID),
	)
	return r.releaseTitle
}

func parseOptionalSelection(raw string, label string) (int, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false, fmt.Errorf("invalid %s value %q: %w", label, raw, err)
	}
	if value <= 0 {
		return 0, false, fmt.Errorf("%s must be > 0", label)
	}
	return value, true, nil
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
