package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashicorp/go-hclog"

	"github.com/yourorg/toolkit/pkg/config"
	"github.com/yourorg/toolkit/pkg/core"
	"github.com/yourorg/toolkit/pkg/logging"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("tools.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logAdapter, err := logging.NewLogger(logging.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		File:   cfg.Logging.File,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}

	// 创建 hclog.Logger 用于 core 包
	hcLogger := hclog.New(&hclog.LoggerOptions{
		Name:   "toolkit",
		Level:  hclog.LevelFromString(cfg.Logging.Level),
		Output: os.Stdout,
		Color:  hclog.AutoColor,
	})

	logAdapter.Info("toolkit starting", "version", "1.0.0")

	// Create plugin manager
	manager := core.NewPluginManager(&core.ManagerConfig{
		PluginDirs: cfg.PluginDirs,
	}, hcLogger)

	// Discover and load plugins
	if cfg.General.AutoDiscover {
		if err := manager.LoadPlugins(); err != nil {
			hcLogger.Error("failed to load plugins", "error", err)
		}
	}

	// Initialize plugins from config
	configs := make(map[string]map[string]interface{})
	for name, pCfg := range cfg.Plugins {
		if pCfg.Enabled {
			configs[name] = pCfg.Config
		}
	}
	if err := manager.InitializeAll(configs); err != nil {
		hcLogger.Warn("some plugins failed to initialize", "error", err)
	}

	// List loaded plugins
	plugins := manager.ListPlugins()
	if len(plugins) > 0 {
		hcLogger.Info("loaded plugins:")
		for name, mp := range plugins {
			meta := mp.Tool.Metadata()
			hcLogger.Info("  plugin", "name", name, "version", meta.Version, "state", mp.State, "description", meta.Description)
		}
	} else {
		hcLogger.Info("no plugins loaded")
	}

	// Run CLI or interactive mode
	if len(os.Args) > 1 {
		runCommand(manager, logAdapter, os.Args[1:])
		manager.ShutdownAll()
		return
	}

	// Interactive mode: wait for signal
	runInteractive(manager, logAdapter)

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	hcLogger.Info("shutting down...")
	manager.ShutdownAll()
	hcLogger.Info("goodbye")
}

func runCommand(manager *core.PluginManagerImpl, logger logging.Logger, args []string) {
	command := args[0]

	switch command {
	case "list":
		plugins := manager.ListPlugins()
		fmt.Printf("Loaded plugins (%d):\n", len(plugins))
		for name, mp := range plugins {
			meta := mp.Tool.Metadata()
			fmt.Printf("  - %s v%s [%s] %s\n", name, meta.Version, mp.State, meta.Description)
		}

	case "run":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: toolkit run <plugin-name> [params...]")
			os.Exit(1)
		}
		pluginName := args[1]
		params := parseParams(args[2:])

		result, err := manager.ExecutePlugin(pluginName, params)
		if err != nil {
			logger.Error("execution failed", "plugin", pluginName, "error", err)
			os.Exit(1)
		}
		if result.Success {
			fmt.Printf("Success: %v\n", result.Data)
		} else {
			fmt.Printf("Failed: %s\n", result.Error)
			os.Exit(1)
		}

	case "help":
		printHelp()

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func runInteractive(manager *core.PluginManagerImpl, logger logging.Logger) {
	fmt.Println("Toolkit Interactive Mode")
	fmt.Println("Commands: list, run <plugin>, help, exit")

	for {
		fmt.Print("> ")
		var input string
		if _, err := fmt.Scanln(&input); err != nil {
			break
		}

		switch input {
		case "list":
			plugins := manager.ListPlugins()
			for name, mp := range plugins {
				meta := mp.Tool.Metadata()
				fmt.Printf("  %s v%s [%s]\n", name, meta.Version, mp.State)
			}
		case "exit", "quit":
			return
		case "help":
			printHelp()
		default:
			fmt.Println("unknown command, type 'help'")
		}
	}
}

func parseParams(args []string) map[string]interface{} {
	params := make(map[string]interface{})
	for _, arg := range args {
		var key, value string
		for i, c := range arg {
			if c == '=' {
				key = arg[:i]
				value = arg[i+1:]
				break
			}
		}
		if key != "" {
			params[key] = value
		}
	}
	return params
}

func printHelp() {
	fmt.Println(`Toolkit - Plugin Integration Platform

Commands:
  list                    List all loaded plugins
  run <plugin> [k=v...]   Execute a plugin with parameters
  help                    Show this help message
  exit                    Exit interactive mode

Examples:
  toolkit list
  toolkit run hello name=World
  toolkit run file-manager action=list path=./`)
}
