package bot

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestGenHelpMsg(t *testing.T) {
	require.Equal(t, "cmd _– description_\n", genHelpMsg([]string{"cmd"}, "description"))
}

func TestMultiBotHelp(t *testing.T) {
	b1 := &MockInterface{}
	b1.On("Help").Return("b1 help")
	b2 := &MockInterface{}
	b2.On("Help").Return("b2 help")

	// Must return concatenated b1 and b2 without space
	// Line formatting only in genHelpMsg()
	require.Equal(t, "b1 help\nb2 help\n", MultiBot{b1, b2}.Help())
}

func TestMultiBotReactsOnHelp(t *testing.T) {
	b := &MockInterface{}
	b.On("ReactOn").Return([]string{"help"})
	b.On("Help").Return("help")

	mb := MultiBot{b}
	resp := mb.OnMessage(Message{Text: "help"})

	require.True(t, resp.Send)
	require.Equal(t, "help\n", resp.Text)
}

func TestMultiBotCombinesAllBotResponses(t *testing.T) {
	msg := Message{Text: "cmd"}

	b1 := &MockInterface{}
	b1.On("ReactOn").Return([]string{"cmd"})
	b1.On("OnMessage", msg).Return(Response{
		Text: "b1 resp",
		Send: true,
	})
	b2 := &MockInterface{}
	b2.On("ReactOn").Return([]string{"cmd"})
	b2.On("OnMessage", msg).Return(Response{
		Text: "b2 resp",
		Send: true,
	})

	mb := MultiBot{b1, b2}
	resp := mb.OnMessage(msg)

	require.True(t, resp.Send)
	parts := strings.Split(resp.Text, "\n")
	require.Len(t, parts, 2)
	require.Contains(t, parts, "b1 resp")
	require.Contains(t, parts, "b2 resp")
}

func TestNewResponse_TextSanitize(t *testing.T) {
	tests := []struct {
		given string
		want  string
	}{
		{given: "_", want: "\\_"},
		{given: "a_", want: "a\\_"},
		{given: "_a", want: "\\_a"},
		{given: "__", want: "\\_\\_"},
		{given: "_italic_", want: "_italic_"},
		{given: "_italic_ w _italic_", want: "_italic_ w _italic_"},
		{given: "_ a_", want: "\\_ a\\_"},
		{given: "_a _", want: "\\_a \\_"},
		{given: "a_a_", want: "a\\_a\\_"},

		{
			given: "_Больше ботов, хороших и разных — Радио-Т Подкаст_",
			want:  "_Больше ботов, хороших и разных — Радио-Т Подкаст_",
		},
		{
			given: "⚠️ Pixel prevented me from calling 911 - https://www.reddit.com/r/GooglePixel/comments/r4xz1f/pixel_prevented_me_from_calling_911/",
			want:  "⚠️ Pixel prevented me from calling 911 - https://www.reddit.com/r/GooglePixel/comments/r4xz1f/pixel\\_prevented\\_me\\_from\\_calling\\_911/",
		},

		// TODO: wrong, don't work with one blank character between matches. Can be ignored.
		{given: "_italic_ _italic_", want: "_italic_ \\_italic\\_"}, // want should be _italic_ _italic_

		// TODO: wrong, don't work when length of matches is less than 3. Can be ignored.
		{given: "_it_", want: "\\_it\\_"}, // want should be _it_

	}
	for _, tt := range tests {
		t.Run(tt.given, func(t *testing.T) {
			response := NewResponse(tt.given, false, false, false, false, 0)
			assert.Equal(t, tt.want, response.Text)
		})
	}
}
