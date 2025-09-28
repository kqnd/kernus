package details

import (
	"fmt"
	"strings"

	"github.com/kqnd/kernus/internal/models"
)

type StorageTab struct {
	formatter    *Formatter
	tableBuilder *TableBuilder
}

func NewStorageTab() *StorageTab {
	return &StorageTab{
		formatter:    NewFormatter(),
		tableBuilder: NewTableBuilder(),
	}
}

func (s *StorageTab) Render(container *models.Container) string {
	if container == nil {
		return ""
	}

	sections := []string{
		s.renderMounts(container),
		s.renderBlockIOStats(container),
	}

	return fmt.Sprintf("[yellow]Storage Configuration[white]\n\n%s", strings.Join(sections, "\n\n"))
}

func (s *StorageTab) renderMounts(c *models.Container) string {
	mountsTable := s.tableBuilder.BuildMountsTable(c.Mounts)
	return fmt.Sprintf("[yellow]Mounts (%d)[white]\n%s", len(c.Mounts), mountsTable)
}

func (s *StorageTab) renderBlockIOStats(c *models.Container) string {
	if c.Stats == nil {
		return "[yellow]Block I/O Statistics[white]\n  [gray]Block I/O statistics not available[white]"
	}

	return fmt.Sprintf(`[yellow]Block I/O Statistics[white]
  Read     : %s (%s operations)
  Write    : %s (%s operations)`,
		s.formatter.FormatBytes(c.Stats.BlockIO.ReadBytes),
		s.formatter.FormatNumber(c.Stats.BlockIO.ReadOps),
		s.formatter.FormatBytes(c.Stats.BlockIO.WriteBytes),
		s.formatter.FormatNumber(c.Stats.BlockIO.WriteOps))
}
