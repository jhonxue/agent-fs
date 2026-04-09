package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Version information (set via ldflags during build)
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"

	rootCmd = &cobra.Command{
		Use:           `afs`,
		Short:         `Agent-first cross-platform file operations CLI`,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig()
		},
	}

	configOnce sync.Once
	configErr  error
)

func Execute() error {
	return rootCmd.Execute()
}

func SetVersion(v, c, d string) {
	Version = v
	Commit = c
	BuildDate = d
}

func init() {
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   `version`,
	Short: `Show version information`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("afs version %s\n", Version)
		if Commit != "unknown" {
			fmt.Printf("commit: %s\n", Commit)
		}
		if BuildDate != "unknown" {
			fmt.Printf("built at: %s\n", BuildDate)
		}
		return nil
	},
}

func initConfig() error {
	configOnce.Do(func() {
		viper.SetEnvPrefix(`AFS`)
		viper.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`))
		viper.AutomaticEnv()

		viper.SetConfigName(`.agent-fs`)
		viper.SetConfigType(`yaml`)
		viper.AddConfigPath(`.`)

		if home, err := os.UserHomeDir(); err == nil && home != `` {
			viper.AddConfigPath(home)
		}

		err := viper.ReadInConfig()
		if err == nil {
			return
		}

		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) {
			return
		}
		configErr = err
	})
	return configErr
}
