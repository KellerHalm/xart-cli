package cmd

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"xart-cli/internal/xart"
)

func newCollectionsCommand() *cobra.Command {
	collectionsCmd := &cobra.Command{
		Use:   "collections",
		Short: "Коллекции и комментарии коллекций",
	}

	collectionsCmd.AddCommand(
		newCollectionsInfoCommand(),
		newCollectionsReleasesCommand(),
		newCollectionsProfileCommand(),
		newCollectionsFavoritesCommand(),
		newCollectionsAllCommand(),
		newCollectionsOfReleaseCommand(),
		newCollectionsCreateCommand(),
		newCollectionsEditCommand(),
		newCollectionsDeleteCommand(),
		newCollectionsFavoriteAddCommand(),
		newCollectionsFavoriteRemoveCommand(),
		newCollectionsAddReleaseCommand(),
		newCollectionsEditImageCommand(),
		newCollectionsCommentsCommand(),
		newCollectionsCommentRepliesCommand(),
		newCollectionsCommentAddCommand(),
		newCollectionsCommentEditCommand(),
		newCollectionsCommentDeleteCommand(),
		newCollectionsCommentVoteCommand(),
	)

	return collectionsCmd
}

func newCollectionsInfoCommand() *cobra.Command {
	var id int

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Информация о коллекции",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 {
				return fmt.Errorf("flag --id is required")
			}
			query := url.Values{}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestGET(fmt.Sprintf("/collection/%d", id), query, false, ""))
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "ID коллекции")
	return cmd
}

func newCollectionsReleasesCommand() *cobra.Command {
	var id int
	var page int
	cmd := &cobra.Command{
		Use:   "releases",
		Short: "Релизы внутри коллекции",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 {
				return fmt.Errorf("flag --id is required")
			}
			query := url.Values{}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/collection/%d/releases/%d", id, page),
				query,
				false,
				"",
			))
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "ID коллекции")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	return cmd
}

func newCollectionsProfileCommand() *cobra.Command {
	var profileID int
	var page int

	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Коллекции профиля",
		RunE: func(cmd *cobra.Command, args []string) error {
			if profileID <= 0 {
				return fmt.Errorf("flag --profile-id is required")
			}
			query := url.Values{}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/collection/all/profile/%d/%d", profileID, page),
				query,
				false,
				"",
			))
		},
	}

	cmd.Flags().IntVar(&profileID, "profile-id", 0, "ID профиля")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	return cmd
}

func newCollectionsFavoritesCommand() *cobra.Command {
	var page int
	cmd := &cobra.Command{
		Use:   "favorites",
		Short: "Избранные коллекции пользователя",
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/collectionFavorite/all/%d", page),
				query,
				false,
				"",
			))
		},
	}
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	return cmd
}

func newCollectionsAllCommand() *cobra.Command {
	var page int
	var where int
	var sortValue int
	var previousPage int

	cmd := &cobra.Command{
		Use:   "all",
		Short: "Общий каталог коллекций",
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{
				"where":         []string{strconv.Itoa(where)},
				"sort":          []string{strconv.Itoa(sortValue)},
				"previous_page": []string{strconv.Itoa(previousPage)},
			}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/collection/all/%d", page),
				query,
				false,
				"",
			))
		},
	}
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	cmd.Flags().IntVar(&where, "where", 1, "where")
	cmd.Flags().IntVar(&sortValue, "sort", 4, "sort")
	cmd.Flags().IntVar(&previousPage, "previous-page", 0, "previous_page")
	return cmd
}

func newCollectionsOfReleaseCommand() *cobra.Command {
	var releaseID int
	var page int
	var sortValue int

	cmd := &cobra.Command{
		Use:   "of-release",
		Short: "Коллекции, содержащие релиз",
		RunE: func(cmd *cobra.Command, args []string) error {
			if releaseID <= 0 {
				return fmt.Errorf("flag --release-id is required")
			}
			query := url.Values{
				"sort": []string{strconv.Itoa(sortValue)},
			}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/collection/all/release/%d/%d", releaseID, page),
				query,
				false,
				"",
			))
		},
	}
	cmd.Flags().IntVar(&releaseID, "release-id", 0, "ID релиза")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	cmd.Flags().IntVar(&sortValue, "sort", 1, "Сортировка")
	return cmd
}

