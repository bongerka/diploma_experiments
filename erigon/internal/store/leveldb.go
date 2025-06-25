package store

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

type levelBatch struct {
	b    *leveldb.Batch
	size int
}

func (lb *levelBatch) Put(k, v []byte) {
	lb.b.Put(k, v)
	lb.size += len(k) + len(v)
}
func (lb *levelBatch) Delete(k []byte) {
	lb.b.Delete(k)
	lb.size += len(k)
}
func (lb *levelBatch) Size() int { return lb.size }
func (lb *levelBatch) Clear() {
	lb.b.Reset()
	lb.size = 0
}

type LevelStore struct {
	db       *leveldb.DB
	woSync   *opt.WriteOptions
	woNoSync *opt.WriteOptions
}

func (ls *LevelStore) Open(path string) error {
	opts := &opt.Options{
		OpenFilesCacheCapacity: 2000,
		BlockCacheCapacity:     1 << 30,
		WriteBuffer:            1 << 30,
		Compression:            opt.NoCompression,
	}
	db, err := leveldb.OpenFile(path, opts)
	if err != nil {
		return err
	}
	ls.db = db
	ls.woSync = &opt.WriteOptions{Sync: true}
	ls.woNoSync = &opt.WriteOptions{Sync: false}
	return nil
}

func (ls *LevelStore) Close() error {
	if ls.db != nil {
		return ls.db.Close()
	}
	return nil
}

func (ls *LevelStore) NewBatch() Batch {
	return &levelBatch{b: new(leveldb.Batch)}
}

// disableWAL == true  →  быстрая запись (Sync=false)
// disableWAL == false →  контрольный fsync (Sync=true)
func (ls *LevelStore) Commit(b Batch, disableWAL bool) error {
	lb := b.(*levelBatch)
	wo := ls.woNoSync
	if !disableWAL {
		wo = ls.woSync
	}
	if err := ls.db.Write(lb.b, wo); err != nil {
		return err
	}
	lb.Clear()
	return nil
}

func (ls *LevelStore) Flush() error {
	return ls.db.Write(nil, ls.woSync)
}
