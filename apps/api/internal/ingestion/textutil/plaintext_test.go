package textutil

import "testing"

func TestPlainTextFromHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "raw html",
			input: "<p>Build payments infrastructure as a summer intern.</p>",
			want:  "Build payments infrastructure as a summer intern.",
		},
		{
			name:  "entity encoded greenhouse content",
			input: "&lt;p&gt;&lt;strong&gt;Team:&lt;/strong&gt; Apollo — Block Applied R&amp;amp;D&lt;br&gt;&lt;strong&gt;Location:&lt;/strong&gt; Remote&lt;/p&gt;",
			want:  "Team: Apollo — Block Applied R&D Location: Remote",
		},
		{
			name:  "br tags",
			input: "Line one<br>Line two",
			want:  "Line one Line two",
		},
		{
			name:  "empty",
			input: "   ",
			want:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := PlainTextFromHTML(tc.input)
			if got != tc.want {
				t.Errorf("PlainTextFromHTML() = %q, want %q", got, tc.want)
			}
		})
	}
}
