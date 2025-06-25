package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/bongerka/diploma_exp/internal/generator"
	"github.com/bongerka/diploma_exp/internal/store"
)

func main() {
	var (
		dbType = flag.String("db", "leveldb", "database engine: leveldb|mdbx|lmdb")
		path   = flag.String("path", "./data", "directory for database")
		stage  = flag.Int("stage", 1, "1=bulk load, 2=steady state loop")
	)
	flag.Parse()

	if err := os.MkdirAll(*path, 0o755); err != nil {
		log.Fatalf("cannot create dir: %v", err)
	}

	var st store.Store
	switch *dbType {
	case "leveldb":
		st = &store.LevelStore{}
	case "mdbx":
		st = &store.MdbxStore{}
	case "lmdb":
		st = &store.LmdbStore{}
	default:
		log.Fatalf("unsupported db type %s", *dbType)
	}

	if err := st.Open(filepath.Clean(*path)); err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	g := generator.New()

	metricsPath := filepath.Join(*path, fmt.Sprintf("metrics_%s_stage%d.csv", *dbType, *stage))
	mf, err := os.Create(metricsPath)
	if err != nil {
		log.Fatalf("metrics file: %v", err)
	}
	defer mf.Close()
	mw := bufio.NewWriterSize(mf, 4096)
	defer mw.Flush()

	if *stage == 1 {
		fmt.Println("stage 1")
		fmt.Fprintln(mw, "timestamp_ns,total_written,commit_bytes,latency_ns,throughput_mb_s, is_wal_disabled")
		_ = mw.Flush()
		bulkLoad(st, g, mw)
	} else {
		fmt.Fprintln(mw, "timestamp_ns,batch_seq,commit_bytes,latency_ns,throughput_mb_s")
		_ = mw.Flush()
		steadyState(st, g, mw)
	}
}

func bulkLoad(st store.Store, g *generator.Generator, mw *bufio.Writer) {
	const totalSize = 800 << 30
	fmt.Println("Bulk load started")
	var written int64
	start := time.Now()
	batch := st.NewBatch()

	for written < totalSize {
		op := g.NextOperation()
		applyOp(batch, op)
		if batch.Size() >= generator.BatchSoftBytes {
			sz := batch.Size()
			if err := st.Commit(batch, true); err != nil {
				log.Fatalf("commit: %v", err)
			}
			written += int64(sz)
			fmt.Printf("Committed batch of %d bytes, total written so far: %.2f MiB\n", sz, float64(written)/1024/1024)
			ts := time.Now().UnixNano()
			elapsed := time.Since(start).Seconds()
			if elapsed == 0 {
				elapsed = 1e-9
			}
			throughput := float64(written) / 1024.0 / 1024.0 / elapsed
			fmt.Fprintf(mw, "%d,%d,%d,%d,%.4f\n", ts, written, sz, 0, throughput)
			_ = mw.Flush()
			batch = st.NewBatch()
			if written%(10*generator.BatchSoftBytes) == 0 {
				fmt.Printf("Bulk progress: %d GiB written\n", written>>30)
			}
		}
	}
	dur := time.Since(start)
	fmt.Printf("Bulk load finished: %.2f GB/s\n", float64(written>>20)/dur.Seconds()/1024)
}

func steadyState(st store.Store, g *generator.Generator, mw *bufio.Writer) {
	batchTicker := time.NewTicker(15 * time.Second)
	batch := st.NewBatch()
	var batches int64
	for {
		select {
		case <-batchTicker.C:
			if batch.Size() > 0 {
				sz := batch.Size()
				before := time.Now()
				disableWAL := true
				if batches%60 == 59 {
					disableWAL = false
				}
				if err := st.Commit(batch, disableWAL); err != nil {
					log.Fatalf("commit: %v", err)
				}
				lat := time.Since(before)
				fmt.Printf("Batch %d committed in %s (%.2f MB/s)\n", batches, lat, float64(sz)/lat.Seconds()/1024/1024)
				ts := time.Now().UnixNano()
				fmt.Fprintf(mw, "%d,%d,%d,%d,%.4f,%t\n", ts, batches, sz, lat.Nanoseconds(), float64(sz)/lat.Seconds()/1024/1024, disableWAL)
				_ = mw.Flush()
				batch = st.NewBatch()
				batches++
				if batches%60 == 0 {
					_ = st.Flush()
				}
			}
		default:
			if batch.Size() < generator.BatchSoftBytes {
				applyOp(batch, g.NextOperation())
			} else {
				time.Sleep(100 * time.Microsecond)
			}
		}
	}
}

func applyOp(batch store.Batch, op generator.Operation) {
	switch op.Kind {
	case 0, 1:
		batch.Put(op.Key, op.Value)
	case 2:
		batch.Delete(op.Key)
	}
}
