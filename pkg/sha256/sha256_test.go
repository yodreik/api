package sha256

import "testing"

func TestString(t *testing.T) {
	tt := []struct {
		name string
		data string
		want string
	}{
		{
			name: "Some message",
			data: "Buy more RAM, brah!",
			want: "2a5c11447c1161c0b82055653f5b520d1cc36204d2b66f3d557fe6839e3248ce",
		},
		{
			name: "Empty string",
			data: "",
			want: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := String(tc.data)
			if got != tc.want {
				t.Fatalf("unexpected result of sha256.String: got %v, want %v\n", got, tc.want)
			}
		})
	}
}
