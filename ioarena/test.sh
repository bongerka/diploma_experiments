#!/usr/bin/env bash

for engine in lmdb mdbx rocksdb leveldb; do
  # 1) CRUD + fsync
    ./build-release/src/ioarena \
      -D $engine \
      -B crud \
      -n 1000000 \
      -m sync \
      -k 16 \
      -v 32 \
      -w 1 \
      -C ${engine}_sync_crud > new_results_new/${engine}_sync_crud.txt 2>&1

  # 2) CRUD + lazy
    ./build-release/src/ioarena \
      -D $engine \
      -B crud \
      -n 1000000 \
      -m lazy \
      -k 16 \
      -v 32 \
      -w 1 \
      -C ${engine}_lazy_crud > new_results_new/${engine}_lazy_crud.txt 2>&1

  # 3) CRUD + nosync
    ./build-release/src/ioarena \
      -D $engine \
      -B crud \
      -n 1000000 \
      -m nosync \
      -k 16 \
      -v 32 \
      -w 1 \
      -C ${engine}_nosync_crud > new_results_new/${engine}_nosync_crud.txt 2>&1
done
