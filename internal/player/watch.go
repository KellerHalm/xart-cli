package player

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"xart-cli/internal/xart"
)

var knownPlayers = []string{"mpv", "vlc", "ffplay"}

type Voiceover struct {
	ID   int
	Name string
}

type Source struct {
	ID   int
	Name string
}

type Episode struct {
	Position int
	Name     string
	URL      string
}

type Selection struct {
	ReleaseID int
	Voiceover Voiceover
	Source    Source
	Episode   Episode
}

type LaunchOptions struct {
	Player    string
	ExtraArgs []string
}

type LaunchPlan struct {
	Executable string
	Args       []string
	PlayerName string
	URL        string
}

func ResolveSelection(ctx context.Context, client *xart.Client, releaseID int, token string, voiceoverID, sourceID, episodePosition int) (Selection, error) {
	if releaseID <= 0 {
		return Selection{}, fmt.Errorf("release id must be > 0")
	}

	voiceovers, err := fetchVoiceovers(ctx, client, releaseID)
	if err != nil {
		return Selection{}, err
	}
	voiceover, err := pickVoiceover(voiceovers, voiceoverID)
	if err != nil {
		return Selection{}, err
	}

	sources, err := fetchSources(ctx, client, releaseID, voiceover.ID)
	if err != nil {
		return Selection{}, err
	}
	source, err := pickSource(sources, sourceID)
	if err != nil {
		return Selection{}, err
	}

	episodes, err := fetchEpisodes(ctx, client, releaseID, voiceover.ID, source.ID, token)
	if err != nil {
		return Selection{}, err
	}
	episode, err := pickEpisode(episodes, episodePosition)
	if err != nil {
		return Selection{}, err
	}

	return Selection{
		ReleaseID: releaseID,
		Voiceover: voiceover,
		Source:    source,
		Episode:   episode,
	}, nil
}

func BuildLaunchPlan(streamURL string, opts LaunchOptions) (LaunchPlan, error) {
	streamURL = strings.TrimSpace(streamURL)
	if streamURL == "" {
		return LaunchPlan{}, fmt.Errorf("empty stream url")
	}

	executable := strings.TrimSpace(opts.Player)
	extraArgs := append([]string{}, opts.ExtraArgs...)
	playerName := ""

	if executable == "" {
		for _, candidate := range knownPlayers {
			resolved, err := exec.LookPath(candidate)
			if err != nil {
				continue
			}
			executable = resolved
			playerName = candidate
			break
		}
		if executable == "" {
			return LaunchPlan{}, fmt.Errorf("player not found; install one of: mpv, vlc, ffplay, or pass --player")
		}
	} else {
		parts := strings.Fields(executable)
		executable = parts[0]
		if len(parts) > 1 {
			extraArgs = append(parts[1:], extraArgs...)
		}
		resolved, err := exec.LookPath(executable)
		if err != nil {
			return LaunchPlan{}, fmt.Errorf("player %q not found in PATH", executable)
		}
		executable = resolved
	}

	if playerName == "" {
		playerName = strings.ToLower(filepath.Base(executable))
	}
	normalized := strings.TrimSuffix(playerName, filepath.Ext(playerName))

	args := make([]string, 0, 8+len(extraArgs))
	switch normalized {
	case "mpv":
		args = append(args, "--force-window=yes", "--ytdl=yes")
	case "vlc":
		args = append(args, "--play-and-exit")
	case "ffplay":
		args = append(args, "-loglevel", "warning", "-autoexit")
	}
	args = append(args, extraArgs...)
	args = append(args, streamURL)

	return LaunchPlan{
		Executable: executable,
		Args:       args,
		PlayerName: normalized,
		URL:        streamURL,
	}, nil
}

func DetectAvailablePlayers() []string {
	available := make([]string, 0, len(knownPlayers))
	for _, candidate := range knownPlayers {
		if _, err := exec.LookPath(candidate); err == nil {
			available = append(available, candidate)
		}
	}
	return available
}

func (p LaunchPlan) Command() *exec.Cmd {
	return exec.Command(p.Executable, p.Args...)
}

func MarkEpisodeProgress(ctx context.Context, client *xart.Client, token string, releaseID, sourceID, episodePosition int) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}

	var failures []string
	_, err := client.Do(ctx, xart.Request{
		Method: "GET",
		Path:   fmt.Sprintf("/history/add/%d/%d/%d", releaseID, sourceID, episodePosition),
		Token:  token,
	})
	if err != nil {
		failures = append(failures, "history/add: "+err.Error())
	}

	_, err = client.Do(ctx, xart.Request{
		Method: "GET",
		Path:   fmt.Sprintf("/episode/watch/%d/%d/%d", releaseID, sourceID, episodePosition),
		Token:  token,
	})
	if err != nil {
		failures = append(failures, "episode/watch: "+err.Error())
	}

	if len(failures) > 0 {
		return errors.New(strings.Join(failures, "; "))
	}
	return nil
}