func newCollectionsCreateCommand() *cobra.Command {
	var title string
	var description string
	var isPrivate bool
	var releases string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Создать коллекцию",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(strings.TrimSpace(title)) < 1 {
				return fmt.Errorf("flag --title is required")
			}
			releaseIDs, err := parseCommaIntList(releases)
			if err != nil {
				return err
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}

			body := map[string]any{
				"title":       strings.TrimSpace(title),
				"description": strings.TrimSpace(description),
				"is_private":  isPrivate,
				"private":     isPrivate,
				"releases":    releaseIDs,
			}
			return doAndPrint(requestPOST("/collectionMy/create", nil, body, false, token))
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Название коллекции")
	cmd.Flags().StringVar(&description, "description", "", "Описание")
	cmd.Flags().BoolVar(&isPrivate, "private", false, "Приватная коллекция")
	cmd.Flags().StringVar(&releases, "releases", "", "ID релизов через запятую: 1,2,3")
	return cmd
}

func newCollectionsEditCommand() *cobra.Command {
	var id int
	var title string
	var description string
	var isPrivate bool
	var releases string

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Редактировать коллекцию",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 {
				return fmt.Errorf("flag --id is required")
			}
			if len(strings.TrimSpace(title)) < 1 {
				return fmt.Errorf("flag --title is required")
			}
			releaseIDs, err := parseCommaIntList(releases)
			if err != nil {
				return err
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}

			body := map[string]any{
				"title":       strings.TrimSpace(title),
				"description": strings.TrimSpace(description),
				"is_private":  isPrivate,
				"private":     isPrivate,
				"releases":    releaseIDs,
			}
			return doAndPrint(requestPOST(fmt.Sprintf("/collectionMy/edit/%d", id), nil, body, false, token))
		},
	}

	cmd.Flags().IntVar(&id, "id", 0, "ID коллекции")
	cmd.Flags().StringVar(&title, "title", "", "Название")
	cmd.Flags().StringVar(&description, "description", "", "Описание")
	cmd.Flags().BoolVar(&isPrivate, "private", false, "Приватная")
	cmd.Flags().StringVar(&releases, "releases", "", "ID релизов через запятую")
	return cmd
}

func newCollectionsDeleteCommand() *cobra.Command {
	var id int
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Удалить свою коллекцию",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 {
				return fmt.Errorf("flag --id is required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET(fmt.Sprintf("/collectionMy/delete/%d", id), nil, false, token))
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "ID коллекции")
	return cmd
}

func newCollectionsFavoriteAddCommand() *cobra.Command {
	var id int
	cmd := &cobra.Command{
		Use:   "favorite-add",
		Short: "Добавить коллекцию в избранное",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 {
				return fmt.Errorf("flag --id is required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET(fmt.Sprintf("/collectionFavorite/add/%d", id), nil, false, token))
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "ID коллекции")
	return cmd
}

func newCollectionsFavoriteRemoveCommand() *cobra.Command {
	var id int
	cmd := &cobra.Command{
		Use:   "favorite-remove",
		Short: "Удалить коллекцию из избранного",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 {
				return fmt.Errorf("flag --id is required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET(fmt.Sprintf("/collectionFavorite/delete/%d", id), nil, false, token))
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "ID коллекции")
	return cmd
}

func newCollectionsAddReleaseCommand() *cobra.Command {
	var collectionID int
	var releaseID int
	cmd := &cobra.Command{
		Use:   "add-release",
		Short: "Добавить релиз в свою коллекцию",
		RunE: func(cmd *cobra.Command, args []string) error {
			if collectionID <= 0 || releaseID <= 0 {
				return fmt.Errorf("flags --collection-id and --release-id are required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			query := url.Values{
				"release_id": []string{strconv.Itoa(releaseID)},
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/collectionMy/release/add/%d", collectionID),
				query,
				false,
				token,
			))
		},
	}
	cmd.Flags().IntVar(&collectionID, "collection-id", 0, "ID коллекции")
	cmd.Flags().IntVar(&releaseID, "release-id", 0, "ID релиза")
	return cmd
}

func newCollectionsEditImageCommand() *cobra.Command {
	var id int
	var filePath string

	cmd := &cobra.Command{
		Use:   "edit-image",
		Short: "Обновить обложку коллекции",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 || filePath == "" {
				return fmt.Errorf("flags --id and --file are required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			fileData, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("read image: %w", err)
			}

			var buffer bytes.Buffer
			writer := multipart.NewWriter(&buffer)
			part, err := writer.CreateFormFile("image", "image.jpg")
			if err != nil {
				return err
			}
			if _, err := part.Write(fileData); err != nil {
				return err
			}
			if err := writer.WriteField("name", "image"); err != nil {
				return err
			}
			if err := writer.Close(); err != nil {
				return err
			}

			payload, err := rt.client.Do(withContext(), xart.Request{
				Method:      "POST",
				Path:        fmt.Sprintf("/collectionMy/editImage/%d", id),
				Body:        buffer.Bytes(),
				ContentType: writer.FormDataContentType(),
				Token:       token,
			})
			if err != nil {
				return err
			}
			return printPayload(payload)
		},
	}

	cmd.Flags().IntVar(&id, "id", 0, "ID коллекции")
	cmd.Flags().StringVar(&filePath, "file", "", "Путь к изображению")
	return cmd
}

func newCollectionsCommentsCommand() *cobra.Command {
	var id int
	var page int
	var sortValue int

	cmd := &cobra.Command{
		Use:   "comments",
		Short: "Комментарии коллекции",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id <= 0 {
				return fmt.Errorf("flag --id is required")
			}
			query := url.Values{
				"sort": []string{strconv.Itoa(sortValue)},
			}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/collection/comment/all/%d/%d", id, page),
				query,
				false,
				"",
			))
		},
	}

	cmd.Flags().IntVar(&id, "id", 0, "ID коллекции")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	cmd.Flags().IntVar(&sortValue, "sort", 3, "Сортировка")
	return cmd
}

func newCollectionsCommentRepliesCommand() *cobra.Command {
	var commentID int
	var page int
	var sortValue int

	cmd := &cobra.Command{
		Use:   "comment-replies",
		Short: "Ответы на комментарий коллекции",
		RunE: func(cmd *cobra.Command, args []string) error {
			if commentID <= 0 {
				return fmt.Errorf("flag --comment-id is required")
			}
			query := url.Values{
				"sort": []string{strconv.Itoa(sortValue)},
			}
			if token := tokenOptional(); token != "" {
				query.Set("token", token)
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/collection/comment/replies/%d/%d", commentID, page),
				query,
				false,
				"",
			))
		},
	}

	cmd.Flags().IntVar(&commentID, "comment-id", 0, "ID комментария")
	cmd.Flags().IntVar(&page, "page", 0, "Страница")
	cmd.Flags().IntVar(&sortValue, "sort", 2, "Сортировка")
	return cmd
}

