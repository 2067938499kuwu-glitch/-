package app

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
)

const publicCatalogPageLimit = 20

var (
	publicArticlePattern = regexp.MustCompile(`(?is)<article\b[^>]*>(.*?)</article>`)
	publicHrefPattern    = regexp.MustCompile(`(?is)href\s*=\s*["']([^"']+)["']`)
	publicImagePattern   = regexp.MustCompile(`(?is)(?:src|data-src)\s*=\s*["']([^"']+)["']`)
	publicTitlePattern   = regexp.MustCompile(`(?is)<h[1-6]\b[^>]*class=["'][^"']*(?:dm-card-title|title)[^"']*["'][^>]*>(.*?)</h[1-6]>`)
	publicDescription    = regexp.MustCompile(`(?is)<p\b[^>]*class=["'][^"']*(?:dm-card-sub|description|excerpt)[^"']*["'][^>]*>(.*?)</p>`)
	publicHeatPattern    = regexp.MustCompile(`(?is)class=["'][^"']*(?:dm-card-heat-text|heat)[^"']*["'][^>]*>(.*?)</`)
	publicTagPattern     = regexp.MustCompile(`(?is)<(?:a|span)\b[^>]*class=["'][^"']*(?:dm-tag|plot-tag|tag)[^"']*["'][^>]*>(.*?)</(?:a|span)>`)
	publicEpisodePattern = regexp.MustCompile(`(?i)(?:更新至|全)?\s*(\d+)\s*集`)
	publicTagStrip       = regexp.MustCompile(`(?is)<[^>]+>`)
	publicSpacePattern   = regexp.MustCompile(`\s+`)
)

func (d *Downloader) fetchHuangdouCatalog(ctx context.Context) ([]Drama, error) {
	return d.fetchPagedPublicCatalog(ctx, sourceHuangdou, "https://hddj.tv/", func(page int) string {
		if page <= 1 {
			return "https://hddj.tv/"
		}
		return fmt.Sprintf("https://hddj.tv/page/%d/", page)
	})
}

func (d *Downloader) fetchHuangguoCatalog(ctx context.Context) ([]Drama, error) {
	return d.fetchPagedPublicCatalog(ctx, sourceHuangguo, "https://huangguo.video/", func(page int) string {
		return fmt.Sprintf("https://huangguo.video/videos?page=%d", page)
	})
}

func (d *Downloader) fetchPagedPublicCatalog(ctx context.Context, source, base string, pageURL func(int) string) ([]Drama, error) {
	var all []Drama
	seen := map[string]bool{}
	for pageNumber := 1; pageNumber <= publicCatalogPageLimit; pageNumber++ {
		body, err := d.fetchPublicCatalogPage(ctx, pageURL(pageNumber), base)
		if err != nil {
			if len(all) > 0 {
				return all, err
			}
			return nil, err
		}
		items := parsePublicCatalog(source, base, body)
		added := 0
		for _, item := range items {
			if seen[item.ID] {
				continue
			}
			seen[item.ID] = true
			all = append(all, item)
			added++
		}
		if len(items) == 0 || added == 0 || !publicCatalogHasNext(body, pageNumber) {
			break
		}
	}
	if len(all) == 0 {
		return nil, errors.New("公开目录暂无可识别内容")
	}
	return all, nil
}

