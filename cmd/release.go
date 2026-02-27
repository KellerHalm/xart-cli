package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

func newReleaseCommand() *cobra.Command {
	releaseCmd := &cobra.Command{
		Use:   "release",
		Short: "Операции с релизами и плеером",
	}

	releaseCmd.AddCommand(
		newReleaseInfoCommand(),
		newReleaseRelatedCommand(),
		newReleaseEpisodesCommand(),
		newReleaseSourcesCommand(),
		newReleaseStreamCommand(),
		newReleaseLicensedCommand(),
		newReleaseVoteCommand(),
		newReleaseVoteRemoveCommand(),
		newReleaseHistoryAddCommand(),
		newReleaseEpisodeWatchCommand(),
		newReleaseCollectionsCommand(),
		newReleaseCommentsCommand(),
		newReleaseCommentRepliesCommand(),
		newReleaseCommentAddCommand(),
		newReleaseCommentEditCommand(),
		newReleaseCommentDeleteCommand(),
		newReleaseCommentVoteCommand(),
	)

	return releaseCmd
}

func newReleaseInfoCommand() *cobra.Command {
	var id int

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Информация о релизе",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 {
				return fmt.Errorf("flag --id is required")
			}
			query := url.Values{}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestGET(fmt.Sprintf("/release/%d", id), query, false, ""))
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "ID релиза")
	return cmd
}

func newReleaseRelatedCommand() *cobra.Command {
	var id int
	var page int

	cmd := &cobra.Command{
		Use:   "related",
		Short: "Похожие релизы",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 {
				return fmt.Errorf("flag --id is required")
			}
			query := url.Values{}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/related/%d/%d", id, page),
				query,
				true,
				"",
			))
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "ID релиза")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	return cmd
}

func newReleaseEpisodesCommand() *cobra.Command {
	var id int

	cmd := &cobra.Command{
		Use:   "episodes",
		Short: "Список озвучек/типов эпизодов релиза",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 {
				return fmt.Errorf("flag --id is required")
			}
			return doAndPrint(requestGET(fmt.Sprintf("/episode/%d", id), nil, false, ""))
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "ID релиза")
	return cmd
}

func newReleaseSourcesCommand() *cobra.Command {
	var id int
	var voiceoverID int

	cmd := &cobra.Command{
		Use:   "sources",
		Short: "Источники плеера для озвучки",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 || voiceoverID <= 0 {
				return fmt.Errorf("flags --id and --voiceover-id are required")
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/episode/%d/%d", id, voiceoverID),
				nil,
				false,
				"",
			))
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "ID релиза")
	cmd.Flags().IntVar(&voiceoverID, "voiceover-id", 0, "ID озвучки")
	return cmd
}

func newReleaseStreamCommand() *cobra.Command {
	var id int
	var voiceoverID int
	var sourceID int
	var useV2 bool

	cmd := &cobra.Command{
		Use:   "stream",
		Short: "Получить список эпизодов/URL потока для источника",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 || voiceoverID <= 0 || sourceID <= 0 {
				return fmt.Errorf("flags --id, --voiceover-id and --source-id are required")
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/episode/%d/%d/%d", id, voiceoverID, sourceID),
				nil,
				useV2,
				"",
			))
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "ID релиза")
	cmd.Flags().IntVar(&voiceoverID, "voiceover-id", 0, "ID озвучки")
	cmd.Flags().IntVar(&sourceID, "source-id", 0, "ID источника")
	cmd.Flags().BoolVar(&useV2, "v2", false, "Использовать API-Version: v2")
	return cmd
}

func newReleaseLicensedCommand() *cobra.Command {
	var id int

	cmd := &cobra.Command{
		Use:   "licensed",
		Short: "Лицензионные платформы релиза",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 {
				return fmt.Errorf("flag --id is required")
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/release/streaming/platform/%d", id),
				nil,
				false,
				"",
			))
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "ID релиза")
	return cmd
}

