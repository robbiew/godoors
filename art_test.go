package godoors

import "testing"

func TestTruncateTextUsesRunesAndEllipsis(t *testing.T) {
	if got := TruncateText("你好世界", 2); got != "你好" {
		t.Fatalf("max=2 got %q", got)
	}
	if got := TruncateText("你好世界", 3); got != "..." {
		t.Fatalf("max=3 got %q", got)
	}
	if got := TruncateText("你好世界啊", 4); got != "你..." {
		t.Fatalf("max=4 got %q", got)
	}
	if got := TruncateText("abc", 3); got != "abc" {
		t.Fatalf("exact fit got %q", got)
	}
	if got := TruncateText("abc", 0); got != "" {
		t.Fatalf("max=0 got %q", got)
	}
}
