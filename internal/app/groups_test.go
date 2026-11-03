package app_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"

	"github.com/Myraxion/wordweave/internal/app"
)

type group struct {
	ID        int64   `json:"id"`
	Type      string  `json:"type"`
	Note      string  `json:"note"`
	CreatedAt string  `json:"createdAt"`
	Members   []entry `json:"members"`
}

func collect(t *testing.T, s *httptest.Server, spelling, definition string) entry {
	t.Helper()
	body, err := json.Marshal(map[string]string{"spelling": spelling, "definition": definition})
	if err != nil {
		t.Fatal(err)
	}
	var got entry
	request(t, s, "POST", "/api/entries", string(body), http.StatusCreated, &got)
	return got
}

func groupsPath(id int64) string { return "/api/entries/" + strconv.FormatInt(id, 10) + "/groups" }

func createGroup(t *testing.T, s *httptest.Server, origin int64, kind, note string, ids ...int64) group {
	t.Helper()
	var got group
	request(t, s, "POST", groupsPath(origin), groupBody(t, kind, note, ids), http.StatusCreated, &got)
	return got
}

func groupBody(t *testing.T, kind, note string, ids []int64) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{"type": kind, "note": note, "memberIds": ids})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestCreateGroupAndReadFromEveryMember(t *testing.T) {
	dir := t.TempDir()
	a, err := app.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := httptest.NewServer(a)
	t.Cleanup(func() { s.Close(); a.Close() })
	draw := collect(t, s, "draw", "画\n拉")
	drew := collect(t, s, "Drew", "过去式")
	drawn := collect(t, s, "drawn", "")
	note := "draw 是原形\n<script>drawn 是过去分词</script>"
	got := createGroup(t, s, drew.ID, "屈折变化", note, drawn.ID, draw.ID, drew.ID, draw.ID)
	if got.ID <= 0 || got.Type != "屈折变化" || got.Note != note || got.CreatedAt == "" ||
		!reflect.DeepEqual(got.Members, []entry{draw, drawn, drew}) {
		t.Fatalf("unexpected group: %+v", got)
	}
	for _, member := range []entry{draw, drew, drawn} {
		var groups []group
		request(t, s, "GET", groupsPath(member.ID), "", http.StatusOK, &groups)
		if !reflect.DeepEqual(groups, []group{got}) {
			t.Fatalf("member %s sees %+v, want %+v", member.Spelling, groups, got)
		}
	}
	var detail group
	request(t, s, "GET", "/api/groups/"+strconv.FormatInt(got.ID, 10), "", http.StatusOK, &detail)
	if !reflect.DeepEqual(detail, got) {
		t.Fatalf("group detail changed: %+v", detail)
	}
	// Reopening the same on-disk notebook must retain the whole group.
	s.Close()
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	reopened := startApp(t, dir)
	var retained []group
	request(t, reopened, "GET", groupsPath(drawn.ID), "", http.StatusOK, &retained)
	if !reflect.DeepEqual(retained, []group{got}) {
		t.Fatalf("group was not persisted: %+v", retained)
	}
}

func TestDuplicateGroupPointsToOriginal(t *testing.T) {
	s := startApp(t, t.TempDir())
	a := collect(t, s, "retreat", "撤退")
	b := collect(t, s, "Withdraw", "撤回")
	c := collect(t, s, "retire", "退休")
	original := createGroup(t, s, a.ID, "近义", "原说明", b.ID, c.ID)
	var conflict struct {
		Error         string `json:"error"`
		ExistingGroup group  `json:"existingGroup"`
	}
	request(t, s, "POST", groupsPath(c.ID), groupBody(t, "近义", "不同说明", []int64{b.ID, a.ID}), http.StatusConflict, &conflict)
	if conflict.Error == "" || !reflect.DeepEqual(conflict.ExistingGroup, original) {
		t.Fatalf("conflict must link unchanged original: %+v", conflict)
	}
	var groups []group
	request(t, s, "GET", groupsPath(a.ID), "", http.StatusOK, &groups)
	if !reflect.DeepEqual(groups, []group{original}) {
		t.Fatalf("duplicate changed groups: %+v", groups)
	}
	// A different type and a proper subset remain independent.
	otherType := createGroup(t, s, b.ID, "语义相关", "", c.ID, a.ID)
	subset := createGroup(t, s, a.ID, "近义", "子集说明", b.ID)
	request(t, s, "GET", groupsPath(a.ID), "", http.StatusOK, &groups)
	if !reflect.DeepEqual(groups, []group{subset, original, otherType}) {
		t.Fatalf("independent groups lost boundaries or order: %+v", groups)
	}
}

