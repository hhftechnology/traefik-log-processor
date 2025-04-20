package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
	"github.com/natefinch/lumberjack"
)

// Config holds the application configuration
type Config struct {
	Input struct {
		File      string `yaml:"file"`
		Directory string `yaml:"directory"`
		Pattern   string `yaml:"pattern"`
		Stdin     bool   `yaml:"stdin"`
	} `yaml:"input"`
	Output struct {
		Directory string `yaml:"directory"`
		Format    string `yaml:"format"`
	} `yaml:"output"`
	Rotation struct {
		MaxSize    int  `yaml:"max_size"`
		MaxAge     int  `yaml:"max_age"`
		MaxBackups int  `yaml:"max_backups"`
		Compress   bool `yaml:"compress"`
	} `yaml:"rotation"`
	FieldMapping map[string]string `yaml:"field_mapping"`
}

// TraefikLog represents a parsed Traefik log entry
type TraefikLog map[string]interface{}

// LogWriter manages writing logs to service-specific files with rotation
type LogWriter struct {
	basePath string
	writers  map[string]*lumberjack.Logger
	config   *Config
}

// NewLogWriter creates a new LogWriter
func NewLogWriter(config *Config) *LogWriter {
	return &LogWriter{
		basePath: config.Output.Directory,
		writers:  make(map[string]*lumberjack.Logger),
		config:   config,
	}
}

// WriteLog writes a log entry to the appropriate file based on ServiceName
func (lw *LogWriter) WriteLog(logEntry TraefikLog, rawLog string) error {
	// Extract ServiceName
	serviceName, ok := logEntry["ServiceName"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid ServiceName field")
	}

	// Create directory if needed
	dirPath := filepath.Join(lw.basePath, serviceName)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}

	// Get or create writer for this service
	writer, ok := lw.writers[serviceName]
	if !ok {
		writer = &lumberjack.Logger{
			Filename:   filepath.Join(dirPath, "service.log"),
			MaxSize:    lw.config.Rotation.MaxSize,
			MaxAge:     lw.config.Rotation.MaxAge,
			MaxBackups: lw.config.Rotation.MaxBackups,
			Compress:   lw.config.Rotation.Compress,
		}
		lw.writers[serviceName] = writer
	}

	// Write the raw log entry
	if _, err := writer.Write([]byte(rawLog + "\n")); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}

	return nil
}

// Close closes all open writers
func (lw *LogWriter) Close() {
	for _, writer := range lw.writers {
		writer.Close()
	}
}

// ProcessLogLine processes a single log line
func ProcessLogLine(line string, writer *LogWriter) error {
	// Skip empty lines
	if strings.TrimSpace(line) == "" {
		return nil
	}

	// Parse JSON
	var logEntry TraefikLog
	if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
		return fmt.Errorf("failed to parse JSON log entry: %w", err)
	}

	// Write to appropriate file
	if err := writer.WriteLog(logEntry, line); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}

	return nil
}

// ProcessLogFile processes a single log file
func ProcessLogFile(filePath string, writer *LogWriter) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if err := ProcessLogLine(scanner.Text(), writer); err != nil {
			log.Printf("Warning: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return nil
}

// WatchFile watches a single file for changes
func WatchFile(filePath string, writer *LogWriter) error {
	// Process existing content
	if err := ProcessLogFile(filePath, writer); err != nil {
		log.Printf("Warning: %v", err)
	}

	// Set up file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	// Add file to watcher
	if err := watcher.Add(filepath.Dir(filePath)); err != nil {
		return fmt.Errorf("failed to watch file %s: %w", filePath, err)
	}

	// Track file position
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}
	lastSize := fileInfo.Size()

	// Watch for changes
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("watcher closed")
			}

			// Only process write events for our file
			if event.Name == filePath && (event.Op&fsnotify.Write == fsnotify.Write) {
				fileInfo, err := os.Stat(filePath)
				if err != nil {
					log.Printf("Warning: failed to stat file %s: %v", filePath, err)
					continue
				}

				// Only process new content
				if fileInfo.Size() > lastSize {
					file, err := os.Open(filePath)
					if err != nil {
						log.Printf("Warning: failed to open file %s: %v", filePath, err)
						continue
					}

					// Seek to last position
					if _, err := file.Seek(lastSize, 0); err != nil {
						log.Printf("Warning: failed to seek in file %s: %v", filePath, err)
						file.Close()
						continue
					}

					// Process new lines
					scanner := bufio.NewScanner(file)
					for scanner.Scan() {
						if err := ProcessLogLine(scanner.Text(), writer); err != nil {
							log.Printf("Warning: %v", err)
						}
					}

					file.Close()
					lastSize = fileInfo.Size()
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher closed")
			}
			log.Printf("Warning: watcher error: %v", err)
		}
	}
}

