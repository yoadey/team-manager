package storage

import "testing"

func TestKeyBuilders(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"team photo", TeamPhotoKey("t1"), "teams/t1/photo"},
		{"team logo", TeamLogoKey("t1"), "teams/t1/logo"},
		{"user photo", UserPhotoKey("u1"), "users/u1/photo"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.got != c.want {
				t.Errorf("got %q, want %q", c.got, c.want)
			}
		})
	}
}
