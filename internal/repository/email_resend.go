package repository

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"

	"github.com/resend/resend-go/v2"
	"github.com/tsongpon/delphi/internal/service"
)

// Compile-time check that ResendEmailSender implements service.EmailSender.
var _ service.EmailSender = (*ResendEmailSender)(nil)

//go:embed templates/feedback_digest.html
var feedbackDigestTemplate string

//go:embed templates/password_reset.html
var passwordResetTemplate string

var digestTmpl = template.Must(template.New("digest").Parse(feedbackDigestTemplate))
var passwordResetTmpl = template.Must(template.New("passwordReset").Parse(passwordResetTemplate))

// ResendEmailSender sends transactional emails via the Resend API.
type ResendEmailSender struct {
	client    *resend.Client
	fromEmail string
	appURL    string
}

// NewResendEmailSender creates a ResendEmailSender using the provided API key and sender address.
func NewResendEmailSender(apiKey, fromEmail, appURL string) *ResendEmailSender {
	return &ResendEmailSender{
		client:    resend.NewClient(apiKey),
		fromEmail: fromEmail,
		appURL:    appURL,
	}
}

// SendFeedbackDigest emails toEmail a digest notifying them of count new feedbacks.
func (s *ResendEmailSender) SendFeedbackDigest(ctx context.Context, toName, toEmail string, count int) error {
	data := struct {
		Name  string
		Count int
		URL   string
	}{
		Name:  toName,
		Count: count,
		URL:   s.appURL,
	}

	var buf bytes.Buffer
	if err := digestTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	subject := fmt.Sprintf("You have %d new feedback", count)
	if count > 1 {
		subject += "s"
	}
	subject += " — Feedback360"

	params := &resend.SendEmailRequest{
		From:    s.fromEmail,
		To:      []string{toEmail},
		Subject: subject,
		Html:    buf.String(),
	}

	if _, err := s.client.Emails.Send(params); err != nil {
		return fmt.Errorf("failed to send feedback digest email to %s: %w", toEmail, err)
	}

	return nil
}

// SendPasswordResetEmail emails toEmail a link to reset their password.
func (s *ResendEmailSender) SendPasswordResetEmail(ctx context.Context, toName, toEmail, resetLink string) error {
	data := struct {
		Name      string
		ResetLink string
	}{
		Name:      toName,
		ResetLink: resetLink,
	}

	var buf bytes.Buffer
	if err := passwordResetTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to render password reset email template: %w", err)
	}

	params := &resend.SendEmailRequest{
		From:    s.fromEmail,
		To:      []string{toEmail},
		Subject: "Reset your password — Feedback360",
		Html:    buf.String(),
	}

	if _, err := s.client.Emails.Send(params); err != nil {
		return fmt.Errorf("failed to send password reset email to %s: %w", toEmail, err)
	}

	return nil
}