func TestInvalidGroupCreatesNoPartialData(t *testing.T) {
	s := startApp(t, t.TempDir())
	a := collect(t, s, "quiet", "")
	b := collect(t, s, "quilt", "")
	var empty []group
	request(t, s, "GET", groupsPath(a.ID), "", http.StatusOK, &empty)
	if empty == nil || len(empty) != 0 {
		t.Fatalf("want [], got %+v", empty)
	}
	for _, tc := range []struct {
		name, kind string
		ids        []int64
	}{
		{"no selection", "近义", nil},
		{"only repeated origin", "近义", []int64{a.ID, a.ID}},
		{"unknown type", "自定义", []int64{b.ID}},
		{"empty type", "", []int64{b.ID}},
		{"nonpositive member", "形音相似", []int64{0, b.ID}},
		// The valid member is inserted before the missing one; all must roll back.
		{"missing member", "形音相似", []int64{b.ID, b.ID + 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request(t, s, "POST", groupsPath(a.ID), groupBody(t, tc.kind, "", tc.ids), http.StatusBadRequest, nil)
			for _, id := range []int64{a.ID, b.ID} {
				var got []group
				request(t, s, "GET", groupsPath(id), "", http.StatusOK, &got)
				if got == nil || len(got) != 0 {
					t.Fatalf("invalid group left partial data: %+v", got)
				}
			}
		})
	}
	c := collect(t, s, "Quite", "")
	got := createGroup(t, s, a.ID, "形音相似", "", b.ID, c.ID)
	if len(got.Members) != 3 {
		t.Fatalf("retry after invalid group failed: %+v", got)
	}
	request(t, s, "POST", groupsPath(99999), groupBody(t, "近义", "", []int64{a.ID}), http.StatusNotFound, nil)
	request(t, s, "GET", groupsPath(99999), "", http.StatusNotFound, nil)
	request(t, s, "GET", "/api/groups/99999", "", http.StatusNotFound, nil)
	request(t, s, "POST", groupsPath(a.ID), "{", http.StatusBadRequest, nil)
}

func TestGroupTypeOrderAndOverlappingBoundaries(t *testing.T) {
	s := startApp(t, t.TempDir())
	a := collect(t, s, "Apple", "苹果")
	b := collect(t, s, "banana", "香蕉")
	c := collect(t, s, "Cherry", "樱桃")
	d := collect(t, s, "date", "枣")
	wantTypes := []string{"形音相似", "近义", "反义", "词族", "屈折变化", "拼写变体", "语义相关"}
	var types []string
	request(t, s, "GET", "/api/group-types", "", http.StatusOK, &types)
	if !reflect.DeepEqual(types, wantTypes) {
		t.Fatalf("fixed types = %v", types)
	}
	want := make([]group, len(wantTypes))
	// Reverse creation order ensures fixed type order wins over creation time.
	for i := len(wantTypes) - 1; i >= 0; i-- {
		want[i] = createGroup(t, s, a.ID, wantTypes[i], wantTypes[i]+"说明", b.ID, c.ID)
	}
	overlap := createGroup(t, s, c.ID, "近义", "另一组的说明", d.ID)
	var got []group
	request(t, s, "GET", groupsPath(a.ID), "", http.StatusOK, &got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("types out of order or cross-group relation inferred: %+v", got)
	}
	request(t, s, "GET", groupsPath(c.ID), "", http.StatusOK, &got)
	wantC := append([]group{want[0], overlap}, want[1:]...)
	if !reflect.DeepEqual(got, wantC) {
		t.Fatalf("overlapping groups changed: %+v", got)
	}
	request(t, s, "GET", groupsPath(d.ID), "", http.StatusOK, &got)
	if !reflect.DeepEqual(got, []group{overlap}) {
		t.Fatalf("cross-group relations inferred: %+v", got)
	}
}

func TestEditedEntriesAppearInAllGroups(t *testing.T) {
	s := startApp(t, t.TempDir())
	a := collect(t, s, "dew", "露水")
	b := collect(t, s, "dewy", "")
	original := createGroup(t, s, a.ID, "词族", "共享说明", b.ID)
	var updated entry
	request(t, s, "PUT", "/api/entries/"+strconv.FormatInt(b.ID, 10),
		`{"spelling":"A dewy morning","definition":"带露水的清晨\n新释义"}`, http.StatusOK, &updated)
	var got []group
	request(t, s, "GET", groupsPath(a.ID), "", http.StatusOK, &got)
	original.Members = []entry{updated, a}
	if !reflect.DeepEqual(got, []group{original}) {
		t.Fatalf("group shows stale spelling, definition or sorting: %+v", got)
	}
}
