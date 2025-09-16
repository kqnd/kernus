package details

import (
	"fmt"
	"time"

	"github.com/kern/internal/models"
)

type StatsTab struct {
	formatter     *Formatter
	visualizer    *StatsVisualizer
	lastStats     *models.ContainerStats
	cachedContent string
	lastUpdate    time.Time
}

func NewStatsTab() *StatsTab {
	return &StatsTab{
		formatter:  NewFormatter(),
		visualizer: NewStatsVisualizer(),
	}
}

func (s *StatsTab) Render(container *models.Container) string {
	if container == nil || container.Stats == nil {
		return s.renderNoStats()
	}

	if s.shouldUseCache(container.Stats) {
		return s.cachedContent
	}

	content := s.renderStats(container.Stats)
	s.updateCache(container.Stats, content)
	return content
}

func (s *StatsTab) shouldUseCache(stats *models.ContainerStats) bool {
	return s.lastStats != nil &&
		time.Since(s.lastUpdate) < 2*time.Second &&
		s.cachedContent != ""
}

func (s *StatsTab) updateCache(stats *models.ContainerStats, content string) {
	s.lastStats = stats
	s.cachedContent = content
	s.lastUpdate = time.Now()
}

func (s *StatsTab) renderNoStats() string {
	return fmt.Sprintf(`[yellow]Resource Statistics[white]

[red]No Statistics Available[white]

Statistics are only available for running containers.
%s`, s.visualizer.BuildStatsPlaceholder())
}

func (s *StatsTab) renderStats(stats *models.ContainerStats) string {
	return fmt.Sprintf(`[yellow]Resource Statistics[white]

[yellow]CPU Performance[white]
%s

[yellow]Memory Usage[white]
%s

[yellow]Network Activity[white]
%s

[yellow]Storage I/O[white]
%s

[yellow]Process Information[white]
  Active PIDs: [cyan]%d[white] processes

[gray]Last Updated: %s[white]`,
		s.visualizer.BuildCPUVisualization(stats.CPU),
		s.visualizer.BuildMemoryVisualization(stats.Memory),
		s.visualizer.BuildNetworkVisualization(stats.Network),
		s.visualizer.BuildBlockIOVisualization(stats.BlockIO),
		stats.PIDs,
		s.formatter.FormatTime(stats.Timestamp))
}