// WatchDirectory watches a directory for log files
func WatchDirectory(dirPath string, pattern string, writer *LogWriter) error {
	// Process existing files
	matches, err := filepath.Glob(filepath.Join(dirPath, pattern))
	if err != nil {
		return fmt.Errorf("failed to find log files: %w", err)
	}

	for _, match := range matches {
		if err := ProcessLogFile(match, writer); err != nil {
			log.Printf("Warning: %v", err)
		}
	}

	// Set up directory watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	// Add directory to watcher
	if err := watcher.Add(dirPath); err != nil {
		return fmt.Errorf("failed to watch directory %s: %w", dirPath, err)
	}

	// Watch for changes
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("watcher closed")
			}

			// Only process create or write events
			if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write {
				// Check if the file matches the pattern
				if matched, _ := filepath.Match(pattern, filepath.Base(event.Name)); matched {
					// Process new content
					if err := ProcessLogFile(event.Name, writer); err != nil {
						log.Printf("Warning: %v", err)
					}
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher closed")
			}
			log.Printf("Warning: watcher error: %v", err)
		}
	}
}

// ProcessStdin processes logs from stdin
func ProcessStdin(writer *LogWriter) error {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		if err := ProcessLogLine(scanner.Text(), writer); err != nil {
			log.Printf("Warning: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read from stdin: %w", err)
	}

	return nil
}

func main() {
	// Parse command line flags
	configFile := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	config := &Config{}
	configData, err := os.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	if err := yaml.Unmarshal(configData, config); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	// Set defaults
	if config.Output.Directory == "" {
		config.Output.Directory = "output"
	}
	if config.Output.Format == "" {
		config.Output.Format = "{{.ServiceName}}"
	}
	if config.Rotation.MaxSize == 0 {
		config.Rotation.MaxSize = 100 // 100 MB
	}
	if config.Rotation.MaxAge == 0 {
		config.Rotation.MaxAge = 7 // 7 days
	}
	if config.Rotation.MaxBackups == 0 {
		config.Rotation.MaxBackups = 5 // 5 backups
	}

	// Create log writer
	writer := NewLogWriter(config)
	defer writer.Close()

	// Process logs based on configuration
	if config.Input.Stdin {
		// Process from stdin
		if err := ProcessStdin(writer); err != nil {
			log.Fatalf("Failed to process stdin: %v", err)
		}
	} else if config.Input.File != "" {
		// Process a single file
		if err := WatchFile(config.Input.File, writer); err != nil {
			log.Fatalf("Failed to watch file: %v", err)
		}
	} else if config.Input.Directory != "" {
		// Process a directory
		pattern := config.Input.Pattern
		if pattern == "" {
			pattern = "*.log"
		}
		if err := WatchDirectory(config.Input.Directory, pattern, writer); err != nil {
			log.Fatalf("Failed to watch directory: %v", err)
		}
	} else {
		log.Fatalf("No input source configured. Set stdin, file, or directory in config.")
	}
}