func newReleaseVoteCommand() *cobra.Command {
	var id int
	var score int

	cmd := &cobra.Command{
		Use:   "vote",
		Short: "Поставить оценку релизу (1..5)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 || score < 1 || score > 5 {
				return fmt.Errorf("flags --id and --score(1..5) are required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/release/vote/add/%d/%d", id, score),
				nil,
				false,
				token,
			))
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "ID релиза")
	cmd.Flags().IntVar(&score, "score", 0, "Оценка 1..5")
	return cmd
}

func newReleaseVoteRemoveCommand() *cobra.Command {
	var id int
	cmd := &cobra.Command{
		Use:   "vote-remove",
		Short: "Удалить свою оценку релиза",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 {
				return fmt.Errorf("flag --id is required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/release/vote/delete/%d", id),
				nil,
				false,
				token,
			))
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "ID релиза")
	return cmd
}

func newReleaseHistoryAddCommand() *cobra.Command {
	var id int
	var sourceID int
	var episode int

	cmd := &cobra.Command{
		Use:   "history-add",
		Short: "Добавить эпизод в историю просмотра",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 || sourceID <= 0 || episode < 0 {
				return fmt.Errorf("flags --id, --source-id and --episode are required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/history/add/%d/%d/%d", id, sourceID, episode),
				nil,
				false,
				token,
			))
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "ID релиза")
	cmd.Flags().IntVar(&sourceID, "source-id", 0, "ID источника")
	cmd.Flags().IntVar(&episode, "episode", -1, "Позиция эпизода")
	return cmd
}

func newReleaseEpisodeWatchCommand() *cobra.Command {
	var id int
	var sourceID int
	var episode int

	cmd := &cobra.Command{
		Use:   "episode-watch",
		Short: "Пометить эпизод просмотренным",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 || sourceID <= 0 || episode < 0 {
				return fmt.Errorf("flags --id, --source-id and --episode are required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/episode/watch/%d/%d/%d", id, sourceID, episode),
				nil,
				false,
				token,
			))
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "ID релиза")
	cmd.Flags().IntVar(&sourceID, "source-id", 0, "ID источника")
	cmd.Flags().IntVar(&episode, "episode", -1, "Позиция эпизода")
	return cmd
}

func newReleaseCollectionsCommand() *cobra.Command {
	var id int
	var page int
	var sort int

	cmd := &cobra.Command{
		Use:   "collections",
		Short: "Коллекции, содержащие релиз",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 {
				return fmt.Errorf("flag --id is required")
			}
			query := url.Values{
				"sort": []string{fmt.Sprintf("%d", sort)},
			}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/collection/all/release/%d/%d", id, page),
				query,
				false,
				"",
			))
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "ID релиза")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	cmd.Flags().IntVar(&sort, "sort", 1, "Сортировка 1..6")
	return cmd
}

func newReleaseCommentsCommand() *cobra.Command {
	var id int
	var page int
	var sort int

	cmd := &cobra.Command{
		Use:   "comments",
		Short: "Комментарии релиза",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 {
				return fmt.Errorf("flag --id is required")
			}
			query := url.Values{
				"sort": []string{fmt.Sprintf("%d", sort)},
			}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/release/comment/all/%d/%d", id, page),
				query,
				false,
				"",
			))
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "ID релиза")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	cmd.Flags().IntVar(&sort, "sort", 1, "Сортировка: 1-новые, 2-старые, 3-популярные")
	return cmd
}

func newReleaseCommentRepliesCommand() *cobra.Command {
	var commentID int
	var page int
	var sort int

	cmd := &cobra.Command{
		Use:   "comment-replies",
		Short: "Ответы на комментарий",
		RunE: func(cmd *cobra.Command, args []string) error {
			if commentID <= 0 {
				return fmt.Errorf("flag --comment-id is required")
			}
			query := url.Values{
				"sort": []string{fmt.Sprintf("%d", sort)},
			}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/release/comment/replies/%d/%d", commentID, page),
				query,
				false,
				"",
			))
		},
	}
	cmd.Flags().IntVar(&commentID, "comment-id", 0, "ID комментария")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	cmd.Flags().IntVar(&sort, "sort", 2, "Сортировка")
	return cmd
}