func fetchVoiceovers(ctx context.Context, client *xart.Client, releaseID int) ([]Voiceover, error) {
	payload, err := client.Do(ctx, xart.Request{
		Method: "GET",
		Path:   fmt.Sprintf("/episode/%d", releaseID),
	})
	if err != nil {
		return nil, err
	}
	root, ok := payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid voiceovers payload: %T", payload)
	}

	rawTypes, _ := root["types"].([]any)
	out := make([]Voiceover, 0, len(rawTypes))
	for _, raw := range rawTypes {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id := intFromAny(item["id"])
		if id <= 0 {
			continue
		}
		out = append(out, Voiceover{
			ID:   id,
			Name: stringFromAny(item["name"]),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("voiceovers not found for release %d", releaseID)
	}
	return out, nil
}

func fetchSources(ctx context.Context, client *xart.Client, releaseID, voiceoverID int) ([]Source, error) {
	payload, err := client.Do(ctx, xart.Request{
		Method: "GET",
		Path:   fmt.Sprintf("/episode/%d/%d", releaseID, voiceoverID),
	})
	if err != nil {
		return nil, err
	}
	root, ok := payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid sources payload: %T", payload)
	}

	rawSources, _ := root["sources"].([]any)
	out := make([]Source, 0, len(rawSources))
	for _, raw := range rawSources {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id := intFromAny(item["id"])
		if id <= 0 {
			continue
		}
		out = append(out, Source{
			ID:   id,
			Name: stringFromAny(item["name"]),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("sources not found for release %d and voiceover %d", releaseID, voiceoverID)
	}
	return out, nil
}

func fetchEpisodes(ctx context.Context, client *xart.Client, releaseID, voiceoverID, sourceID int, token string) ([]Episode, error) {
	payload, err := client.Do(ctx, xart.Request{
		Method: "GET",
		Path:   fmt.Sprintf("/episode/%d/%d/%d", releaseID, voiceoverID, sourceID),
		Token:  token,
	})
	if err != nil {
		return nil, err
	}
	root, ok := payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid episodes payload: %T", payload)
	}

	rawEpisodes, _ := root["episodes"].([]any)
	out := make([]Episode, 0, len(rawEpisodes))
	for _, raw := range rawEpisodes {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ep := Episode{
			Position: intFromAny(item["position"]),
			Name:     stringFromAny(item["name"]),
			URL:      strings.TrimSpace(stringFromAny(item["url"])),
		}
		if ep.Position <= 0 || ep.URL == "" {
			continue
		}
		out = append(out, ep)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("episodes not found for release %d / voiceover %d / source %d", releaseID, voiceoverID, sourceID)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Position < out[j].Position
	})
	return out, nil
}

func pickVoiceover(voiceovers []Voiceover, voiceoverID int) (Voiceover, error) {
	if len(voiceovers) == 0 {
		return Voiceover{}, fmt.Errorf("voiceovers list is empty")
	}
	if voiceoverID <= 0 {
		return voiceovers[0], nil
	}
	for _, voiceover := range voiceovers {
		if voiceover.ID == voiceoverID {
			return voiceover, nil
		}
	}
	return Voiceover{}, fmt.Errorf("voiceover id %d is not available", voiceoverID)
}

func pickSource(sources []Source, sourceID int) (Source, error) {
	if len(sources) == 0 {
		return Source{}, fmt.Errorf("sources list is empty")
	}
	if sourceID <= 0 {
		return sources[0], nil
	}
	for _, source := range sources {
		if source.ID == sourceID {
			return source, nil
		}
	}
	return Source{}, fmt.Errorf("source id %d is not available", sourceID)
}

func pickEpisode(episodes []Episode, episodePosition int) (Episode, error) {
	if len(episodes) == 0 {
		return Episode{}, fmt.Errorf("episodes list is empty")
	}
	if episodePosition > 0 {
		for _, episode := range episodes {
			if episode.Position == episodePosition {
				return episode, nil
			}
		}
		return Episode{}, fmt.Errorf("episode %d is not available", episodePosition)
	}
	return episodes[len(episodes)-1], nil
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return 0
		}
		var parsed int
		_, err := fmt.Sscanf(v, "%d", &parsed)
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
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
