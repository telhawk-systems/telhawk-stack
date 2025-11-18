package seeder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/telhawk-systems/telhawk-stack/cli/pkg/attacks"
)

// Runner handles the event seeding execution
type Runner struct {
	Config     *Config
	HTTPClient *http.Client
}

// NewRunner creates a new seeder runner
func NewRunner(config *Config) *Runner {
	return &Runner{
		Config: config,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Run executes the seeding process
func (r *Runner) Run(selectedAttacks []string) error {
	gofakeit.Seed(time.Now().UnixNano())

	log.Printf("Starting event seeder:")
	log.Printf("  HEC URL: %s", r.Config.Defaults.HECURL)
	log.Printf("  Event count: %d", r.Config.Defaults.Count)
	log.Printf("  Interval: %v", r.Config.Defaults.Interval)
	log.Printf("  Batch size: %d", r.Config.Defaults.BatchSize)
	log.Printf("  Time spread: %v", r.Config.Defaults.TimeSpread)

	if r.Config.Defaults.TimeSpread > 0 {
		days := r.Config.Defaults.TimeSpread / (24 * time.Hour)
		eventsPerDay := float64(r.Config.Defaults.Count) / float64(days)
		log.Printf("  Distribution: Jittered (%.1f events/day avg)", eventsPerDay)
	} else {
		log.Printf("  Distribution: Real-time")
	}
	log.Printf("  Event types: %v", r.Config.Defaults.EventTypes)

	successCount := 0
	failCount := 0
	attackSuccessCount := 0

	// Generate and send attack events if configured
	if len(selectedAttacks) > 0 {
		attackSuccess, attackFail := r.runAttacks(selectedAttacks)
		successCount += attackSuccess
		failCount += attackFail
		attackSuccessCount = attackSuccess
	}

	// Generate and send baseline events
	batch := make([]HECEvent, 0, r.Config.Defaults.BatchSize)

	for i := 0; i < r.Config.Defaults.Count; i++ {
		eventType := r.Config.Defaults.EventTypes[rand.Intn(len(r.Config.Defaults.EventTypes))]
		event := GenerateEvent(eventType, i, r.Config.Defaults.Count, r.Config.Defaults.TimeSpread)
		batch = append(batch, event)

		if len(batch) >= r.Config.Defaults.BatchSize || i == r.Config.Defaults.Count-1 {
			if err := r.sendBatch(batch); err != nil {
				log.Printf("Failed to send batch: %v", err)
				failCount += len(batch)
			} else {
				successCount += len(batch)
				progressInterval := r.Config.Defaults.Count / 20
				if progressInterval < 1000 {
					progressInterval = 1000
				}
				if successCount%progressInterval == 0 || successCount >= r.Config.Defaults.Count {
					log.Printf("Progress: %d/%d baseline events sent (%.1f%%)",
						successCount-attackSuccessCount, r.Config.Defaults.Count,
						float64(successCount-attackSuccessCount)*100.0/float64(r.Config.Defaults.Count))
				}
			}
			batch = batch[:0]
		}

		if r.Config.Defaults.Interval > 0 && i < r.Config.Defaults.Count-1 {
			time.Sleep(r.Config.Defaults.Interval)
		}
	}

	log.Printf("\nSeeding complete:")
	log.Printf("  Success: %d events", successCount)
	log.Printf("  Failed: %d events", failCount)

	return nil
}

// runAttacks generates and sends attack pattern events
func (r *Runner) runAttacks(selectedAttacks []string) (successCount, failCount int) {
	for _, attackName := range selectedAttacks {
		attackCfg, ok := r.Config.GetAttack(attackName)
		if !ok {
			log.Printf("Warning: Attack %s not found in config, skipping", attackName)
			continue
		}

		if !attackCfg.Enabled {
			log.Printf("Warning: Attack %s is disabled, skipping", attackName)
			continue
		}

		pattern, ok := attacks.Get(attackCfg.Pattern)
		if !ok {
			log.Printf("Warning: Attack pattern %s not found, skipping %s", attackCfg.Pattern, attackName)
			continue
		}

		// Build attack configuration
		cfg := &attacks.Config{
			Now:        time.Now(),
			TimeSpread: r.Config.Defaults.TimeSpread,
			Params:     attackCfg.Params,
		}

		attackEvents, err := pattern.Generate(cfg)
		if err != nil {
			log.Printf("Failed to generate attack %s: %v", attackName, err)
			continue
		}

		log.Printf("\nInjecting attack: %s (%s)", attackName, attackCfg.Pattern)
		log.Printf("  Attack events: %d", len(attackEvents))
		if len(attackCfg.Params) > 0 {
			log.Printf("  Parameters: %v", attackCfg.Params)
		}

		// Send attack events in batches
		for i := 0; i < len(attackEvents); i += r.Config.Defaults.BatchSize {
			end := i + r.Config.Defaults.BatchSize
			if end > len(attackEvents) {
				end = len(attackEvents)
			}

			// Convert attacks.HECEvent to seeder.HECEvent
			batch := make([]HECEvent, end-i)
			for j, ae := range attackEvents[i:end] {
				batch[j] = HECEvent{
					Time:       ae.Time,
					Event:      ae.Event,
					SourceType: ae.SourceType,
					Index:      ae.Index,
				}
			}

			if err := r.sendBatch(batch); err != nil {
				log.Printf("Failed to send attack batch: %v", err)
				failCount += len(batch)
			} else {
				successCount += len(batch)
			}
		}

		log.Printf("  Attack events sent: %d", len(attackEvents))
	}

	return successCount, failCount
}

// sendBatch sends a batch of events to HEC
func (r *Runner) sendBatch(events []HECEvent) error {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)

	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			return fmt.Errorf("failed to encode event: %w", err)
		}
	}

	// DEBUG: Print first event JSON if DEBUG_DUMP_JSON is set
	if os.Getenv("DEBUG_DUMP_JSON") == "true" && len(events) > 0 {
		eventJSON, _ := json.MarshalIndent(events[0], "", "  ")
		log.Printf("=== DEBUG: First HEC event JSON ===\n%s\n=== END DEBUG ===", string(eventJSON))
	}

	req, err := http.NewRequest("POST", r.Config.Defaults.HECURL+"/services/collector/event", &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Splunk "+r.Config.Defaults.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HEC returned status %d", resp.StatusCode)
	}

	return nil
}

