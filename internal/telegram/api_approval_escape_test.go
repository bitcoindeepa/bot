package telegram

import "testing"

// Usernames with Markdown characters must be escaped in approval messages,
// or Telegram rejects the message ("can't parse entities") and the API send
// fails.
func TestAPIRecipientMentionEscapesMarkdown(t *testing.T) {
	cases := map[string]string{
		"naween_95": `@naween\_95`,
		"a*b":       `@a\*b`,
		"a`b":       "@a\\`b",
		"a[b":       `@a\[b`,
		"alice":     "@alice",
	}
	for in, want := range cases {
		if got := apiRecipientMention(in); got != want {
			t.Errorf("apiRecipientMention(%q) = %q, want %q", in, got, want)
		}
	}
}
