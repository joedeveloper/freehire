package telegram

import "testing"

func TestTextToHTML(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{
			name: "paragraphs and line breaks",
			in:   "Требования:\n- Go\n- Postgres\n\nПишите @hr",
			want: "<p>Требования:<br>- Go<br>- Postgres</p><p>Пишите @hr</p>",
		},
		{
			name: "markup is escaped, not interpreted",
			in:   "Stack: C++ & <script>alert(1)</script>",
			want: "<p>Stack: C++ &amp; &lt;script&gt;alert(1)&lt;/script&gt;</p>",
		},
		{
			name: "surrounding whitespace trimmed",
			in:   "\n\n  hello  \n\n",
			want: "<p>hello</p>",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := TextToHTML(tc.in); got != tc.want {
				t.Errorf("TextToHTML(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
