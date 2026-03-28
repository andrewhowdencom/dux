package cli

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Shows the version of the built application",
	Long:  `Shows the version of the built application, based on Go's internal build primitives.`,
	Run: func(cmd *cobra.Command, args []string) {
		info, ok := debug.ReadBuildInfo()
		if !ok {
			fmt.Println("unknown")
			return
		}

		var revision string
		var time string
		var modified bool

		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = setting.Value
			case "vcs.time":
				time = setting.Value
			case "vcs.modified":
				modified = setting.Value == "true"
			}
		}

		if revision != "" && time != "" {
			// e.g. v0.0.0-20260126085723-f3472ac67d26
			// Format time as YYYYMMDDHHMMSS (remove -, T, :, Z)
			timeStr := strings.ReplaceAll(time, "-", "")
			timeStr = strings.ReplaceAll(timeStr, "T", "")
			timeStr = strings.ReplaceAll(timeStr, ":", "")
			timeStr = strings.ReplaceAll(timeStr, "Z", "")

			shortRev := revision
			if len(shortRev) > 12 {
				shortRev = shortRev[:12]
			}

			versionStr := fmt.Sprintf("v0.0.0-%s-%s", timeStr, shortRev)
			if modified {
				versionStr += "-dirty"
			}
			fmt.Println(versionStr)
			return
		}

		// Use the module version if it's set
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			fmt.Println(info.Main.Version)
			return
		}

		fmt.Println("(devel)")
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
