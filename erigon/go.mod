module github.com/bongerka/diploma_exp

go 1.24

require (
	github.com/bmatsuo/lmdb-go v1.8.0 // LMDB bindings
	github.com/syndtr/goleveldb v1.0.0 // LevelDB bindings
)

require github.com/golang/snappy v0.0.0-20180518054509-2e65f85255db // indirect

require github.com/erigontech/mdbx-go v0.40.0

require (
	github.com/ianlancetaylor/cgosymbolizer v0.0.0-20241129212102-9c50ad6b591e // indirect
	golang.org/x/sys v0.31.0 // indirect
)

// Any import of "github.com/bmatsuo/lmdb-go/lmdb" actually comes from the v1.8.0 module root:
replace github.com/bmatsuo/lmdb-go/lmdb => github.com/bmatsuo/lmdb-go v1.8.0

// Redirect torquem-ch import to the real erigontech module:
replace github.com/torquem-ch/mdbx-go => github.com/erigontech/mdbx-go v0.40.0
