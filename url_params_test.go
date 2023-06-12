package api2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClassifier(t *testing.T) {
	cl := newPathClassifier([]string{
		"/users",
		"/users/:user",
		"/users/:user/posts",
		"/users/:user/posts/:post",
		"/users/:user/posts/:post/comments",
		"/users/:user/comments",
		"/users/:user/comments/:comment",
		"/users/:user/comments/:comment/responses",
	})

	cases := []struct {
		url             string
		wantIndex       int
		wantParam2value map[string]string
	}{
		{
			url:             "/users",
			wantIndex:       0,
			wantParam2value: map[string]string{},
		},
		{
			url:       "/user",
			wantIndex: -1,
		},
		{
			url:       "/users2",
			wantIndex: -1,
		},
		{
			url:       "/users/123",
			wantIndex: 1,
			wantParam2value: map[string]string{
				"user": "123",
			},
		},
		{
			url:       "/users/123/",
			wantIndex: 1,
			wantParam2value: map[string]string{
				"user": "123",
			},
		},
		{
			url:       "/users/123/test",
			wantIndex: -1,
		},
		{
			url:       "/users/123/posts",
			wantIndex: 2,
			wantParam2value: map[string]string{
				"user": "123",
			},
		},
		{
			url:       "/users/123/posts/456-789",
			wantIndex: 3,
			wantParam2value: map[string]string{
				"user": "123",
				"post": "456-789",
			},
		},
		{
			url:       "/users/123/posts/456-789/",
			wantIndex: 3,
			wantParam2value: map[string]string{
				"user": "123",
				"post": "456-789",
			},
		},
		{
			url:       "/users/123/posts/456-789/comments",
			wantIndex: 4,
			wantParam2value: map[string]string{
				"user": "123",
				"post": "456-789",
			},
		},
		{
			url:       "/users/123/posts/456-789/comments/",
			wantIndex: 4,
			wantParam2value: map[string]string{
				"user": "123",
				"post": "456-789",
			},
		},
		{
			url:       "/users/123/comments",
			wantIndex: 5,
			wantParam2value: map[string]string{
				"user": "123",
			},
		},
		{
			url:       "/users/123/comments/",
			wantIndex: 5,
			wantParam2value: map[string]string{
				"user": "123",
			},
		},
		{
			url:       "/users/123/comments/ab505",
			wantIndex: 6,
			wantParam2value: map[string]string{
				"user":    "123",
				"comment": "ab505",
			},
		},
		{
			url:       "/users/123/comments/ab505/",
			wantIndex: 6,
			wantParam2value: map[string]string{
				"user":    "123",
				"comment": "ab505",
			},
		},
		{
			url:       "/users/123/comments/ab505/responses",
			wantIndex: 7,
			wantParam2value: map[string]string{
				"user":    "123",
				"comment": "ab505",
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.url, func(t *testing.T) {
			gotIndex, gotParam2value := cl.Classify(tc.url)
			require.Equal(t, tc.wantIndex, gotIndex)
			require.Equal(t, tc.wantParam2value, gotParam2value)
		})
	}
}

func TestSplitUrl(t *testing.T) {
	cases := []struct {
		url  string
		want []string
	}{
		{
			url:  "/foo",
			want: []string{"foo"},
		},
		{
			url:  "//foo",
			want: []string{"foo"},
		},
		{
			url:  "/foo/bar",
			want: []string{"foo", "bar"},
		},
		{
			url:  "/foo/bar/",
			want: []string{"foo", "bar"},
		},
		{
			url:  "/foo/bar//",
			want: []string{"foo", "bar"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.url, func(t *testing.T) {
			got := splitUrl(tc.url)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestFindUrlKeys(t *testing.T) {
	cases := []struct {
		mask string
		want []string
	}{
		{
			mask: "/",
			want: []string{},
		},
		{
			mask: "/foo",
			want: []string{},
		},
		{
			mask: "/:foo",
			want: []string{"foo"},
		},
		{
			mask: "/:foo/",
			want: []string{"foo"},
		},
		{
			mask: "/bar/:foo/zoo/:baz",
			want: []string{"foo", "baz"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.mask, func(t *testing.T) {
			got := findUrlKeys(tc.mask)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestCutUrlParams(t *testing.T) {
	cases := []struct {
		mask string
		want string
	}{
		{
			mask: "/",
			want: "/",
		},
		{
			mask: "/foo",
			want: "/foo",
		},
		{
			mask: "/bar/:foo",
			want: "/bar/",
		},
		{
			mask: "/bar/:foo/",
			want: "/bar/",
		},
		{
			mask: "/:foo",
			want: "/",
		},
		{
			mask: "/:foo/",
			want: "/",
		},
		{
			mask: "/bar/:foo/zoo/:baz",
			want: "/bar/",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.mask, func(t *testing.T) {
			got := cutUrlParams(tc.mask)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestBuildUrl(t *testing.T) {
	cases := []struct {
		name        string
		mask        string
		param2value map[string]string
		want        string
		wantError   bool
	}{
		{
			name:        "no parameters",
			mask:        "/path",
			param2value: map[string]string{},
			want:        "/path",
		},
		{
			name: "single parameter",
			mask: "/path/:id",
			param2value: map[string]string{
				"id": "123",
			},
			want: "/path/123",
		},
		{
			name: "two parameters",
			mask: "/path/:id/comment/:comment",
			param2value: map[string]string{
				"id":      "123",
				"comment": "444",
			},
			want: "/path/123/comment/444",
		},
		{
			name: "missing parameter in param2value",
			mask: "/path/:id/comment/:comment",
			param2value: map[string]string{
				"comment": "444",
			},
			wantError: true,
		},
		{
			name: "extra parameter in param2value",
			mask: "/path/:id/comment/:comment",
			param2value: map[string]string{
				"id":      "123",
				"comment": "444",
				"user":    "555",
			},
			wantError: true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildUrl(tc.mask, tc.param2value)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			}
		})
	}
}
