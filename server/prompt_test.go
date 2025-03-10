package server

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/template"
)

func tokenize(_ context.Context, s string) (tokens []int, err error) {
	for range strings.Fields(s) {
		tokens = append(tokens, len(tokens))
	}

	return
}

func TestChatPrompt(t *testing.T) {
	type expect struct {
		prompt string
		images [][]byte
	}

	cases := []struct {
		name  string
		limit int
		msgs  []api.Message
		expect
	}{
		{
			name:  "messages",
			limit: 64,
			msgs: []api.Message{
				{Role: "user", Content: "You're a test, Harry!"},
				{Role: "assistant", Content: "I-I'm a what?"},
				{Role: "user", Content: "A test. And a thumping good one at that, I'd wager."},
			},
			expect: expect{
				prompt: "You're a test, Harry! I-I'm a what? A test. And a thumping good one at that, I'd wager. ",
			},
		},
		{
			name:  "truncate messages",
			limit: 1,
			msgs: []api.Message{
				{Role: "user", Content: "You're a test, Harry!"},
				{Role: "assistant", Content: "I-I'm a what?"},
				{Role: "user", Content: "A test. And a thumping good one at that, I'd wager."},
			},
			expect: expect{
				prompt: "A test. And a thumping good one at that, I'd wager. ",
			},
		},
		{
			name:  "truncate messages with image",
			limit: 64,
			msgs: []api.Message{
				{Role: "user", Content: "You're a test, Harry!"},
				{Role: "assistant", Content: "I-I'm a what?"},
				{Role: "user", Content: "A test. And a thumping good one at that, I'd wager.", Images: []api.ImageData{[]byte("something")}},
			},
			expect: expect{
				prompt: "[img-0] A test. And a thumping good one at that, I'd wager. ",
				images: [][]byte{
					[]byte("something"),
				},
			},
		},
		{
			name:  "truncate messages with images",
			limit: 64,
			msgs: []api.Message{
				{Role: "user", Content: "You're a test, Harry!", Images: []api.ImageData{[]byte("something")}},
				{Role: "assistant", Content: "I-I'm a what?"},
				{Role: "user", Content: "A test. And a thumping good one at that, I'd wager.", Images: []api.ImageData{[]byte("somethingelse")}},
			},
			expect: expect{
				prompt: "[img-0] A test. And a thumping good one at that, I'd wager. ",
				images: [][]byte{
					[]byte("somethingelse"),
				},
			},
		},
		{
			name:  "messages with images",
			limit: 2048,
			msgs: []api.Message{
				{Role: "user", Content: "You're a test, Harry!", Images: []api.ImageData{[]byte("something")}},
				{Role: "assistant", Content: "I-I'm a what?"},
				{Role: "user", Content: "A test. And a thumping good one at that, I'd wager.", Images: []api.ImageData{[]byte("somethingelse")}},
			},
			expect: expect{
				prompt: "[img-0] You're a test, Harry! I-I'm a what? [img-1] A test. And a thumping good one at that, I'd wager. ",
				images: [][]byte{
					[]byte("something"),
					[]byte("somethingelse"),
				},
			},
		},
		{
			name:  "message with image tag",
			limit: 2048,
			msgs: []api.Message{
				{Role: "user", Content: "You're a test, Harry! [img]", Images: []api.ImageData{[]byte("something")}},
				{Role: "assistant", Content: "I-I'm a what?"},
				{Role: "user", Content: "A test. And a thumping good one at that, I'd wager.", Images: []api.ImageData{[]byte("somethingelse")}},
			},
			expect: expect{
				prompt: "You're a test, Harry! [img-0] I-I'm a what? [img-1] A test. And a thumping good one at that, I'd wager. ",
				images: [][]byte{
					[]byte("something"),
					[]byte("somethingelse"),
				},
			},
		},
		{
			name:  "messages with interleaved images",
			limit: 2048,
			msgs: []api.Message{
				{Role: "user", Content: "You're a test, Harry!"},
				{Role: "user", Images: []api.ImageData{[]byte("something")}},
				{Role: "user", Images: []api.ImageData{[]byte("somethingelse")}},
				{Role: "assistant", Content: "I-I'm a what?"},
				{Role: "user", Content: "A test. And a thumping good one at that, I'd wager."},
			},
			expect: expect{
				prompt: "You're a test, Harry!\n\n[img-0]\n\n[img-1] I-I'm a what? A test. And a thumping good one at that, I'd wager. ",
				images: [][]byte{
					[]byte("something"),
					[]byte("somethingelse"),
				},
			},
		},
		{
			name:  "truncate message with interleaved images",
			limit: 1024,
			msgs: []api.Message{
				{Role: "user", Content: "You're a test, Harry!"},
				{Role: "user", Images: []api.ImageData{[]byte("something")}},
				{Role: "user", Images: []api.ImageData{[]byte("somethingelse")}},
				{Role: "assistant", Content: "I-I'm a what?"},
				{Role: "user", Content: "A test. And a thumping good one at that, I'd wager."},
			},
			expect: expect{
				prompt: "[img-0] I-I'm a what? A test. And a thumping good one at that, I'd wager. ",
				images: [][]byte{
					[]byte("somethingelse"),
				},
			},
		},
		{
			name:  "message with system prompt",
			limit: 2048,
			msgs: []api.Message{
				{Role: "system", Content: "You are the Test Who Lived."},
				{Role: "user", Content: "You're a test, Harry!"},
				{Role: "assistant", Content: "I-I'm a what?"},
				{Role: "user", Content: "A test. And a thumping good one at that, I'd wager."},
			},
			expect: expect{
				prompt: "You are the Test Who Lived. You're a test, Harry! I-I'm a what? A test. And a thumping good one at that, I'd wager. ",
			},
		},
		{
			name:  "out of order system",
			limit: 2048,
			msgs: []api.Message{
				{Role: "user", Content: "You're a test, Harry!"},
				{Role: "assistant", Content: "I-I'm a what?"},
				{Role: "system", Content: "You are the Test Who Lived."},
				{Role: "user", Content: "A test. And a thumping good one at that, I'd wager."},
			},
			expect: expect{
				prompt: "You're a test, Harry! I-I'm a what? You are the Test Who Lived. A test. And a thumping good one at that, I'd wager. ",
			},
		},
	}

	tmpl, err := template.Parse(`
{{- if .System }}{{ .System }} {{ end }}
{{- if .Prompt }}{{ .Prompt }} {{ end }}
{{- if .Response }}{{ .Response }} {{ end }}`)
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			model := Model{Template: tmpl, ProjectorPaths: []string{"vision"}}
			opts := api.Options{Runner: api.Runner{NumCtx: tt.limit}}
			prompt, images, err := chatPrompt(context.TODO(), &model, tokenize, &opts, tt.msgs, nil)
			if err != nil {
				t.Fatal(err)
			}

			if tt.prompt != prompt {
				t.Errorf("expected %q, got %q", tt.prompt, prompt)
			}

			if diff := cmp.Diff(prompt, tt.prompt); diff != "" {
				t.Errorf("mismatch (-got +want):\n%s", diff)
			}

			if len(images) != len(tt.images) {
				t.Fatalf("expected %d images, got %d", len(tt.images), len(images))
			}

			for i := range images {
				if images[i].ID != i {
					t.Errorf("expected ID %d, got %d", i, images[i].ID)
				}

				if !bytes.Equal(images[i].Data, tt.images[i]) {
					t.Errorf("expected %q, got %q", tt.images[i], images[i])
				}
			}
		})
	}
}
