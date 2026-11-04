package app_test

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"
)

func TestNotebookTotalsCountGroupsAndFollowDeletions(t *testing.T) {
	s := startApp(t, t.TempDir())
	assertTotals := func(entries, groups int) {
		t.Helper()
		var got struct {
			Entries int `json:"entries"`
			Groups  int `json:"groups"`
		}
		// Browse filters and search terms must not change notebook totals.
		request(t, s, "GET", "/api/stats?"+url.Values{"type": {"近义"}, "q": {"missing"}}.Encode(), "", http.StatusOK, &got)
		if got.Entries != entries || got.Groups != groups {
			t.Fatalf("totals = %+v, want %d entries and %d groups", got, entries, groups)
		}
	}
	assertTotals(0, 0)
	a := collect(t, s, "draw", "画")
	b := collect(t, s, "drew", "过去式")
	c := collect(t, s, "drawn", "过去分词")
	assertTotals(3, 0)
	triple := createGroup(t, s, a.ID, "屈折变化", "", b.ID, c.ID)
	createGroup(t, s, a.ID, "语义相关", "", b.ID)
	assertTotals(3, 2)
	request(t, s, "POST", groupsPath(c.ID), groupBody(t, "屈折变化", "", []int64{a.ID, b.ID}), http.StatusConflict, nil)
	assertTotals(3, 2)
	// Removing a member deletes the two-member group and keeps the three-member one.
	request(t, s, "DELETE", "/api/entries/"+strconv.FormatInt(b.ID, 10), "", http.StatusOK, nil)
	assertTotals(2, 1)
	request(t, s, "DELETE", "/api/groups/"+strconv.FormatInt(triple.ID, 10), "", http.StatusOK, nil)
	assertTotals(2, 0)
}
