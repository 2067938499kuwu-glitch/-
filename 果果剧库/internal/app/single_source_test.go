package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCacheShowsAllSourcesAndKeepsHongguoUpdatesSafe(t *testing.T) {
	old := Drama{ID: "hongguo:700001", Source: sourceHongguo, Title: "旧名称", OnlineDate: "2026-03-16", Views: "0", Heat: "20"}
	cache := libraryCache{Dramas: []Drama{
		old, {ID: "700002", Title: "保留旧剧"},
		{ID: "huangdou:700003", Source: "huangdou", Title: "其他来源"},
		{ID: "hongguo:700004", Source: "huangguoai", Title: "冲突来源"},
	}, Sources: map[string]librarySourceState{
		sourceHongguo: {Status: "ready"}, "huangdou": {Status: "failed", Error: "不应恢复"},
	}, LastError: "不应恢复"}
	directory := t.TempDir()
	if err := writeLibraryCache(directory, cache); err != nil {
		t.Fatal(err)
	}
	restored, err := readLibraryCache(directory)
	if err != nil || len(restored.Dramas) != 3 || len(restored.Sources) != 2 {
		t.Fatalf("did not restore all viewable cached resources: %+v, %v", restored, err)
	}
	fresh := []Drama{
		{ID: old.ID, Source: sourceHongguo, Title: "新名称", Heat: "30"},
		{ID: "hongguo:700005", Source: sourceHongguo, Title: "新增剧"},
		{ID: "huangguoai:700006", Source: "huangguoai", Title: "排除其他来源"},
	}
	merged := mergeSourceDramas(restored.Dramas, fresh, nil, sourceHongguo)
	if len(merged) != 4 {
		t.Fatalf("incremental merge replaced cached resources or imported unsupported fresh data: %d", len(merged))
	}
	foundForeign := false
	for _, drama := range merged {
		if drama.ID == "huangdou:700003" {
			foundForeign = true
		}
		if drama.ID == old.ID && (drama.Title != "新名称" || drama.Heat != "30" || drama.Views != "0" || drama.OnlineDate != old.OnlineDate) {
			t.Fatal("fresh primary fields erased existing sort metadata")
		}
	}
	if !foundForeign {
		t.Fatal("cached non-Hongguo resource was hidden")
	}
}

func TestCatalogRequestBoundariesAndHongguoBoards(t *testing.T) {
	app := &UIApp{dramas: []Drama{{ID: "hongguo:700001", Source: sourceHongguo, Title: "cached"}}}
	for _, path := range []string{
		"/api/ui/dramas?source=unknown",
		"/api/ui/dramas?update=1&priority=huangdou:700001", "/api/ui/dramas?update=1&priority=hongguo:invalid",
		"/api/ui/dramas?more=1&priority=huangdou:700001", "/api/ui/dramas?more=1&priority=hongguo:invalid",
	} {
		writer := httptest.NewRecorder()
		app.handleDramas(writer, httptest.NewRequest("GET", path, nil))
		if writer.Code != 400 {
			t.Fatalf("unsupported source accepted: %s, %d", path, writer.Code)
		}
	}
	writer := httptest.NewRecorder()
	app.handleDramas(writer, httptest.NewRequest("GET", "/api/ui/dramas?source=huangdou", nil))
	if writer.Code != 200 {
		t.Fatalf("public catalog source rejected: %d", writer.Code)
	}
	writer = httptest.NewRecorder()
	app.handleRankings(writer, httptest.NewRequest("GET", "/api/ui/rankings", nil))
	var catalog struct {
		Boards []rankingBoard `json:"boards"`
	}
	if json.Unmarshal(writer.Body.Bytes(), &catalog) != nil || len(catalog.Boards) != 4 {
		t.Fatal("expected the four actual Hongguo boards")
	}
	for _, board := range catalog.Boards {
		if board.Source != sourceHongguo {
			t.Fatal("another source's board is exposed")
		}
	}
	for _, id := range []string{"huangdou-all", "huangguo-hot", "hongguo-daily", "hongguo-weekly", "hongguo-monthly"} {
		writer := httptest.NewRecorder()
		app.handleRankings(writer, httptest.NewRequest("GET", "/api/ui/rankings?board="+id, nil))
		if writer.Code != 400 {
			t.Fatalf("unsupported or fabricated board accepted: %s", id)
		}
	}
	for _, handler := range []http.HandlerFunc{app.handleDownload, app.handleUpdate} {
		writer := httptest.NewRecorder()
		handler(writer, httptest.NewRequest("POST", "/api/ui/download", strings.NewReader(`{"ids":["hongguo:700001","huangdou:700002"]}`)))
		if writer.Code != 400 {
			t.Fatal("mixed-source task request accepted")
		}
	}
	if _, _, err := (&Downloader{}).GetDramaChapters(context.Background(), "huangguoai:700001"); err == nil {
		t.Fatal("playback accepted an unsupported source")
	}
}

func TestHongguoSearchAndRankingKeepPublicCatalogDataSeparated(t *testing.T) {
	d := rankingTestDownloader(t, func(request *http.Request) (*http.Response, error) {
		t.Errorf("fixture should not request an upstream resource: %s", request.URL.Path)
		return rankingHTTPResponse(request, 404, ""), nil
	})
	checked := &sortMetadataState{Version: sortMetadataVersion, CheckedAt: time.Now(), CoverChecked: true}
	good := Drama{ID: "hongguo:700001", Source: sourceHongguo, Title: "红果搜索结果", SortMetadata: checked}
	foreign := Drama{ID: "huangdou:700002", Source: "huangdou", Title: "应排除"}
	conflict := Drama{ID: "huangguoai:700003", Source: sourceHongguo, Title: "应排除冲突 ID"}
	d.hongguoClient().searches["测试"] = hongguoSearchEntry{Dramas: []Drama{good, foreign, conflict}, Total: 3, ExpiresAt: time.Now().Add(time.Minute)}
	app := &UIApp{downloader: d, cfg: d.cfg}
	t.Cleanup(app.stopSortMetadata)
	writer := httptest.NewRecorder()
	app.handleLibrarySearch(writer, httptest.NewRequest("GET", "/api/ui/search?q=测试", nil))
	var result struct {
		Data []Drama `json:"data"`
	}
	if writer.Code != 200 || json.Unmarshal(writer.Body.Bytes(), &result) != nil || len(result.Data) != 1 || len(app.dramas) != 1 {
		t.Fatal("search registered unsupported data")
	}
	app.acceptLibraryProgress("huangdou", []Drama{foreign}, nil, true)
	app.acceptRankingDramas(rankingPage{BoardID: "hongguo-hot", Page: 1, FetchedAt: time.Now(), Items: []rankingItem{
		{Rank: 1, Drama: Drama{ID: "hongguo:700004", Source: sourceHongguo, Title: "红果榜单结果", SortMetadata: checked}},
		{Rank: 2, Drama: foreign}, {Rank: 3, Drama: conflict},
	}})
	if len(app.dramas) != 3 || len(app.librarySources) != 2 {
		t.Fatal("catalog progress or ranking registration lost expected data")
	}
	for _, drama := range app.dramas {
		if drama.ID == foreign.ID {
			if dramaProvider(drama) != sourceHuangdou {
				t.Fatal("public catalog source was not preserved")
			}
			continue
		}
		if _, valid := normalizeHongguoDrama(drama); !valid {
			t.Fatal("invalid ranking drama was registered")
		}
	}
}
