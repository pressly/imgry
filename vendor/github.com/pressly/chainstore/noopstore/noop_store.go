package noopstore

type noopStore struct{}

func New() *noopStore {
	return &noopStore{}
}

func (s *noopStore) Open() (err error)                       { return }
func (s *noopStore) Close() (err error)                      { return }
func (s *noopStore) Put(key string, val []byte) (err error)  { return }
func (s *noopStore) Get(key string) (data []byte, err error) { return }
func (s *noopStore) Del(key string) (err error)              { return }
