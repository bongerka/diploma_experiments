package store

import (
	"os"
	"runtime"

	mdbx "github.com/erigontech/mdbx-go/mdbx"
)

/* ---------- batch ---------- */

type mdbxOp struct {
	del   bool
	key   []byte
	value []byte
}

type mdbxBatch struct {
	ops  []mdbxOp
	size int
}

func (b *mdbxBatch) Put(k, v []byte) {
	key := append([]byte(nil), k...)
	val := append([]byte(nil), v...)

	b.ops = append(b.ops, mdbxOp{key: key, value: val})
	b.size += len(key) + len(val)
}

func (b *mdbxBatch) Delete(k []byte) {
	key := append([]byte(nil), k...)
	b.ops = append(b.ops, mdbxOp{del: true, key: key})
	b.size += len(key)
}

func (b *mdbxBatch) Size() int { return b.size }

func (b *mdbxBatch) Clear() {
	b.ops = b.ops[:0]
	b.size = 0
}

type MdbxStore struct {
	env *mdbx.Env
	dbi mdbx.DBI
}

func (s *MdbxStore) Open(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}

	env, err := mdbx.NewEnv(mdbx.Default)
	if err != nil {
		return err
	}

	const mapSize = 2 << 40
	if err := env.SetGeometry(0, int(mapSize), int(mapSize), 0, 0, 0); err != nil {
		return err
	}
	if err := env.SetOption(mdbx.OptMaxDB, 1); err != nil {
		return err
	}

	flags := mdbx.WriteMap | mdbx.NoMetaSync | mdbx.NoReadahead | mdbx.LifoReclaim
	if err := env.Open(path, uint(flags), 0o644); err != nil {
		return err
	}

	txn, err := env.BeginTxn(nil, mdbx.TxRW)
	if err != nil {
		env.Close()
		return err
	}
	defer txn.Abort()

	dbi, err := txn.OpenDBISimple("main", mdbx.Create)
	if err != nil {
		env.Close()
		return err
	}
	if _, err := txn.Commit(); err != nil {
		env.Close()
		return err
	}

	s.env = env
	s.dbi = dbi
	return nil
}

func (s *MdbxStore) Close() error {
	s.env.Close()
	return nil
}

func (s *MdbxStore) NewBatch() Batch { return &mdbxBatch{} }

func (s *MdbxStore) Commit(b Batch, _ bool) error {
	mb := b.(*mdbxBatch)
	if len(mb.ops) == 0 {
		return nil
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	txn, err := s.env.BeginTxn(nil, mdbx.TxRW)
	if err != nil {
		return err
	}
	for _, op := range mb.ops {
		if op.del {
			if err := txn.Del(s.dbi, op.key, nil); err != nil && !mdbx.IsNotFound(err) {
				txn.Abort()
				return err
			}
		} else {
			if err := txn.Put(s.dbi, op.key, op.value, 0); err != nil {
				txn.Abort()
				return err
			}
		}
	}
	if _, err := txn.Commit(); err != nil {
		return err
	}
	mb.Clear()
	return nil
}

func (s *MdbxStore) Flush() error {
	return s.env.Sync(false, false)
}
