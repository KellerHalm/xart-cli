package player

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"xart-cli/internal/xart"
)

var (
	knownPlayers      = []string{"mpv", "vlc", "ffplay"}
	playerHTTPClient  = &http.Client{Timeout: 20 * time.Second}
	kodikBase64Shift  = 18
	kodikVarTemplates = map[string]*regexp.Regexp{
		"type":     regexp.MustCompile(`var\s+type\s*=\s*"([^"]*)";`),
		"videoId":  regexp.MustCompile(`var\s+videoId\s*=\s*"([^"]*)";`),
		"domain":   regexp.MustCompile(`var\s+domain\s*=\s*"([^"]*)";`),
		"d_sign":   regexp.MustCompile(`var\s+d_sign\s*=\s*"([^"]*)";`),
		"pd":       regexp.MustCompile(`var\s+pd\s*=\s*"([^"]*)";`),
		"pd_sign":  regexp.MustCompile(`var\s+pd_sign\s*=\s*"([^"]*)";`),
		"ref":      regexp.MustCompile(`var\s+ref\s*=\s*"([^"]*)";`),
		"ref_sign": regexp.MustCompile(`var\s+ref_sign\s*=\s*"([^"]*)";`),
	}
)

type kodikSource struct {
	Src string `json:"src"`
}

type kodikFtorResponse struct {
	Link    string                   `json:"link"`
	Default int                      `json:"default"`
	Links   map[string][]kodikSource `json:"links"`
}

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
	episode.URL, err = resolvePlayableURL(ctx, episode.URL)
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
		args = append(args, "--force-window=yes")
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

func resolvePlayableURL(ctx context.Context, streamURL string) (string, error) {
	streamURL = strings.TrimSpace(streamURL)
	if streamURL == "" {
		return "", fmt.Errorf("empty stream url")
	}

	parsed, err := url.Parse(streamURL)
	if err != nil {
		return "", fmt.Errorf("parse stream url: %w", err)
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return streamURL, nil
	}
	if !strings.Contains(host, "kodik.") {
		return streamURL, nil
	}

	path := strings.ToLower(parsed.Path)
	if strings.Contains(path, ".m3u8") || strings.Contains(path, ".mp4") {
		return normalizeStreamURL(streamURL), nil
	}

	resolved, err := resolveKodikStreamURL(ctx, parsed)
	if err != nil {
		return "", fmt.Errorf("resolve kodik stream: %w", err)
	}
	return normalizeStreamURL(resolved), nil
}

func resolveKodikStreamURL(ctx context.Context, parsed *url.URL) (string, error) {
	pageReq, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", fmt.Errorf("build kodik page request: %w", err)
	}
	pageResp, err := playerHTTPClient.Do(pageReq)
	if err != nil {
		return "", fmt.Errorf("request kodik page: %w", err)
	}
	defer pageResp.Body.Close()
	if pageResp.StatusCode < 200 || pageResp.StatusCode >= 300 {
		return "", fmt.Errorf("kodik page status %d", pageResp.StatusCode)
	}

	body, err := io.ReadAll(pageResp.Body)
	if err != nil {
		return "", fmt.Errorf("read kodik page: %w", err)
	}
	html := string(body)

	vars := map[string]string{}
	for key, re := range kodikVarTemplates {
		matches := re.FindStringSubmatch(html)
		if len(matches) < 2 {
			return "", fmt.Errorf("missing %s in kodik page", key)
		}
		vars[key] = matches[1]
	}

	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(segments) < 3 {
		return "", fmt.Errorf("unexpected kodik path %q", parsed.Path)
	}
	hash := segments[2]
	requestedQuality := 0
	if len(segments) >= 4 {
		requestedQuality = parseQuality(strings.TrimSpace(segments[3]))
	}

	form := url.Values{}
	form.Set("d", vars["domain"])
	form.Set("d_sign", vars["d_sign"])
	form.Set("pd", vars["pd"])
	form.Set("pd_sign", vars["pd_sign"])
	form.Set("ref", vars["ref"])
	form.Set("ref_sign", vars["ref_sign"])
	form.Set("bad_user", "false")
	form.Set("type", vars["type"])
	form.Set("id", vars["videoId"])
	form.Set("hash", hash)

	ftorURL := fmt.Sprintf("%s://%s/ftor", parsed.Scheme, parsed.Host)
	ftorReq, err := http.NewRequestWithContext(ctx, http.MethodPost, ftorURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build kodik ftor request: %w", err)
	}
	ftorReq.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	ftorResp, err := playerHTTPClient.Do(ftorReq)
	if err != nil {
		return "", fmt.Errorf("request kodik ftor: %w", err)
	}
	defer ftorResp.Body.Close()
	if ftorResp.StatusCode < 200 || ftorResp.StatusCode >= 300 {
		return "", fmt.Errorf("kodik ftor status %d", ftorResp.StatusCode)
	}

	var payload kodikFtorResponse
	if err := json.NewDecoder(ftorResp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode kodik ftor: %w", err)
	}

	if link := strings.TrimSpace(payload.Link); link != "" {
		return link, nil
	}

	sourceURL, err := pickKodikSource(payload, requestedQuality)
	if err != nil {
		return "", err
	}
	return sourceURL, nil
}