// RunFromRulesDirectory generates events from detection rules in a directory
func (r *Runner) RunFromRulesDirectory(rulesDir string) error {
	log.Printf("Loading rules from %s\n", rulesDir)

	rules, err := LoadRulesFromDirectory(rulesDir)
	if err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	if len(rules) == 0 {
		return fmt.Errorf("no rule files found in %s", rulesDir)
	}

	log.Printf("Found %d rule files\n", len(rules))

	successCount := 0
	failCount := 0
	warningCount := 0

	for _, rule := range rules {
		// Check if rule is supported
		supported, reason := rule.IsSupported()
		if !supported {
			log.Printf("\n⚠ WARN: Cannot generate events for rule '%s': %s", rule.Name, reason)
			warningCount++
			continue
		}

		// Validate rule can be generated
		if err := ValidateRuleCanBeGenerated(rule); err != nil {
			log.Printf("\n⚠ WARN: Rule '%s' validation failed: %v", rule.Name, err)
			warningCount++
			continue
		}

		// Generate events
		generator := NewRuleBasedGenerator(rule, 1.5, nil)
		events, err := generator.GenerateEvents()
		if err != nil {
			log.Printf("\n⚠ WARN: Failed to generate events for rule '%s': %v", rule.Name, err)
			warningCount++
			continue
		}

		// Validate events match rule
		if err := ValidateEventsMatchRule(rule, events); err != nil {
			log.Printf("\n⚠ WARN: Events for rule '%s' failed validation: %v", rule.Name, err)
			warningCount++
			continue
		}

		// Send events in batches
		batchCount := 0
		for i := 0; i < len(events); i += r.Config.Defaults.BatchSize {
			end := i + r.Config.Defaults.BatchSize
			if end > len(events) {
				end = len(events)
			}

			batch := events[i:end]
			if err := r.sendBatch(batch); err != nil {
				log.Printf("  ✗ Failed to send batch: %v", err)
				failCount += len(batch)
			} else {
				successCount += len(batch)
				batchCount += len(batch)
			}
		}

		log.Printf("  ✓ Sent: %d events\n", batchCount)
	}

	log.Printf("\nSeeding complete:")
	log.Printf("  Rules processed: %d", len(rules))
	log.Printf("  Events sent: %d", successCount)
	if failCount > 0 {
		log.Printf("  Failed: %d events", failCount)
	}
	if warningCount > 0 {
		log.Printf("  Warnings: %d", warningCount)
	}

	return nil
}

