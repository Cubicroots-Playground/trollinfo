package shiftnotifier

// Service defines a shift notifier.
type Service interface {
	Start() error
	Stop() error
}
