package app_test

import (
	"net/http"
	"net/url"
	"reflect"
	"strconv"
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
	for _, path := range []string{"/api/groups", "/api/groups?type="} {
		request(t, s, "GET", path, "", http.StatusOK, &groups)
		if !reflect.DeepEqual(groups, []group{laterType, empty, second, first}) {
			t.Fatalf("all groups at %s: %+v", path, groups)
		}
	}
	request(t, s, "GET", "/api/groups?type="+url.QueryEscape("近义"), "", http.StatusOK, &groups)
	if !reflect.DeepEqual(groups, []group{second, first}) {
		t.Fatalf("groups: %+v", groups)
	}
	request(t, s, "GET", "/api/groups?type="+url.QueryEscape("反义"), "", http.StatusOK, &groups)
	if !reflect.DeepEqual(groups, []group{laterType, empty}) {
		t.Fatalf("groups: %+v", groups)
	}
}

func TestBrowseGroupsDefaultsToRecentModification(t *testing.T) {
	s := startApp(t, t.TempDir())
	a := collect(t, s, "alpha", "")
	b := collect(t, s, "beta", "")
	first := createGroup(t, s, a.ID, "近义", "旧说明", b.ID)
	second := createGroup(t, s, a.ID, "反义", "另一组", b.ID)
	request(t, s, "PUT", "/api/groups/"+strconv.FormatInt(first.ID, 10),
		groupBody(t, "近义", "更新说明", []int64{a.ID, b.ID}), http.StatusOK, nil)
	var got []group
	request(t, s, "GET", "/api/groups", "", http.StatusOK, &got)
	if len(got) != 2 || got[0].ID != first.ID || got[1].ID != second.ID {
		t.Fatalf("want recently modified group first, got %+v", got)
	}
}

func TestBrowseGroupsTimeSortsAndTypeFilter(t *testing.T) {
	dir := t.TempDir()
	s := startApp(t, dir)
	a := collect(t, s, "alpha", "")
	b := collect(t, s, "beta", "")
	c := collect(t, s, "gamma", "")
	db := fixtureDB(t, dir, "")
	for i, stamp := range []any{
		"2020-01-02T00:00:00Z", "2020-01-02T00:00:00.1Z",
		"2020-01-02T00:00:00.11Z", "2020-01-02T00:00:00.110000001Z",
		"2020-01-02T00:00:00.11Z", nil,
	} {
		kind := []string{"形音相似", "近义", "反义", "词族", "屈折变化", "近义"}[i]
		members := []int64{b.ID}
		if i == 5 {
			members = append(members, c.ID)
		}
		g := createGroup(t, s, a.ID, kind, "", members...)
		created := "2020-01-01T00:00:00Z"
		if stamp == nil {
			created = "2020-01-02T00:00:00.105Z"
		}
		if _, err := db.Exec("UPDATE relation_groups SET created_at = ?, updated_at = ? WHERE id = ?", created, stamp, g.ID); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		sort           string
		want, filtered []int64
	}{
		{"updated_asc", []int64{1, 2, 6, 3, 5, 4}, []int64{2, 6}},
		{"created_desc", []int64{6, 5, 4, 3, 2, 1}, []int64{6, 2}},
		{"created_asc", []int64{1, 2, 3, 4, 5, 6}, []int64{2, 6}},
		{"updated_desc", []int64{4, 5, 3, 6, 2, 1}, []int64{6, 2}},
		{"unknown", []int64{4, 5, 3, 6, 2, 1}, []int64{6, 2}},
		{"", []int64{4, 5, 3, 6, 2, 1}, []int64{6, 2}},
	} {
		t.Run(tc.sort, func(t *testing.T) {
			for _, kind := range []string{"", "近义"} {
				var got []group
				request(t, s, "GET", "/api/groups?"+url.Values{"sort": {tc.sort}, "type": {kind}}.Encode(), "", http.StatusOK, &got)
				var ids []int64
				for _, g := range got {
					ids = append(ids, g.ID)
					if g.ID == 6 && g.UpdatedAt != "" {
						t.Fatal("sorting must preserve the unknown legacy modification time")
					}
				}
				want := tc.want
				if kind != "" {
					want = tc.filtered
				}
				if !reflect.DeepEqual(ids, want) {
					t.Fatalf("type %q: got %v, want %v", kind, ids, want)
				}
				var paged []group
				for offset := 0; offset < len(got); offset += 2 {
					var page []group
					query := url.Values{"sort": {tc.sort}, "type": {kind}, "limit": {"2"}, "offset": {strconv.Itoa(offset)}}
					request(t, s, "GET", "/api/groups?"+query.Encode(), "", http.StatusOK, &page)
					paged = append(paged, page...)
				}
				if !reflect.DeepEqual(paged, got) {
					t.Fatalf("type %q: paginated groups differ: got %+v, want %+v", kind, paged, got)
				}
			}
		})
	}
}