func newCollectionsCommentAddCommand() *cobra.Command {
	var id int
	var message string
	var parentCommentID int
	var replyProfileID int
	var spoiler bool

	cmd := &cobra.Command{
		Use:   "comment-add",
		Short: "Добавить комментарий в коллекцию",
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
				fmt.Sprintf("/collection/comment/add/%d", id),
				nil,
				body,
				false,
				token,
			))
		},
	}
	cmd.Flags().IntVar(&id, "id", 0, "ID коллекции")
	cmd.Flags().StringVar(&message, "message", "", "Текст комментария")
	cmd.Flags().IntVar(&parentCommentID, "parent-comment-id", 0, "ID родительского комментария")
	cmd.Flags().IntVar(&replyProfileID, "reply-profile-id", 0, "ID профиля, которому отвечаем")
	cmd.Flags().BoolVar(&spoiler, "spoiler", false, "Спойлер")
	return cmd
}

func newCollectionsCommentEditCommand() *cobra.Command {
	var commentID int
	var message string
	var spoiler bool

	cmd := &cobra.Command{
		Use:   "comment-edit",
		Short: "Редактировать комментарий коллекции",
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
				fmt.Sprintf("/collection/comment/edit/%d", commentID),
				nil,
				body,
				false,
				token,
			))
		},
	}
	cmd.Flags().IntVar(&commentID, "comment-id", 0, "ID комментария")
	cmd.Flags().StringVar(&message, "message", "", "Новый текст")
	cmd.Flags().BoolVar(&spoiler, "spoiler", false, "Спойлер")
	return cmd
}

func newCollectionsCommentDeleteCommand() *cobra.Command {
	var commentID int
	cmd := &cobra.Command{
		Use:   "comment-delete",
		Short: "Удалить комментарий коллекции",
		RunE: func(cmd *cobra.Command, args []string) error {
			if commentID <= 0 {
				return fmt.Errorf("flag --comment-id is required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/collection/comment/delete/%d", commentID),
				nil,
				false,
				token,
			))
		},
	}
	cmd.Flags().IntVar(&commentID, "comment-id", 0, "ID комментария")
	return cmd
}

func newCollectionsCommentVoteCommand() *cobra.Command {
	var commentID int
	var vote int
	cmd := &cobra.Command{
		Use:   "comment-vote",
		Short: "Голос за комментарий коллекции: 1-dislike, 2-like",
		RunE: func(cmd *cobra.Command, args []string) error {
			if commentID <= 0 || (vote != 1 && vote != 2) {
				return fmt.Errorf("flags --comment-id and --vote(1|2) are required")
			}
			token, err := mustTokenOrError()
			if err != nil {
				return err
			}
			return doAndPrint(requestGET(
				fmt.Sprintf("/collection/comment/vote/%d/%d", commentID, vote),
				nil,
				false,
				token,
			))
		},
	}
	cmd.Flags().IntVar(&commentID, "comment-id", 0, "ID комментария")
	cmd.Flags().IntVar(&vote, "vote", 0, "Голос 1/2")
	return cmd
}
