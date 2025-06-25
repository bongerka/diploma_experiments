# diploma_experiments

## Запуск IOarena тестов (`./test`)

```bash
./ioarena/test.sh   
```
---

## Отрисовка

```bash
python3 ioarena/draw.py -d ioarena/logs -o ioarena/plots
```

---

## Запуск Erigon


```bash
./erigon/erigon \
  --datadir           erigon_data \
  --chain             mainnet \
  --prune.mode        full \
  --http --http.addr=0.0.0.0 --http.port=8545 \
  --http.api=eth,net,web3,txpool,debug,trace,erigon,engine \
  --ws  \
  --authrpc.addr      0.0.0.0 \
  --authrpc.port      8551 \
  --authrpc.vhosts    '*' \
  --authrpc.jwtsecret shared_data/jwt.hex \
  --metrics --metrics.addr 0.0.0.0 --metrics.port 6060 \
  --pprof   --pprof.addr   0.0.0.0 --pprof.port 6061 \
  --verbosity         info \
  --log.dir.disable \
  --downloader.verify
```