func pickKodikSource(payload kodikFtorResponse, requestedQuality int) (string, error) {
	if len(payload.Links) == 0 {
		return "", fmt.Errorf("kodik response does not contain stream links")
	}

	qualityKeys := make([]int, 0, len(payload.Links))
	for key := range payload.Links {
		value, err := strconv.Atoi(key)
		if err != nil {
			continue
		}
		qualityKeys = append(qualityKeys, value)
	}
	sort.Ints(qualityKeys)

	candidates := make([]int, 0, 4)
	if requestedQuality > 0 {
		candidates = append(candidates, requestedQuality)
	}
	if payload.Default > 0 {
		candidates = append(candidates, payload.Default)
	}
	for i := len(qualityKeys) - 1; i >= 0; i-- {
		candidates = append(candidates, qualityKeys[i])
	}

	seen := map[int]struct{}{}
	for _, quality := range candidates {
		if _, ok := seen[quality]; ok {
			continue
		}
		seen[quality] = struct{}{}

		sources := payload.Links[strconv.Itoa(quality)]
		for _, source := range sources {
			decoded, err := decodeKodikSource(source.Src)
			if err != nil {
				continue
			}
			if strings.TrimSpace(decoded) != "" {
				return decoded, nil
			}
		}
	}

	return "", fmt.Errorf("no playable stream in kodik links")
}

func decodeKodikSource(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty source url")
	}
	if strings.Contains(raw, "://") || strings.HasPrefix(raw, "//") {
		return raw, nil
	}

	rotated := make([]rune, 0, len(raw))
	for _, ch := range raw {
		switch {
		case ch >= 'A' && ch <= 'Z':
			next := ch + rune(kodikBase64Shift)
			if next > 'Z' {
				next -= 26
			}
			rotated = append(rotated, next)
		case ch >= 'a' && ch <= 'z':
			next := ch + rune(kodikBase64Shift)
			if next > 'z' {
				next -= 26
			}
			rotated = append(rotated, next)
		default:
			rotated = append(rotated, ch)
		}
	}

	decoded, err := base64.StdEncoding.DecodeString(string(rotated))
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(string(rotated))
		if err != nil {
			return "", fmt.Errorf("decode base64 source: %w", err)
		}
	}
	return string(decoded), nil
}

func parseQuality(value string) int {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimSuffix(value, "p")
	if value == "" {
		return 0
	}
	quality, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return quality
}

func normalizeStreamURL(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "//") {
		return "https:" + value
	}
	return value
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
