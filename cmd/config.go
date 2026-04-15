package cmd

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jhonxue/agent-fs/pkg/apperr"
	"github.com/jhonxue/agent-fs/pkg/output"
)

var configGlobal bool

var configCmd = &cobra.Command{
	Use:   `config`,
	Short: `Configuration management`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var configSetCmd = &cobra.Command{
	Use:   `set <key> <value>`,
	Short: `Set configuration key`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigSet(args[0], args[1])
	},
}

var configGetCmd = &cobra.Command{
	Use:   `get <key>`,
	Short: `Get configuration key`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigGet(args[0])
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)

	configSetCmd.Flags().BoolVar(&configGlobal, `global`, false, `Use ~/.agent-fs.yaml`)
	configGetCmd.Flags().BoolVar(&configGlobal, `global`, false, `Use ~/.agent-fs.yaml`)
}

func runConfigSet(key, rawValue string) error {
	key = strings.TrimSpace(key)
	if key == `` {
		return apperr.New(`config_set`, apperr.CodeInvalidArg, `key is required`)
	}

	cfgPath, err := configPath(configGlobal)
	if err != nil {
		return err
	}
	cfg, err := openConfigFile(cfgPath)
	if err != nil {
		return err
	}
	cfg.Set(key, parseValue(rawValue))
	cfg.SetConfigFile(cfgPath)
	if err := cfg.WriteConfigAs(cfgPath); err != nil {
		return apperr.Wrap(`config_set`, apperr.CodeConfig, `failed to write config file`, err)
	}

	return output.PrintSuccess(`config_set`, map[string]any{
		"config_file": cfgPath,
		"key":         key,
		"value":       cfg.Get(key),
	})
}

func runConfigGet(key string) error {
	key = strings.TrimSpace(key)
	if key == `` {
		return apperr.New(`config_get`, apperr.CodeInvalidArg, `key is required`)
	}

	cfgPath, err := configPath(configGlobal)
	if err != nil {
		return err
	}
	cfg, err := openConfigFile(cfgPath)
	if err != nil {
		return err
	}
	if !cfg.IsSet(key) {
		return apperr.New(`config_get`, apperr.CodeNotFound, `config key not found`)
	}

	return output.PrintSuccess(`config_get`, map[string]any{
		"config_file": cfgPath,
		"key":         key,
		"value":       cfg.Get(key),
	})
}

func configPath(global bool) (string, error) {
	if global {
		home, err := os.UserHomeDir()
		if err != nil {
			return ``, apperr.Wrap(`config`, apperr.CodeConfig, `failed to resolve home directory`, err)
		}
		return filepath.Join(home, `.agent-fs.yaml`), nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return ``, apperr.Wrap(`config`, apperr.CodeConfig, `failed to resolve current directory`, err)
	}
	return filepath.Join(cwd, `.agent-fs.yaml`), nil
}

func openConfigFile(cfgPath string) (*viper.Viper, error) {
	cfg := viper.New()
	cfg.SetConfigFile(cfgPath)
	cfg.SetConfigType(`yaml`)

	if _, err := os.Stat(cfgPath); err == nil {
		if err := cfg.ReadInConfig(); err != nil {
			return nil, apperr.Wrap(`config`, apperr.CodeConfig, `failed to read config file`, err)
		}
		return cfg, nil
	}

	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return nil, apperr.Wrap(`config`, apperr.CodeConfig, `failed to create config directory`, err)
	}
	if err := os.WriteFile(cfgPath, []byte{}, 0o600); err != nil {
		return nil, apperr.Wrap(`config`, apperr.CodeConfig, `failed to initialize config file`, err)
	}
	return cfg, nil
}

func parseValue(input string) any {
	value := strings.TrimSpace(input)
	if parsedBool, err := strconv.ParseBool(value); err == nil {
		return parsedBool
	}
	if parsedInt, err := strconv.ParseInt(value, 10, 64); err == nil {
		return parsedInt
	}
	if parsedFloat, err := strconv.ParseFloat(value, 64); err == nil {
		return parsedFloat
	}
	return input
}