func newReleaseCommentAddCommand() *cobra.Command {
	var id int
	var message string
	var parentCommentID int
	var replyProfileID int
	var spoiler bool

	cmd := &cobra.Command{
		Use:   "comment-add",
		Short: "Добавить комментарий к релизу",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 || strings.TrimSpace(message) == "" {
				return fmt.Errorf("flags --id and --message are required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			body := map[string]any{
				"message":          strings.TrimSpace(message),
				"parentCommentId":  nil,
				"replyToProfileId": nil,
				"spoiler":          spoiler,
			}
			if parentCommentID > 0 {
				body["parentCommentId"] = parentCommentID
			}
			if replyProfileID > 0 {
				body["replyToProfileId"] = replyProfileID
			}
			return doAndPrint(requestPOST(
				fmt.Sprintf("/release/comment/add/%d", id),
				nil,
				body,
				false,
				token,
			))
		},
	}

	cmd.Flags().IntVar(&id, "id", 0, "ID релиза")
	cmd.Flags().StringVar(&message, "message", "", "Текст комментария")
	cmd.Flags().IntVar(&parentCommentID, "parent-comment-id", 0, "ID родительского комментария")
	cmd.Flags().IntVar(&replyProfileID, "reply-profile-id", 0, "ID профиля, которому отвечаем")
	cmd.Flags().BoolVar(&spoiler, "spoiler", false, "Пометить комментарий как спойлер")
	return cmd
}

func newReleaseCommentEditCommand() *cobra.Command {
	var commentID int
	var message string
	var spoiler bool

	cmd := &cobra.Command{
		Use:   "comment-edit",
		Short: "Редактировать комментарий релиза",
		RunE: func(cmd *cobra.Command, args []string) error {
			if commentID <= 0 || strings.TrimSpace(message) == "" {
				return fmt.Errorf("flags --comment-id and --message are required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			body := map[string]any{
				"message": strings.TrimSpace(message),
				"spoiler": spoiler,
			}
			return doAndPrint(requestPOST(
				fmt.Sprintf("/release/comment/edit/%d", commentID),
				nil,
				body,
				false,
				token,
			))
		},
	}

	cmd.Flags().IntVar(&commentID, "comment-id", 0, "ID комментария")
	cmd.Flags().StringVar(&message, "message", "", "Новый текст комментария")
	cmd.Flags().BoolVar(&spoiler, "spoiler", false, "Спойлер")
	return cmd
}

func newReleaseCommentDeleteCommand() *cobra.Command {
	var commentID int

	cmd := &cobra.Command{
		Use:   "comment-delete",
		Short: "Удалить комментарий релиза",
		RunE: func(cmd *cobra.Command, args []string) error {
			if commentID <= 0 {
				return fmt.Errorf("flag --comment-id is required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/release/comment/delete/%d", commentID),
				nil,
				false,
				token,
			))
		},
	}

	cmd.Flags().IntVar(&commentID, "comment-id", 0, "ID комментария")
	return cmd
}

func newReleaseCommentVoteCommand() *cobra.Command {
	var commentID int
	var vote int

	cmd := &cobra.Command{
		Use:   "comment-vote",
		Short: "Голос за комментарий: 1-dislike, 2-like",
		RunE: func(cmd *cobra.Command, args []string) error {
			if commentID <= 0 || (vote != 1 && vote != 2) {
				return fmt.Errorf("flags --comment-id and --vote(1|2) are required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/release/comment/vote/%d/%d", commentID, vote),
				nil,
				false,
				token,
			))
		},
	}

	cmd.Flags().IntVar(&commentID, "comment-id", 0, "ID комментария")
	cmd.Flags().IntVar(&vote, "vote", 0, "Тип голоса: 1/2")
	return cmd
}
