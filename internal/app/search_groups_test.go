package app_test

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

func TestSearchAndBrowseGroups(t *testing.T) {
	s := startApp(t, t.TempDir())
	a := collect(t, s, "alpha", "shared phrase")
	b := collect(t, s, "Beta", "")
	c := collect(t, s, "gamma", "")
	first := createGroup(t, s, a.ID, "近义", "Shared phrase %_", b.ID)
	second := createGroup(t, s, a.ID, "近义", "shared phrase other", b.ID, c.ID)
	empty := createGroup(t, s, b.ID, "反义", "", c.ID)
	laterType := createGroup(t, s, a.ID, "反义", "shared phrase", c.ID)
	for _, tc := range []struct {
		query   string
		entries []entry
		groups  []group
	}{
		{" SHARED PHRASE ", []entry{a}, []group{second, first, laterType}},
		{"%_", []entry{}, []group{first}},
		{"alpha", []entry{a}, []group{}},
		{"shared other", []entry{}, []group{}},
		{" ", []entry{c, b, a}, []group{}},
	} {
		t.Run(tc.query, func(t *testing.T) {
			var got struct {
				Entries []entry `json:"entries"`
				Groups  []group `json:"groups"`
			}
			request(t, s, "GET", "/api/search?q="+url.QueryEscape(tc.query), "", http.StatusOK, &got)
			if !reflect.DeepEqual(got.Entries, tc.entries) || !reflect.DeepEqual(got.Groups, tc.groups) {
				t.Fatalf("got %+v", got)
			}
		})
	}
	var groups []group
	request(t, s, "GET", "/api/groups?type="+url.QueryEscape("近义"), "", http.StatusOK, &groups)
	if !reflect.DeepEqual(groups, []group{second, first}) {
		t.Fatalf("groups: %+v", groups)
	}
	request(t, s, "GET", "/api/groups?type="+url.QueryEscape("反义"), "", http.StatusOK, &groups)
	if !reflect.DeepEqual(groups, []group{laterType, empty}) {
		t.Fatalf("groups: %+v", groups)
	}
}
