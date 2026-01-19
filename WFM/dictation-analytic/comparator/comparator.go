package comparator

import (
	"math"
	"regexp"
	"strings"
)

type opType int

const (
	match opType = iota
	substitution
	insertion
	deletion
)

type op struct {
	typ opType
	a   string
	b   string
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

var tokenRe = regexp.MustCompile(`[\p{L}\p{M}\p{N}]+|[^\s\p{L}\p{M}\p{N}]`)

func tokenize(s string) []string {
	return tokenRe.FindAllString(s, -1)
}

func isPunct(s string) bool {
	return regexp.MustCompile(`^[^\w\s]$`).MatchString(s)
}

func equalFold(a, b string) bool {
	return strings.EqualFold(a, b)
}

func levenshtein(a, b string) int {
	aLower := strings.ToLower(a)
	bLower := strings.ToLower(b)
	la := len(aLower)
	lb := len(bLower)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	dp := make([][]int, la+1)
	for i := range dp {
		dp[i] = make([]int, lb+1)
	}
	for i := 0; i <= la; i++ {
		dp[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		dp[0][j] = j
	}
	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 0
			if aLower[i-1] != bLower[j-1] {
				cost = 1
			}
			dp[i][j] = min(dp[i-1][j]+1, dp[i][j-1]+1, dp[i-1][j-1]+cost)
		}
	}
	return dp[la][lb]
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func scoreBetween(a, b string) int {
    if strings.EqualFold(a, b) {
        return 200
    }

    if isPunct(a) != isPunct(b) {
        return -1000
    }

    if isPunct(a) && isPunct(b) {
        return -200
    }

    d := levenshtein(a, b)

    if d == 1 {
        return 80
    }
    if d == 2 {
        return 40
    }

    return -1000
}

func align(a, b []string) []op {
    la := len(a)
    lb := len(b)

    type cell struct {
        score int
        prev  int
    }

    dp := make([][]cell, la+1)
    for i := range dp {
        dp[i] = make([]cell, lb+1)
    }

    const gapPenalty = -120

    dp[0][0] = cell{score: 0, prev: -1}
    for i := 1; i <= la; i++ {
        dp[i][0] = cell{score: dp[i-1][0].score + gapPenalty, prev: 1}
    }
    for j := 1; j <= lb; j++ {
        dp[0][j] = cell{score: dp[0][j-1].score + gapPenalty, prev: 2}
    }

    for i := 1; i <= la; i++ {
        for j := 1; j <= lb; j++ {
            diag := dp[i-1][j-1].score + scoreBetween(a[i-1], b[j-1])
            up   := dp[i-1][j].score + gapPenalty
            left := dp[i][j-1].score + gapPenalty

            best := diag
            prev := 0

            if up > best {
                best = up
                prev = 1
            }
            if left > best {
                best = left
                prev = 2
            }

            dp[i][j] = cell{score: best, prev: prev}
        }
    }

    i, j := la, lb
    ops := []op{}

    for i > 0 || j > 0 {
        if i > 0 && j > 0 && dp[i][j].prev == 0 {
            score := scoreBetween(a[i-1], b[j-1])

            if score == 200 {
                ops = append([]op{{typ: match, a: a[i-1], b: b[j-1]}}, ops...)
            } else {
                ops = append([]op{{typ: substitution, a: a[i-1], b: b[j-1]}}, ops...)
            }

            i--
            j--
            continue
        }

        if i > 0 && (j == 0 || dp[i][j].prev == 1) {
            ops = append([]op{{typ: deletion, a: a[i-1]}}, ops...)
            i--
            continue
        }

        if j > 0 && (i == 0 || dp[i][j].prev == 2) {
            ops = append([]op{{typ: insertion, b: b[j-1]}}, ops...)
            j--
            continue
        }
    }

    return ops
}


func CompareTexts(original, user string) map[string]interface{} {
	a := tokenize(original)
	b := tokenize(user)

	ops := align(a, b)

	spelling := map[string][]string{}
	replacement := map[string][]string{}
	missingWords := []string{}
	extraWords := []string{}
	punctuationMissing := []string{}
	punctuationExtra := []string{}
	punctuationMismatch := []string{}

	for _, o := range ops {
		switch o.typ {
		case match:
		case substitution:
			if isPunct(o.a) && isPunct(o.b) {
				if o.a != o.b {
					punctuationMismatch = append(punctuationMismatch, o.a+" => "+o.b)
				}
			} else if isPunct(o.a) && !isPunct(o.b) {
				punctuationMissing = append(punctuationMissing, o.a)
				extraWords = append(extraWords, o.b)
			} else if !isPunct(o.a) && isPunct(o.b) {
				punctuationExtra = append(punctuationExtra, o.b)
				missingWords = append(missingWords, o.a)
			} else {
				d := levenshtein(o.a, o.b)
				if d > 0 && d <= 2 {
					spelling[o.a] = append(spelling[o.a], o.b)
				} else {
					replacement[o.a] = append(replacement[o.a], o.b)
				}
			}
		case deletion:
			if isPunct(o.a) {
				punctuationMissing = append(punctuationMissing, o.a)
			} else {
				missingWords = append(missingWords, o.a)
			}
		case insertion:
			if isPunct(o.b) {
				punctuationExtra = append(punctuationExtra, o.b)
			} else {
				extraWords = append(extraWords, o.b)
			}
		}
	}

	res := map[string]interface{}{
		"spelling":             spelling,
		"replacement":          replacement,
		"missing":              missingWords,
		"extra":                extraWords,
		"punctuation_missing":  punctuationMissing,
		"punctuation_extra":    punctuationExtra,
		"punctuation_mismatch": punctuationMismatch,
	}
	return res
}

func FullCompare(original, user string) []WordResult {

	origTokens := tokenize(original)
	userTokens := tokenize(user)
	ops := align(origTokens, userTokens)

	results := []WordResult{}
	id := 1

	for _, o := range ops {
		switch o.typ {

		case match:
			results = append(results, WordResult{
				ID:         id,
				Word:       o.b,
				HasMistake: false,
			})
			id++

		case substitution:
			if isPunct(o.a) && isPunct(o.b) {
				results = append(results, WordResult{
					ID:            id,
					Word:          o.b,
					HasMistake:    true,
					MistakeType:   "punctuation_mismatch",
					OriginalWord:  o.a,
					RemovedPoints: 0.5,
				})
				id++
				continue
			}

			if isPunct(o.a) && !isPunct(o.b) {
				results = append(results, WordResult{
					ID:            id,
					Word:          o.b,
					HasMistake:    true,
					MistakeType:   "extra_word",
					RemovedPoints: 2,
				})
				id++
				continue
			}

			if !isPunct(o.a) && isPunct(o.b) {
				results = append(results, WordResult{
					ID:            id,
					Word:          o.b,
					HasMistake:    true,
					MistakeType:   "punctuation_extra",
					OriginalWord:  o.a,
					RemovedPoints: 0.5,
				})
				id++
				continue
			}

			results = append(results, WordResult{
				ID:            id,
				Word:          o.b,
				HasMistake:    true,
				MistakeType:   "spelling",
				OriginalWord:  o.a,
				RemovedPoints: 1,
			})
			id++

		case deletion:
			if isPunct(o.a) {
				results = append(results, WordResult{
					ID:            id,
					Word:          "",
					HasMistake:    true,
					MistakeType:   "punctuation_missing",
					OriginalWord:  o.a,
					RemovedPoints: 0.5,
				})
			} else {
				results = append(results, WordResult{
					ID:            id,
					Word:          "",
					HasMistake:    true,
					MistakeType:   "missing_word",
					OriginalWord:  o.a,
					RemovedPoints: 2,
				})
			}
			id++

		case insertion:
			if isPunct(o.b) {
				results = append(results, WordResult{
					ID:            id,
					Word:          o.b,
					HasMistake:    true,
					MistakeType:   "punctuation_extra",
					RemovedPoints: 0.5,
				})
			} else {
				results = append(results, WordResult{
					ID:            id,
					Word:          o.b,
					HasMistake:    true,
					MistakeType:   "extra_word",
					RemovedPoints: 2,
				})
			}
			id++
		}
	}

	return results
}

func countOriginalWordsOnly(original string) int {
	toks := tokenize(original)
	cnt := 0
	for _, t := range toks {
		if !isPunct(t) {
			cnt++
		}
	}
	return cnt
}

func sumRemovedPoints(data []WordResult) float64 {
	sum := 0.0
	for _, w := range data {
		sum += w.RemovedPoints
	}
	return sum
}

func countMistakes(data []WordResult) int {
	c := 0
	for _, w := range data {
		if w.HasMistake {
			c++
		}
	}
	return c
}

func AnalyzeText(original, user string) CompareAnalytics {
    data := FullCompare(original, user)

    origWordCount := float64(countOriginalWordsOnly(original))
    if origWordCount == 0 {
        return CompareAnalytics{
            CountMistakes: 0,
            Similarity:    100,
            TotalScore:    100,
            Data:          data,
        }
    }

    correctWords := 0.0
    for _, w := range data {
        if !w.HasMistake && !isPunct(w.Word) {
            correctWords++
        }
    }
    similarity := (correctWords / origWordCount) * 100

    totalMistakePoints := sumRemovedPoints(data)

    penaltyFactor := 50.0
    totalScore := 100 - (totalMistakePoints/origWordCount)*penaltyFactor
    if totalScore < 0 {
        totalScore = 0
    }

    return CompareAnalytics{
        CountMistakes: countMistakes(data),
        Similarity:    round(similarity, 2),
        TotalScore:    round(totalScore, 2),
        Data:          data,
    }
}

func round(val float64, precision int) float64 {
    pow := math.Pow(10, float64(precision))
    return math.Round(val*pow) / pow
}
