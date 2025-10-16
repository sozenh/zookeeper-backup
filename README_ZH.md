# zkbackup - ZooKeeper Backup and Restore Tool

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](LICENSE)

zkbackup 是一个专注于 ZooKeeper 数据备份和恢复的独立工具,提供可靠、一致、易用的备份解决方案。

## 特性

- ✅ **完整备份**: 备份所有 snapshot 和 txnlog 文件
- ✅ **可靠恢复**: 自动验证和修复损坏的 txnlog
- ✅ **数据完整**: 保证备份数据完整,不丢失事务
- ✅ **易于集成**: 清晰的 CLI 接口,方便与其他系统集成
- ✅ **通用兼容**: 支持所有 ZooKeeper 版本 (3.4.x ~ 3.9.x)

## 安装

### 从源码构建

```bash
git clone https://github.com/sozenh/zookeeper-backup.git
cd zkbackup
make build
sudo mv zkbackup /usr/local/bin/
```

### 下载二进制

从 [Releases](https://github.com/yourusername/zkbackup/releases) 页面下载对应平台的二进制文件。

### Docker

```bash
docker pull registry.example.com/zkbackup:latest
```

## 快速开始

### 备份

```bash
zkbackup backup \
  --zk-data-dir /zookeeper/data/version-2 \
  --zk-log-dir /zookeeper/datalog/version-2 \
  --output-dir /backup/zookeeper \
  --zk-host localhost:2181
```

### 恢复

```bash
zkbackup restore \
  --backup-dir /backup/zookeeper/backup-20250115-103000 \
  --zk-data-dir /zookeeper/data/version-2 \
  --zk-log-dir /zookeeper/datalog/version-2
```

### 验证备份

```bash
zkbackup verify --backup-dir /backup/zookeeper/backup-20250115-103000
```

### 列出所有备份

```bash
zkbackup list --backup-base-dir /backup/zookeeper
```

### 查看备份详情

```bash
zkbackup info backup-20250115-103000 --backup-base-dir /backup/zookeeper
```

### 清理旧备份

```bash
zkbackup prune --keep-days 7 --keep-min-count 3
```

## 命令详解

### backup - 备份命令

完整备份 ZooKeeper 数据。

```bash
zkbackup backup [flags]

Flags:
  --zk-data-dir string      ZooKeeper dataDir 路径 (必需)
  --zk-log-dir string       ZooKeeper dataLogDir 路径 (必需)
  --output-dir string       备份输出目录 (必需)
  --zk-host string          ZooKeeper 主机地址 (默认: localhost:2181)
  --backup-id string        备份 ID (可选,默认自动生成)
  --verify                  备份后立即验证 (默认: true)
  --compression string      压缩方式: none|gzip|zstd (默认: gzip)
  --verbose                 详细输出
```

### restore - 恢复命令

从备份恢复 ZooKeeper 数据。

```bash
zkbackup restore [flags]

Flags:
  --backup-dir string       备份目录路径 (必需)
  --zk-data-dir string      ZooKeeper dataDir 路径 (必需)
  --zk-log-dir string       ZooKeeper dataLogDir 路径 (必需)
  --force                   强制恢复,不进行确认 (危险)
  --dry-run                 模拟恢复,不实际执行
  --skip-verify             跳过恢复前的验证 (不推荐)
  --truncate-to-zxid string 恢复到指定 ZXID (可选)
  --verbose                 详细输出
```

### verify - 验证命令

验证备份完整性。

```bash
zkbackup verify [flags]

Flags:
  --backup-dir string       备份目录路径 (必需)
  --fix                     自动修复损坏的文件
  --output-format string    输出格式: text|json (默认: text)
  --verbose                 详细输出
```

### list - 列表命令

列出所有备份。

```bash
zkbackup list [flags]

Flags:
  --backup-base-dir string  备份基础目录 (默认: /backup/zookeeper)
  --format string           输出格式: table|json|simple (默认: table)
  --sort-by string          排序方式: time|size|zxid (默认: time)
  --limit int               限制显示数量 (默认: 20)
```

### info - 详情命令

显示备份详细信息。

```bash
zkbackup info <backup-id> [flags]

Flags:
  --backup-base-dir string  备份基础目录 (默认: /backup/zookeeper)
  --format string           输出格式: text|json (默认: text)
```

### prune - 清理命令

清理旧备份。

```bash
zkbackup prune [flags]

Flags:
  --backup-base-dir string  备份基础目录 (默认: /backup/zookeeper)
  --keep-days int           保留天数 (默认: 7)
  --keep-count int          保留数量 (默认: 0,不限制)
  --keep-min-count int      最少保留数量 (默认: 3)
  --dry-run                 模拟删除,不实际执行
  --force                   强制删除,不确认
  --verbose                 详细输出
```

## 配置文件

zkbackup 支持使用配置文件来简化命令行参数:

```yaml
# zkbackup.yaml

zookeeper:
  data_dir: /zookeeper/data/version-2
  log_dir: /zookeeper/datalog/version-2
  host: localhost:2181
  timeout: 5

backup:
  base_dir: /backup/zookeeper
  compression: gzip
  compression_level: 6
  auto_verify: true
  auto_repair: true
  concurrent_copies: 4

restore:
  require_confirmation: true
  verify_before_restore: true
  backup_before_restore: true

prune:
  keep_days: 7
  keep_min_count: 3
  require_confirmation: true

logging:
  level: info
  format: text
  output: stdout
```

配置文件查找顺序:
1. 命令行参数 `--config`
2. 当前目录 `./zkbackup.yaml`
3. 用户目录 `~/.zkbackup.yaml`
4. 系统目录 `/etc/zkbackup/config.yaml`

## 与其他系统集成

### Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: zookeeper-backup
spec:
  schedule: "0 2 * * *"  # 每天凌晨 2 点
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: registry.example.com/zkbackup:latest
            command:
            - /usr/local/bin/zkbackup
            - backup
            - --zk-data-dir=/zookeeper/data/version-2
            - --zk-log-dir=/zookeeper/datalog/version-2
            - --output-dir=/backup/zookeeper
            - --zk-host=zk-0.zk-headless:2181
            volumeMounts:
            - name: zk-data
              mountPath: /zookeeper/data
              readOnly: true
            - name: zk-log
              mountPath: /zookeeper/datalog
              readOnly: true
            - name: backup
              mountPath: /backup
          volumes:
          - name: zk-data
            persistentVolumeClaim:
              claimName: data-zk-0
          - name: zk-log
            persistentVolumeClaim:
              claimName: datalog-zk-0
          - name: backup
            persistentVolumeClaim:
              claimName: backup-storage
          restartPolicy: OnFailure
```

## 备份目录结构

```
<backup-id>/
├── snapshots/              # Snapshot 文件
│   ├── snapshot.100000000
│   ├── snapshot.200000000
│   └── snapshot.300000000
├── txnlogs/                # TxnLog 文件
│   ├── log.100000000
│   ├── log.200000000
│   └── log.300000000
├── metadata/               # 元数据
│   ├── backup_info.json    # 机器可读的元数据
│   ├── MANIFEST.txt        # 人类可读的清单
│   ├── zk_mntr.txt         # ZooKeeper mntr 输出
│   ├── zk_stat.txt         # ZooKeeper stat 输出
│   └── zk_conf.txt         # ZooKeeper conf 输出
└── logs/                   # 日志
    ├── backup.log
    └── verify.log
```

## FAQ

### Q: zkbackup 支持在线备份吗?

A: 是的,zkbackup 支持在线备份 (ZooKeeper 无需停止)。但建议在低峰期备份。

### Q: 为什么需要验证和修复 txnlog?

A: 因为备份时 txnlog 可能正在写入,导致文件不完整或 Checksum 错误。如果不验证,恢复时会失败。

### Q: 如何验证备份是否可用?

A: 使用 `zkbackup verify --backup-dir /path/to/backup` 命令验证。

### Q: 可以备份单个 znode 吗?

A: 不支持。zkbackup 只做全量备份。如需备份单个 znode,请使用 ZooKeeper 客户端。

## 开发

### 构建

```bash
make build
```

### 测试

```bash
make test
```

### 测试覆盖率

```bash
make test-coverage
```

### 代码格式化

```bash
make fmt
```

## 贡献

欢迎提交 Issue 和 Pull Request!

## 许可证

Apache License 2.0

## 文档

详细设计文档请查看 [ZKBACKUP_DESIGN.md](docs/ZKBACKUP_DESIGN.md)
