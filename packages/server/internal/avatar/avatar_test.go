package avatar

import "testing"

func TestURL(t *testing.T) {
	tests := []struct{ name, email, want string }{
		{"qq", "12345@qq.com", "https://q.qlogo.cn/headimg_dl?dst_uin=12345&spec=100"},
		{"normalized qq", " 12345@QQ.COM ", "https://q.qlogo.cn/headimg_dl?dst_uin=12345&spec=100"},
		{"gravatar", " Test@Example.com ", "https://www.gravatar.com/avatar/55502f40dc8b7c769880b10874abc9d0?d=identicon&s=80"},
		{"missing", "", ""}, {"blank", "  ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := URL(tt.email, 80); got != tt.want {
				t.Fatalf("URL(%q) = %q, want %q", tt.email, got, tt.want)
			}
		})
	}
}
