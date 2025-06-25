package store

type Batch interface {
	Put(key, value []byte)
	Delete(key []byte)
	Size() int
	Clear()
}

type Store interface {
	Open(path string) error
	Close() error
	NewBatch() Batch
	Commit(b Batch, disableWAL bool) error
	Flush() error
}
