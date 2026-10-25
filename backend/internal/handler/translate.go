package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/0x2E/fusion/internal/store"
	"github.com/0x2E/fusion/internal/translator"
	"github.com/gin-gonic/gin"
)

type TranslateItemResponse struct {
	ID              int64  `json:"id"`
	OriginalTitle   string `json:"original_title"`
	TranslatedTitle string `json:"translated_title"`
	DetectedLang    string `json:"detected_lang"`
	Cached          bool   `json:"cached"`
}

type TranslateTextRequest struct {
	Text string `json:"text" binding:"required"`
}

type TranslateTextResponse struct {
	OriginalText   string `json:"original_text"`
	TranslatedText string `json:"translated_text"`
	DetectedLang   string `json:"detected_lang"`
	Cached         bool   `json:"cached"`
}

func hashText(text string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(text)))
	return hex.EncodeToString(sum[:])
}

func (h *Handler) translateItemTitle(c *gin.Context) {
	userID := getUserID(c)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	item, err := h.store.GetItem(userID, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	title := strings.TrimSpace(item.Title)
	if title == "" {
		c.JSON(http.StatusOK, TranslateItemResponse{
			ID:              item.ID,
			OriginalTitle:   "",
			TranslatedTitle: "",
			DetectedLang:    translator.LangEnglish,
			Cached:          true,
		})
		return
	}

	detectedLang := translator.DetectLanguage(title)

	// If item already has a non-empty translated title, return it immediately
	if strings.TrimSpace(item.TranslatedTitle) != "" {
		c.JSON(http.StatusOK, TranslateItemResponse{
			ID:              item.ID,
			OriginalTitle:   item.Title,
			TranslatedTitle: item.TranslatedTitle,
			DetectedLang:    detectedLang,
			Cached:          true,
		})
		return
	}

	contentHash := hashText(title)

	// Check global persistent cache
	cachedTranslation, err := h.store.GetCachedTranslation(contentHash)
	if err == nil && cachedTranslation != "" {
		_ = h.store.UpdateItemTranslatedTitle(userID, item.ID, cachedTranslation)
		c.JSON(http.StatusOK, TranslateItemResponse{
			ID:              item.ID,
			OriginalTitle:   item.Title,
			TranslatedTitle: cachedTranslation,
			DetectedLang:    detectedLang,
			Cached:          true,
		})
		return
	}

	if h.translator == nil {
		h.translator = translator.New()
	}

	translated, err := h.translator.TranslateTitle(c.Request.Context(), title)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "translation service error: " + err.Error()})
		return
	}

	// Persist translation to item and cache
	_ = h.store.UpdateItemTranslatedTitle(userID, item.ID, translated)
	_ = h.store.SetCachedTranslation(contentHash, detectedLang, title, translated)

	c.JSON(http.StatusOK, TranslateItemResponse{
		ID:              item.ID,
		OriginalTitle:   item.Title,
		TranslatedTitle: translated,
		DetectedLang:    detectedLang,
		Cached:          false,
	})
}

func (h *Handler) translateText(c *gin.Context) {
	var req TranslateTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text is required"})
		return
	}

	text := strings.TrimSpace(req.Text)
	if text == "" {
		c.JSON(http.StatusOK, TranslateTextResponse{
			OriginalText:   "",
			TranslatedText: "",
			DetectedLang:   translator.LangEnglish,
			Cached:         true,
		})
		return
	}

	detectedLang := translator.DetectLanguage(text)
	contentHash := hashText(text)

	cachedTranslation, err := h.store.GetCachedTranslation(contentHash)
	if err == nil && cachedTranslation != "" {
		c.JSON(http.StatusOK, TranslateTextResponse{
			OriginalText:   text,
			TranslatedText: cachedTranslation,
			DetectedLang:   detectedLang,
			Cached:         true,
		})
		return
	}

	if h.translator == nil {
		h.translator = translator.New()
	}

	translated, err := h.translator.TranslateTitle(c.Request.Context(), text)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "translation service error: " + err.Error()})
		return
	}

	_ = h.store.SetCachedTranslation(contentHash, detectedLang, text, translated)

	c.JSON(http.StatusOK, TranslateTextResponse{
		OriginalText:   text,
		TranslatedText: translated,
		DetectedLang:   detectedLang,
		Cached:         false,
	})
}

type TranslateBatchRequest struct {
	IDs []int64 `json:"ids" binding:"required"`
}

type TranslateBatchResponseItem struct {
	ID              int64  `json:"id"`
	OriginalTitle   string `json:"original_title"`
	TranslatedTitle string `json:"translated_title"`
	DetectedLang    string `json:"detected_lang"`
	Cached          bool   `json:"cached"`
}

type TranslateBatchResponse struct {
	Items []TranslateBatchResponseItem `json:"items"`
}

func (h *Handler) translateItemsBatch(c *gin.Context) {
	userID := getUserID(c)
	var req TranslateBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids array is required"})
		return
	}

	if len(req.IDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"items": []TranslateBatchResponseItem{}})
		return
	}

	if len(req.IDs) > 100 {
		req.IDs = req.IDs[:100]
	}

	if h.translator == nil {
		h.translator = translator.New()
	}

	results := make([]TranslateBatchResponseItem, len(req.IDs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // Bounded concurrency limit

	for i, id := range req.IDs {
		wg.Add(1)
		go func(idx int, itemId int64) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			it, err := h.store.GetItem(userID, itemId)
			if err != nil {
				return
			}

			title := strings.TrimSpace(it.Title)
			if title == "" {
				results[idx] = TranslateBatchResponseItem{
					ID:              itemId,
					OriginalTitle:   "",
					TranslatedTitle: "",
					DetectedLang:    translator.LangEnglish,
					Cached:          true,
				}
				return
			}

			detectedLang := translator.DetectLanguage(title)

			// Already translated
			if strings.TrimSpace(it.TranslatedTitle) != "" {
				results[idx] = TranslateBatchResponseItem{
					ID:              itemId,
					OriginalTitle:   title,
					TranslatedTitle: it.TranslatedTitle,
					DetectedLang:    detectedLang,
					Cached:          true,
				}
				return
			}

			contentHash := hashText(title)
			cachedTranslation, err := h.store.GetCachedTranslation(contentHash)
			if err == nil && cachedTranslation != "" {
				_ = h.store.UpdateItemTranslatedTitle(userID, itemId, cachedTranslation)
				results[idx] = TranslateBatchResponseItem{
					ID:              itemId,
					OriginalTitle:   title,
					TranslatedTitle: cachedTranslation,
					DetectedLang:    detectedLang,
					Cached:          true,
				}
				return
			}

			translated, err := h.translator.TranslateTitle(c.Request.Context(), title)
			if err == nil && translated != "" {
				_ = h.store.UpdateItemTranslatedTitle(userID, itemId, translated)
				_ = h.store.SetCachedTranslation(contentHash, detectedLang, title, translated)
				results[idx] = TranslateBatchResponseItem{
					ID:              itemId,
					OriginalTitle:   title,
					TranslatedTitle: translated,
					DetectedLang:    detectedLang,
					Cached:          false,
				}
			} else {
				results[idx] = TranslateBatchResponseItem{
					ID:              itemId,
					OriginalTitle:   title,
					TranslatedTitle: title,
					DetectedLang:    detectedLang,
					Cached:          false,
				}
			}
		}(i, id)
	}

	wg.Wait()

	validResults := make([]TranslateBatchResponseItem, 0, len(results))
	for _, res := range results {
		if res.ID > 0 {
			validResults = append(validResults, res)
		}
	}

	c.JSON(http.StatusOK, gin.H{"items": validResults})
}
