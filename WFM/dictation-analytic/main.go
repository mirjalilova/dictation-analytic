package main

import (
	"net/http"

	"github.com/gin-gonic/gin"

	swagFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "dictation-analytic/docs"

	"dictation-analytic/comparator"
	// "dictation-analytic/ai"
)

// @title Dictation Analytic API
// @version 1.0
// @description API for comparing original and user dictation text.
// @BasePath /
type CompareRequest struct {
    Original string `json:"original" binding:"required"`
    User     string `json:"user" binding:"required"`
    // UseAI    bool   `json:"use_ai"`
}
type WordResult struct {
	ID            int     `json:"id"`
	Word          string  `json:"word"`
	HasMistake    bool    `json:"has_mistake"`
	MistakeType   string  `json:"mistake_type,omitempty"`
	OriginalWord  string  `json:"original_word,omitempty"`
	RemovedPoints float64 `json:"removed_points,omitempty"`
}

type CompareAnalytics struct {
    CountMistakes int          `json:"count_mistakes"`
    Similarity    float64      `json:"similarity"`
    TotalScore    float64      `json:"total_score"`
    Data          []WordResult `json:"data"`
}

// CompareTexts godoc
// @Summary Compare original and user text
// @Description Returns spelling, missing, extra, punctuation errors etc.
// @Accept json
// @Produce json
// @Param data body CompareRequest true "Texts to compare"
// @Success 200 {object} CompareAnalytics
// @Router /compare [post]
func main() {
    r := gin.Default()

    r.POST("/compare", func(c *gin.Context) {
        var req CompareRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{
                "error": err.Error(),
            })
            return
        }

		results := comparator.AnalyzeText(req.Original, req.User)

        // if req.UseAI {
        //     apiKey := os.Getenv("GEMINI_API_KEY")
        //     model := os.Getenv("GEMINI_MODEL")
        //     if apiKey != "" && model != "" {
        //         if aiRes, err := ai.AugmentWithGemini(apiKey, model, req.Original, req.User, res); err == nil {
        //             res["ai_suggestion"] = aiRes
        //         } else {
        //             res["ai_error"] = err.Error()
        //         }
        //     } else {
        //         res["ai_error"] = "GEMINI_API_KEY or GEMINI_MODEL not set"
        //     }
        // }

        c.JSON(http.StatusOK, results)
    })

    r.GET("/", func(c *gin.Context) {
        c.String(http.StatusOK, "Text Compare API is running. POST /compare")
    })

    r.GET("/swagger/*any", ginSwagger.WrapHandler(swagFiles.Handler))

    r.Run(":8080")
}
