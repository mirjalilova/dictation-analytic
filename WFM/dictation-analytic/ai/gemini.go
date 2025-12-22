package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type AiSuggestion struct {
    Message string `json:"message"`
}

func AugmentWithGemini(apiKey, model, original, user string, localResult map[string]interface{}) (AiSuggestion, error) {

    prompt := fmt.Sprintf(`You are a proofreading assistant. Original: "%s"
User text: "%s"
The local tool produced this JSON: %v
Please produce a short suggestion: list of spelling mistakes, replacements, and corrected user text.
Response language: Uzbek.`, original, user, localResult)

    payload := map[string]interface{}{
        "contents": []map[string]interface{}{
            {
                "parts": []map[string]string{
                    {"text": prompt},
                },
            },
        },
    }

    body, _ := json.Marshal(payload)
	fmt.Println("model", model)
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		model,
		apiKey,
	)

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
    if err != nil {
        return AiSuggestion{}, err
    }

    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{Timeout: 20 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return AiSuggestion{}, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return AiSuggestion{}, fmt.Errorf("non-200 from gemini: %s", resp.Status)
    }

    var respBody map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&respBody)

    text := ""
    if c, ok := respBody["candidates"]; ok {
        arr := c.([]interface{})
        first := arr[0].(map[string]interface{})
        content := first["content"].(map[string]interface{})
        parts := content["parts"].([]interface{})
        text = parts[0].(map[string]interface{})["text"].(string)
    }

    return AiSuggestion{Message: text}, nil
}
