package store

import (
	"os"

	"github.com/bmatsuo/lmdb-go/lmdb"
)

type lmdbOp struct {
	del   bool
	key   []byte
	value []byte
}

type lmdbBatch struct {
	ops  []lmdbOp
	size int
}

func (lb *lmdbBatch) Put(key, value []byte) {
	k := append([]byte(nil), key...)
	v := append([]byte(nil), value...)
	lb.ops = append(lb.ops, lmdbOp{key: k, value: v})
	lb.size += len(k) + len(v)
}

func (lb *lmdbBatch) Delete(key []byte) {
	k := append([]byte(nil), key...)
	lb.ops = append(lb.ops, lmdbOp{del: true, key: k})
	lb.size += len(k)
}

func (lb *lmdbBatch) Size() int { return lb.size }

func (lb *lmdbBatch) Clear() {
	lb.ops = lb.ops[:0]
	lb.size = 0
}

type LmdbStore struct {
	env *lmdb.Env
	dbi lmdb.DBI
}

func (ls *LmdbStore) Open(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	env, err := lmdb.NewEnv()
	if err != nil {
		return err
	}
	if err := env.SetMaxDBs(1); err != nil {
		return err
	}
	if err := env.SetMapSize(1 << 41); err != nil {
		return err
	}
	flags := lmdb.WriteMap | lmdb.NoMetaSync
	if err := env.Open(path, uint(flags), 0o644); err != nil {
		return err
	}
	var dbi lmdb.DBI
	err = env.Update(func(txn *lmdb.Txn) (err error) {
		dbi, err = txn.OpenRoot(0)
		return err
	})
	if err != nil {
		env.Close()
		return err
	}
	ls.env = env
	ls.dbi = dbi
	return nil
}

func (ls *LmdbStore) Close() error {
	if ls.env != nil {
		ls.env.Close()
	}
	return nil
}

func (ls *LmdbStore) NewBatch() Batch {
	return &lmdbBatch{}
}

func (ls *LmdbStore) Commit(b Batch, _ bool) error {
	lb := b.(*lmdbBatch)
	if len(lb.ops) == 0 {
		return nil
	}
	err := ls.env.Update(func(txn *lmdb.Txn) error {
		for _, op := range lb.ops {
			if op.del {
				if e := txn.Del(ls.dbi, op.key, nil); e != nil && e != lmdb.NotFound {
					return e
				}
			} else {
				if e := txn.Put(ls.dbi, op.key, op.value, 0); e != nil {
					return e
				}
			}
		}
		return nil
	})
	if err == nil {
		lb.Clear()
	}
	return err
}

func (ls *LmdbStore) Flush() error {
	return ls.env.Sync(false)
}
