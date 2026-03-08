package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"xart-cli/internal/xart"
)

func requestGET(path string, query url.Values, useV2 bool, token string) xart.Request {
	return xart.Request{
		Method: "GET",
		Path:   path,
		Query:  query,
		Token:  token,
		UseV2:  useV2,
	}
}

func requestPOST(path string, query url.Values, body any, useV2 bool, token string) xart.Request {
	rawBody, _ := jsonMarshal(body)
	return xart.Request{
		Method: "POST",
		Path:   path,
		Query:  query,
		Body:   rawBody,
		Token:  token,
		UseV2:  useV2,
		Headers: map[string]string{
			"Content-Type": "application/json; charset=UTF-8",
		},
	}
}

func doAndPrint(req xart.Request) error {
	payload, err := rt.client.Do(withContext(), req)
	if err != nil {
		return err
	}
	return printPayload(payload)
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0
		}
		return int(n)
	case string:
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}

func boolFromAny(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v == "true"
	case int:
		return v != 0
	case float64:
		return v != 0
	default:
		return false
	}
}

func mustTokenOrError() (string, error) {
	token, err := tokenRequired()
	if err != nil {
		return "", fmt.Errorf("auth required: %w", err)
	}
	return token, nil
}
