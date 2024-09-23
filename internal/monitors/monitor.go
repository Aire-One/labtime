package monitors

type Monitor interface {
	ID() string
	Run() error
}
