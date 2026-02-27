package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"xart-cli/internal/xart"
)

func newAPICommand() *cobra.Command {
	apiCmd := &cobra.Command{
		Use:   "api",
		Short: "Низкоуровневые вызовы API",
	}

	apiCmd.AddCommand(
		newAPIGetCommand(),
		newAPIPostCommand(),
		newAPIRequestCommand(),
	)

	return apiCmd
}

func newAPIGetCommand() *cobra.Command {
	var query []string
	var headers []string
	var useV2 bool
	var requireAuth bool

	cmd := &cobra.Command{
		Use:   "get <path>",
		Short: "GET запрос к API",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			queryValues, err := parseQueryValues(query)
			if err != nil {
				return err
			}
			headerValues, err := parseHeaderValues(headers)
			if err != nil {
				return err
			}

			token := ""
			if requireAuth {
				token, err = tokenRequired()
				if err != nil {
					return err
				}
			}

			payload, err := rt.client.Do(withContext(), xart.Request{
				Method:  "GET",
				Path:    args[0],
				Query:   queryValues,
				Headers: headerValues,
				Token:   token,
				UseV2:   useV2,
			})
			if err != nil {
				return err
			}
			return printPayload(payload)
		},
	}

	cmd.Flags().StringArrayVarP(&query, "query", "q", nil, "Параметры query в формате key=value (можно повторять)")
	cmd.Flags().StringArrayVarP(&headers, "header", "H", nil, "Заголовки в формате key=value (можно повторять)")
	cmd.Flags().BoolVar(&useV2, "v2", false, "Добавить заголовок API-Version: v2")
	cmd.Flags().BoolVar(&requireAuth, "auth", false, "Добавить токен из конфига и требовать авторизацию")
	return cmd
}

func newAPIPostCommand() *cobra.Command {
	var query []string
	var headers []string
	var body string
	var bodyFile string
	var useV2 bool
	var requireAuth bool

	cmd := &cobra.Command{
		Use:   "post <path>",
		Short: "POST запрос к API",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			queryValues, err := parseQueryValues(query)
			if err != nil {
				return err
			}
			headerValues, err := parseHeaderValues(headers)
			if err != nil {
				return err
			}
			payloadBody, err := parseJSONBody(body, bodyFile)
			if err != nil {
				return err
			}
			bodyBytes, err := jsonMarshal(payloadBody)
			if err != nil {
				return err
			}

			token := ""
			if requireAuth {
				token, err = tokenRequired()
				if err != nil {
					return err
				}
			}

			payload, err := rt.client.Do(withContext(), xart.Request{
				Method:  "POST",
				Path:    args[0],
				Query:   queryValues,
				Headers: headerValues,
				Body:    bodyBytes,
				Token:   token,
				UseV2:   useV2,
			})
			if err != nil {
				return err
			}
			return printPayload(payload)
		},
	}

	cmd.Flags().StringArrayVarP(&query, "query", "q", nil, "Параметры query в формате key=value (можно повторять)")
	cmd.Flags().StringArrayVarP(&headers, "header", "H", nil, "Заголовки в формате key=value (можно повторять)")
	cmd.Flags().StringVar(&body, "body", "", "JSON-тело строкой")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Путь к JSON-файлу с телом")
	cmd.Flags().BoolVar(&useV2, "v2", false, "Добавить заголовок API-Version: v2")
	cmd.Flags().BoolVar(&requireAuth, "auth", false, "Добавить токен из конфига и требовать авторизацию")
	return cmd
}

func newAPIRequestCommand() *cobra.Command {
	var method string
	var query []string
	var headers []string
	var body string
	var bodyFile string
	var useV2 bool
	var requireAuth bool

	cmd := &cobra.Command{
		Use:   "request <path>",
		Short: "Произвольный HTTP-запрос к API",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			queryValues, err := parseQueryValues(query)
			if err != nil {
				return err
			}
			headerValues, err := parseHeaderValues(headers)
			if err != nil {
				return err
			}

			token := ""
			if requireAuth {
				token, err = tokenRequired()
				if err != nil {
					return err
				}
			}

			upperMethod := strings.ToUpper(strings.TrimSpace(method))
			if upperMethod == "" {
				return fmt.Errorf("method is required")
			}

			var bodyPayload []byte
			if upperMethod != "GET" && upperMethod != "DELETE" {
				payloadBody, err := parseJSONBody(body, bodyFile)
				if err != nil {
					return err
				}
				if payloadBody != nil {
					bodyPayload, err = jsonMarshal(payloadBody)
					if err != nil {
						return err
					}
				}
			}

			payload, err := rt.client.Do(withContext(), xart.Request{
				Method:  upperMethod,
				Path:    args[0],
				Query:   queryValues,
				Headers: headerValues,
				Body:    bodyPayload,
				Token:   token,
				UseV2:   useV2,
			})
			if err != nil {
				return err
			}
			return printPayload(payload)
		},
	}

	cmd.Flags().StringVar(&method, "method", "GET", "HTTP-метод")
	cmd.Flags().StringArrayVarP(&query, "query", "q", nil, "Параметры query в формате key=value (можно повторять)")
	cmd.Flags().StringArrayVarP(&headers, "header", "H", nil, "Заголовки в формате key=value (можно повторять)")
	cmd.Flags().StringVar(&body, "body", "", "JSON-тело строкой")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Путь к JSON-файлу с телом")
	cmd.Flags().BoolVar(&useV2, "v2", false, "Добавить заголовок API-Version: v2")
	cmd.Flags().BoolVar(&requireAuth, "auth", false, "Добавить токен из конфига и требовать авторизацию")
	return cmd
}

func parseHeaderValues(values []string) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	headers := make(map[string]string, len(values))
	for _, kv := range values {
		key, value, ok := strings.Cut(kv, "=")
		if !ok {
			return nil, fmt.Errorf("invalid header format %q, expected key=value", kv)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return nil, fmt.Errorf("empty header key in %q", kv)
		}
		headers[key] = value
	}
	return headers, nil
}

func jsonMarshal(value any) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode body: %w", err)
	}
	return data, nil
}
