package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"regexp"
)

const (
	RegexpMatchCounter = "regexp_match_counter"
)

type regexpMonitoring struct {
	source             string
	regexps            []*Regexp
	regexpMatchCounter *prometheus.CounterVec
}

type Regexp struct {
	Code   string // Code name in monitoring
	Regexp string // Regexp regular expression to search text match
}

// RegexpMonitoring text metrics
type RegexpMonitoring interface {
	AddRegexps(regexps ...*Regexp)
	Text(text string)
	// GetCollector returns metric collector
	GetCollector() MetricsCollector
}

func NewRegexpMonitoring(cfg *Config, source string) RegexpMonitoring {
	if !cfg.Enabled {
		return &regexpMonitoringMock{}
	}

	m := &regexpMonitoring{
		source: source,
	}

	m.regexpMatchCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: RegexpMatchCounter,
		Help: "Counts regexp match",
	}, []string{"source", "name"})

	return m
}

func (m *regexpMonitoring) AddRegexps(regexps ...*Regexp) {
	m.regexps = append(m.regexps, regexps...)
}

func (m *regexpMonitoring) Text(text string) {
	for _, r := range m.regexps {
		if matched, _ := regexp.MatchString(r.Regexp, text); matched {
			m.regexpMatchCounter.WithLabelValues(m.source, r.Code).Inc()
		}
	}
}

func (m *regexpMonitoring) GetCollector() MetricsCollector {
	return func() MetricsCollection {
		return MetricsCollection{
			m.regexpMatchCounter,
		}
	}
}
