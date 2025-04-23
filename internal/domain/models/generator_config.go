package models

import "time"

type GeneratorConfig struct {
	KafkaBrokers []string
	KafkaTopic   string
	Count        int
	Interval     time.Duration
	PrintOnly    bool
}