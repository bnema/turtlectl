package cmd

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/bnema/turtlectl/internal/launcher"
	uilauncher "github.com/bnema/turtlectl/internal/ui/launcher"
)

var installCmd = &cobra.Command{
	Use:     "install",
	Aliases: []string{"i"},
	Short:   "Install/update AppImage and create desktop file",
	RunE: func(cmd *cobra.Command, args []string) error {
		l := launcher.New(getLogger())

		m := uilauncher.NewInstallModel(l)
		p := tea.NewProgram(m)

		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		fm := finalModel.(uilauncher.InstallModel)
		return fm.GetError()
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