// RunFromConfigRules generates events from rules configured in the YAML config
func (r *Runner) RunFromConfigRules() error {
	enabledRules := r.Config.GetEnabledRules()
	if len(enabledRules) == 0 {
		return fmt.Errorf("no enabled rules in configuration")
	}

	log.Printf("Generating events from %d configured rules\n", len(enabledRules))

	successCount := 0
	failCount := 0
	warningCount := 0

	for name, ruleCfg := range enabledRules {
		log.Printf("\nProcessing rule config: %s", name)
		log.Printf("  Rule file: %s", ruleCfg.RuleFile)

		// Load rule from file
		rule, err := ParseRuleFile(ruleCfg.RuleFile)
		if err != nil {
			log.Printf("  ✗ Failed to load rule: %v", err)
			warningCount++
			continue
		}

		// Check if rule is supported
		supported, reason := rule.IsSupported()
		if !supported {
			log.Printf("  ⚠ WARN: Rule not supported: %s", reason)
			warningCount++
			continue
		}

		// Validate rule can be generated
		if err := ValidateRuleCanBeGenerated(rule); err != nil {
			log.Printf("  ⚠ WARN: Rule validation failed: %v", err)
			warningCount++
			continue
		}

		// Generate events with configured multiplier and params
		generator := NewRuleBasedGenerator(rule, ruleCfg.Multiplier, ruleCfg.Params)
		events, err := generator.GenerateEvents()
		if err != nil {
			log.Printf("  ✗ Failed to generate events: %v", err)
			warningCount++
			continue
		}

		// Validate events match rule
		if err := ValidateEventsMatchRule(rule, events); err != nil {
			log.Printf("  ⚠ WARN: Events failed validation: %v", err)
			warningCount++
			// Continue anyway - events might still be useful
		}

		// Send events in batches
		batchCount := 0
		for i := 0; i < len(events); i += r.Config.Defaults.BatchSize {
			end := i + r.Config.Defaults.BatchSize
			if end > len(events) {
				end = len(events)
			}

			batch := events[i:end]
			if err := r.sendBatch(batch); err != nil {
				log.Printf("  ✗ Failed to send batch: %v", err)
				failCount += len(batch)
			} else {
				successCount += len(batch)
				batchCount += len(batch)
			}
		}

		log.Printf("  ✓ Sent: %d events", batchCount)
	}

	log.Printf("\nSeeding complete:")
	log.Printf("  Rule configs processed: %d", len(enabledRules))
	log.Printf("  Events sent: %d", successCount)
	if failCount > 0 {
		log.Printf("  Failed: %d events", failCount)
	}
	if warningCount > 0 {
		log.Printf("  Warnings: %d", warningCount)
	}

	return nil
}

// ListAttacks lists all available attack patterns
func ListAttacks() {
	fmt.Println("Available attack patterns:")
	for _, name := range attacks.List() {
		pattern, _ := attacks.Get(name)
		fmt.Printf("  %-15s - %s\n", name, pattern.Description())
	}
}
