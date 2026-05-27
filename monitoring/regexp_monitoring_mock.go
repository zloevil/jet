package monitoring

type regexpMonitoringMock struct {
}

func (i *regexpMonitoringMock) AddRegexps(regexps ...*Regexp) {
	// nothing to do
	return
}

func (i *regexpMonitoringMock) Text(text string) {
	// nothing to do
}

func (i *regexpMonitoringMock) GetCollector() MetricsCollector {
	return func() MetricsCollection {
		return nil
	}
}
