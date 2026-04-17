package api

import (
	"strings"
	"time"
	"veda-anchor-engine/src/internal/data/repository"
)

// --- Types ---

type AppLeaderboardItem struct {
	Rank           int    `json:"rank"`
	Name           string `json:"name"`        // Display name (commercial name if available)
	ProcessName    string `json:"processName"` // Actual process name for blocking
	ExecutablePath string `json:"executablePath"`
	Icon           string `json:"icon"`
	Count          int    `json:"count"`
}

type WebLeaderboardItem struct {
	Rank   int    `json:"rank"`
	Domain string `json:"domain"`
	Title  string `json:"title"`
	Icon   string `json:"icon"`
	Count  int    `json:"count"`
}

type ScreenTimeItem struct {
	Name            string `json:"name"`
	ExecutablePath  string `json:"executablePath"`
	Icon            string `json:"icon"`
	DurationSeconds int    `json:"durationSeconds"`
}

// --- App Usage ---

func (s *Server) GetAppLeaderboard(since, until string) ([]AppLeaderboardItem, error) {
	sinceTime, _ := repository.ParseTime(since)
	untilTime, _ := repository.ParseTime(until)

	records, err := s.Apps.GetUsageRanking(sinceTime, untilTime)
	if err != nil {
		return nil, err
	}

	leaderboard := make([]AppLeaderboardItem, 0, len(records))
	for i, r := range records {
		leaderboard = append(leaderboard, AppLeaderboardItem{
			Rank:           i + 1,
			Name:           r.ProcessName, // Will be enriched by agent
			ProcessName:    r.ProcessName,
			ExecutablePath: r.ExecutablePath,
			Icon:           "", // Will be enriched by agent
			Count:          r.Count,
		})
	}
	return leaderboard, nil
}

func (s *Server) GetScreenTime() ([]ScreenTimeItem, error) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()

	records, err := s.Apps.GetScreenTimeTotals(todayStart)
	if err != nil {
		return nil, err
	}

	items := make([]ScreenTimeItem, 0, len(records))
	for _, r := range records {
		items = append(items, ScreenTimeItem{
			Name:            extractFileName(r.ExecutablePath), // Will be enriched by agent
			ExecutablePath:  r.ExecutablePath,
			Icon:            "", // Will be enriched by agent
			DurationSeconds: r.DurationSeconds,
		})
	}
	return items, nil
}

func (s *Server) GetTotalScreenTime() (int, error) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	return s.Apps.GetTotalDayScreenTime(todayStart)
}

// --- Web Usage ---

func (s *Server) GetWebLeaderboard(since, until string) ([]WebLeaderboardItem, error) {
	sinceTime, _ := repository.ParseTime(since)
	untilTime, _ := repository.ParseTime(until)

	records, err := s.Web.GetUsageRanking(sinceTime, untilTime)
	if err != nil {
		return nil, err
	}

	leaderboard := make([]WebLeaderboardItem, 0, len(records))
	for i, r := range records {
		item := WebLeaderboardItem{
			Rank:   i + 1,
			Domain: r.Domain,
			Count:  r.Count,
		}
		if meta, err := s.Web.GetMetadata(r.Domain); err == nil && meta != nil {
			item.Title = meta.Title
			item.Icon = meta.IconURL
		}
		leaderboard = append(leaderboard, item)
	}
	return leaderboard, nil
}

// --- Logs & Search ---

func (s *Server) Search(queryStr, since, until string) ([][]string, error) {
	return s.Apps.SearchEvents(strings.ToLower(queryStr), since, until)
}

func (s *Server) GetWebLogs(queryStr, since, until string) ([][]string, error) {
	return s.Web.GetLogs(queryStr, since, until)
}

func (s *Server) LogWebEvent(urlStr string) error {
	s.Web.LogWebEvent(urlStr)
	return nil
}

func (s *Server) SaveWebMetadata(domain, title, iconURL string) error {
	return s.Web.SaveMetadata(domain, title, iconURL)
}

// --- Utils ---

func extractFileName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '\\' || path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

// --- Agent Communication ---

func (s *Server) UpdateScreenTime(pid uint32, seconds int64) error {
	return s.Apps.UpdateScreenTimeByPID(pid, seconds)
}

func (s *Server) ReportActiveApp(pid uint32, exePath string) error {
	return s.Apps.ReportActiveApp(pid, exePath)
}