func (d *Downloader) fetchPublicCatalogPage(ctx context.Context, address, referer string) (string, error) {
	if d == nil || d.client == nil {
		return "", errors.New("站源客户端未初始化")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "text/html,application/xhtml+xml")
	request.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.7")
	request.Header.Set("Referer", referer)
	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/140 Safari/537.36")
	response, err := d.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("目录站返回 HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func parsePublicCatalog(source, base, body string) []Drama {
	baseURL, err := url.Parse(base)
	if err != nil {
		return nil
	}
	var rows []Drama
	for _, match := range publicArticlePattern.FindAllStringSubmatch(body, -1) {
		article := match[1]
		href := firstPublicMatch(publicHrefPattern, article)
		pageURL := resolvePublicURL(baseURL, href)
		if pageURL == "" || !publicDetailPath(source, href) {
			continue
		}
		title := cleanPublicText(firstPublicMatch(publicTitlePattern, article))
		if title == "" {
			title = cleanPublicText(publicAttribute(article, "aria-label"))
		}
		if title == "" {
			title = cleanPublicText(publicAttribute(article, "title"))
		}
		if title == "" {
			continue
		}
		sourceID := strings.TrimSuffix(path.Base(strings.TrimRight(href, "/")), path.Ext(href))
		if sourceID == "" || sourceID == "." || sourceID == "/" {
			continue
		}
		cover := resolvePublicURL(baseURL, firstPublicMatch(publicImagePattern, article))
		description := cleanPublicText(firstPublicMatch(publicDescription, article))
		heat := cleanPublicText(firstPublicMatch(publicHeatPattern, article))
		var tags []string
		remark := ""
		for _, tagMatch := range publicTagPattern.FindAllStringSubmatch(article, -1) {
			text := cleanPublicText(tagMatch[1])
			if text == "" {
				continue
			}
			if publicEpisodePattern.MatchString(text) {
				remark = text
				continue
			}
			if len(tags) < 12 {
				tags = append(tags, text)
			}
		}
		count := 0
		if episode := publicEpisodePattern.FindStringSubmatch(remark); len(episode) > 1 {
			count, _ = strconv.Atoi(episode[1])
		}
		category := "公开目录"
		if len(tags) > 0 {
			category = tags[0]
		}
		label := "黄果"
		if source == sourceHuangdou {
			label = "黄豆"
		}
		rows = append(rows, Drama{
			ID: providerDramaID(source, sourceID), Source: source, SourceID: sourceID,
			Title: title, Name: title, Desc: description, Intro: description,
			Cover: cover, CoverURL: cover, CategoryName: category, ChannelName: label,
			Remark: remark, TotalEpisode: count, EpisodeCount: count, Tags: tags,
			Heat: heat, PageURL: pageURL,
		})
	}
	return viewableDramas(rows)
}

func publicDetailPath(source, href string) bool {
	pathValue := strings.ToLower(strings.TrimSpace(href))
	if source == sourceHuangdou {
		return strings.Contains(pathValue, "/series/details/")
	}
	return strings.Contains(pathValue, "/video/") || strings.Contains(pathValue, "/series/") || strings.Contains(pathValue, "/photo/")
}

func publicCatalogHasNext(body string, pageNumber int) bool {
	next := fmt.Sprintf("/page/%d/", pageNumber+1)
	return strings.Contains(body, next) || strings.Contains(body, fmt.Sprintf("page=%d", pageNumber+1)) || strings.Contains(strings.ToLower(body), `rel="next"`) || strings.Contains(strings.ToLower(body), `rel='next'`)
}

func firstPublicMatch(pattern *regexp.Regexp, value string) string {
	match := pattern.FindStringSubmatch(value)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

func publicAttribute(value, name string) string {
	pattern := regexp.MustCompile(`(?is)` + regexp.QuoteMeta(name) + `\s*=\s*["']([^"']+)["']`)
	return firstPublicMatch(pattern, value)
}

func cleanPublicText(value string) string {
	value = publicTagStrip.ReplaceAllString(value, " ")
	value = html.UnescapeString(value)
	return strings.TrimSpace(publicSpacePattern.ReplaceAllString(value, " "))
}

func resolvePublicURL(base *url.URL, value string) string {
	value = strings.TrimSpace(html.UnescapeString(value))
	if value == "" || strings.HasPrefix(value, "data:") || strings.HasPrefix(value, "javascript:") {
		return ""
	}
	reference, err := url.Parse(value)
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(reference)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return ""
	}
	return resolved.String()
}
