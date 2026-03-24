package handler

import (
	"bytes"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/labstack/echo/v5"
	"github.com/tsongpon/delphi/internal/model"
)

//go:embed fonts/Sarabun-Regular.ttf
var sarabunRegular []byte

//go:embed fonts/Sarabun-SemiBold.ttf
var sarabunSemiBold []byte

var (
	fontDirOnce sync.Once
	fontDir     string
	fontDirErr  error
)

func initFontDir() {
	fontDirOnce.Do(func() {
		dir, err := os.MkdirTemp("", "delphi-fonts-*")
		if err != nil {
			fontDirErr = fmt.Errorf("failed to create font temp dir: %w", err)
			return
		}
		if err := os.WriteFile(filepath.Join(dir, "Sarabun-Regular.ttf"), sarabunRegular, 0644); err != nil {
			fontDirErr = fmt.Errorf("failed to write regular font: %w", err)
			return
		}
		if err := os.WriteFile(filepath.Join(dir, "Sarabun-SemiBold.ttf"), sarabunSemiBold, 0644); err != nil {
			fontDirErr = fmt.Errorf("failed to write semibold font: %w", err)
			return
		}
		fontDir = dir
	})
}

func (h *FeedbackHandler) ExportMyFeedbacksPDF(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	userName, _ := c.Get("name").(string)

	ctx := c.Request().Context()
	entries, err := h.FeedbackService.ExportFeedbacksForUser(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate report"})
	}

	initFontDir()
	if fontDirErr != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to initialize fonts"})
	}

	now := time.Now()
	cutoff := now.AddDate(-1, 0, 0)

	pdf := gofpdf.New("P", "mm", "A4", fontDir)
	pdf.AddUTF8Font("Sarabun", "", "Sarabun-Regular.ttf")
	pdf.AddUTF8Font("Sarabun", "B", "Sarabun-SemiBold.ttf")
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	pageW, _ := pdf.GetPageSize()
	contentW := pageW - 40

	// ── Header ────────────────────────────────────────────────────────────────
	pdf.SetFont("Sarabun", "B", 20)
	pdf.SetTextColor(30, 30, 30)
	pdf.CellFormat(contentW, 10, "360\u00b0 Feedback Report", "", 1, "L", false, 0, "")

	pdf.SetFont("Sarabun", "", 11)
	pdf.SetTextColor(90, 90, 90)
	if userName != "" {
		pdf.CellFormat(contentW, 7, userName, "", 1, "L", false, 0, "")
	}
	pdf.CellFormat(contentW, 7,
		fmt.Sprintf("Period: %s \u2013 %s",
			cutoff.Format("Jan 2006"),
			now.Format("Jan 2006")),
		"", 1, "L", false, 0, "")

	pdf.Ln(4)
	pdf.SetDrawColor(200, 200, 200)
	pdf.Line(20, pdf.GetY(), pageW-20, pdf.GetY())
	pdf.Ln(6)

	// ── Summary ───────────────────────────────────────────────────────────────
	pdf.SetFont("Sarabun", "B", 13)
	pdf.SetTextColor(30, 30, 30)
	pdf.CellFormat(contentW, 8, "Summary", "", 1, "L", false, 0, "")
	pdf.Ln(2)

	pdf.SetFont("Sarabun", "", 10)
	pdf.SetTextColor(60, 60, 60)
	pdf.CellFormat(contentW, 6, fmt.Sprintf("Total feedbacks received: %d", len(entries)), "", 1, "L", false, 0, "")
	pdf.Ln(2)

	if len(entries) > 0 {
		var commSum, leadSum, techSum, collabSum, delivSum float64
		for _, e := range entries {
			commSum += float64(e.Feedback.CommunicationScore)
			leadSum += float64(e.Feedback.LeadershipScore)
			techSum += float64(e.Feedback.TechnicalScore)
			collabSum += float64(e.Feedback.CollaborationScore)
			delivSum += float64(e.Feedback.DeliveryScore)
		}
		n := float64(len(entries))

		type avgRow struct{ label, val string }
		rows := []avgRow{
			{"Communication", fmt.Sprintf("%.1f / 5", commSum/n)},
			{"Leadership", fmt.Sprintf("%.1f / 5", leadSum/n)},
			{"Technical Skills", fmt.Sprintf("%.1f / 5", techSum/n)},
			{"Collaboration", fmt.Sprintf("%.1f / 5", collabSum/n)},
			{"Delivery", fmt.Sprintf("%.1f / 5", delivSum/n)},
		}
		colW := contentW / 2
		for _, r := range rows {
			pdf.SetFont("Sarabun", "", 10)
			pdf.CellFormat(colW, 6, r.label, "", 0, "L", false, 0, "")
			pdf.SetFont("Sarabun", "B", 10)
			pdf.CellFormat(colW, 6, r.val, "", 1, "L", false, 0, "")
		}
	}

	pdf.Ln(4)
	pdf.Line(20, pdf.GetY(), pageW-20, pdf.GetY())
	pdf.Ln(6)

	// ── Individual feedback entries ────────────────────────────────────────────
	pdf.SetFont("Sarabun", "B", 13)
	pdf.SetTextColor(30, 30, 30)
	pdf.CellFormat(contentW, 8, "Feedback Entries", "", 1, "L", false, 0, "")
	pdf.Ln(2)

	if len(entries) == 0 {
		pdf.SetFont("Sarabun", "", 10)
		pdf.SetTextColor(120, 120, 120)
		pdf.CellFormat(contentW, 8, "No feedback received in the past 12 months.", "", 1, "L", false, 0, "")
	}

	for i, e := range entries {
		f := e.Feedback

		// Page break guard: if less than 60mm remain, add a new page.
		if pdf.GetY() > 247 {
			pdf.AddPage()
			pdf.Ln(4)
		}

		// Entry header: show reviewer name for named feedback, "Anonymous" otherwise.
		reviewer := reviewerLabel(e)

		pdf.SetFillColor(245, 245, 245)
		pdf.SetFont("Sarabun", "B", 10)
		pdf.SetTextColor(30, 30, 30)
		pdf.CellFormat(contentW, 7,
			fmt.Sprintf("%d.  %s  \u2022  %s  \u2022  %s",
				i+1, f.Period, reviewer, f.CreatedAt.Format("Jan 2, 2006")),
			"", 1, "L", true, 0, "")
		pdf.Ln(1)

		// Scores row
		pdf.SetFont("Sarabun", "", 9)
		pdf.SetTextColor(60, 60, 60)
		scores := []struct {
			label string
			val   int
		}{
			{"Comm", f.CommunicationScore},
			{"Lead", f.LeadershipScore},
			{"Tech", f.TechnicalScore},
			{"Collab", f.CollaborationScore},
			{"Delivery", f.DeliveryScore},
		}
		colW := contentW / float64(len(scores))
		for _, s := range scores {
			pdf.CellFormat(colW, 6,
				fmt.Sprintf("%s: %d/5", s.label, s.val),
				"", 0, "L", false, 0, "")
		}
		pdf.Ln(8)

		// Comments
		if f.StrengthsComment != "" {
			pdf.SetFont("Sarabun", "B", 9)
			pdf.SetTextColor(40, 100, 60)
			pdf.CellFormat(contentW, 5, "Strengths", "", 1, "L", false, 0, "")
			pdf.SetFont("Sarabun", "", 9)
			pdf.SetTextColor(60, 60, 60)
			pdf.MultiCell(contentW, 5, f.StrengthsComment, "", "L", false)
			pdf.Ln(1)
		}
		if f.WeaknessesComment != "" {
			pdf.SetFont("Sarabun", "B", 9)
			pdf.SetTextColor(160, 80, 30)
			pdf.CellFormat(contentW, 5, "Areas for Improvement", "", 1, "L", false, 0, "")
			pdf.SetFont("Sarabun", "", 9)
			pdf.SetTextColor(60, 60, 60)
			pdf.MultiCell(contentW, 5, f.WeaknessesComment, "", "L", false)
			pdf.Ln(1)
		}

		pdf.Ln(3)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to render PDF"})
	}

	c.Response().Header().Set("Content-Type", "application/pdf")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=\"feedback-report.pdf\"")
	return c.Blob(http.StatusOK, "application/pdf", buf.Bytes())
}

// reviewerLabel returns the display string for the reviewer column.
// Named feedback shows the reviewer's name; anonymous shows "Anonymous".
func reviewerLabel(e *model.FeedbackExportEntry) string {
	if e.Feedback.Visibility == "anonymous" {
		return "Anonymous"
	}
	if e.ReviewerName != "" {
		return e.ReviewerName
	}
	return "Named Reviewer"
}

