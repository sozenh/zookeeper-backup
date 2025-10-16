# ZooKeeper Backup Tool 设计文档

## 1. 项目概述

### 1.1 项目定位

**zkbackup** 是一个专注于 ZooKeeper 数据备份和恢复的独立工具，提供可靠、一致、易用的备份解决方案。

**设计原则**：
- **专注核心**：只做备份/恢复，不涉及应用层逻辑（挂载、调度等）
- **通用性强**：支持所有 ZooKeeper 版本（3.4.x ~ 3.9.x）
- **数据完整**：保证备份数据完整，不丢失事务
- **恢复可靠**：自动验证和修复损坏的 txnlog
- **易于集成**：清晰的 CLI 接口，方便与其他系统集成

### 1.2 核心功能

```
┌─────────────────────────────────────────┐
│         zkbackup 核心功能                │
├─────────────────────────────────────────┤
│                                         │
│  1. backup   - 完整备份                  │
│  2. restore  - 精确恢复                  │
│  3. verify   - 验证备份                  │
│  4. list     - 列出备份                  │
│  5. info     - 备份详情                  │
│  6. prune    - 清理旧备份                │
│                                         │
└─────────────────────────────────────────┘
```

### 1.3 非功能

以下功能**不**在 zkbackup 范围内：
- ❌ 备份存储管理（S3/NFS 挂载）
- ❌ 备份调度（定时任务）
- ❌ 集群管理
- ❌ 监控告警
- ❌ Web UI

这些功能应由外部系统（如 Kubernetes CronJob、备份服务）提供。

---

## 2. 架构设计

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────┐
│                    zkbackup CLI                          │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │  backup  │  │ restore  │  │  verify  │              │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘              │
│       │             │             │                     │
│       └─────────────┼─────────────┘                     │
│                     │                                   │
│  ┌──────────────────▼──────────────────┐                │
│  │      Core Backup Engine             │                │
│  ├─────────────────────────────────────┤                │
│  │ - Snapshot Manager                  │                │
│  │ - TxnLog Manager                    │                │
│  │ - Integrity Validator               │                │
│  │ - Metadata Manager                  │                │
│  └─────────────────┬───────────────────┘                │
│                    │                                    │
│  ┌─────────────────▼───────────────────┐                │
│  │      ZooKeeper File I/O             │                │
│  ├─────────────────────────────────────┤                │
│  │ - Snapshot Reader/Writer            │                │
│  │ - TxnLog Reader/Writer/Truncator    │                │
│  │ - ZXID Parser                       │                │
│  │ - Checksum Validator                │                │
│  └─────────────────┬───────────────────┘                │
│                    │                                    │
└────────────────────┼────────────────────────────────────┘
                     │
         ┌───────────▼───────────┐
         │  ZooKeeper 文件系统    │
         │  - dataDir             │
         │  - dataLogDir          │
         └───────────────────────┘
```

### 2.2 模块划分

#### 2.2.1 CLI 层 (cmd/)
```
cmd/
├── backup.go      # backup 命令实现
├── restore.go     # restore 命令实现
├── verify.go      # verify 命令实现
├── list.go        # list 命令实现
├── info.go        # info 命令实现
├── prune.go       # prune 命令实现
└── root.go        # 根命令和全局配置
```

#### 2.2.2 核心引擎 (pkg/engine/)
```
pkg/engine/
├── backup.go      # 备份引擎
├── restore.go     # 恢复引擎
├── verify.go      # 验证引擎
└── config.go      # 配置管理
```

#### 2.2.3 文件处理 (pkg/zkfile/)
```
pkg/zkfile/
├── snapshot.go    # Snapshot 文件读写
├── txnlog.go      # TxnLog 文件读写解析
├── validator.go   # 文件完整性验证
├── truncator.go   # TxnLog 截断工具
└── zxid.go        # ZXID 解析和比较
```

#### 2.2.4 元数据管理 (pkg/metadata/)
```
pkg/metadata/
├── backup_info.go # 备份元数据结构
├── manager.go     # 元数据持久化
└── report.go      # 备份报告生成
```

#### 2.2.5 工具函数 (pkg/utils/)
```
pkg/utils/
├── file.go        # 文件操作工具
├── zk_client.go   # ZooKeeper 客户端（获取 ZXID）
└── logger.go      # 日志工具
```

---

## 3. 核心功能设计

### 3.1 Backup（备份）

#### 3.1.1 命令接口

```bash
zkbackup backup [flags]

Flags:
  --zk-data-dir string      ZooKeeper dataDir 路径（必需）
  --zk-log-dir string       ZooKeeper dataLogDir 路径（必需）
  --output-dir string       备份输出目录（必需）
  --zk-host string          ZooKeeper 主机地址，用于获取 ZXID（默认: localhost:2181）
  --backup-id string        备份 ID（可选，默认自动生成）
  --verify                  备份后立即验证（默认: true）
  --compression string      压缩方式: none|gzip|zstd（默认: gzip）
  --verbose                 详细输出
```

#### 3.1.2 备份流程

```
1. 预检查
   ├─ 检查 zk-data-dir 和 zk-log-dir 是否存在
   ├─ 检查输出目录是否可写
   └─ 连接 ZooKeeper 获取当前状态

2. 记录备份基准点
   ├─ 获取当前 ZXID（通过 mntr 命令）
   ├─ 记录备份时间戳
   └─ 生成备份 ID

3. 备份 Snapshot 文件
   ├─ 列出所有 snapshot.*
   ├─ 复制到 output-dir/snapshots/
   └─ 记录文件列表和 ZXID

4. 备份 TxnLog 文件
   ├─ 列出所有 log.*
   ├─ 复制到 output-dir/txnlogs/
   ├─ 标记可能正在写入的最新文件
   └─ 记录文件列表和 ZXID 范围

5. 备份配置文件（可选，通过 --include-config）
   ├─ 备份 zoo.cfg
   ├─ 备份 myid
   ├─ 备份 jaas.conf（如果存在）
   ├─ 备份 java.env（如果存在）
   ├─ 备份 SSL 证书（如果启用）
   └─ ⚠️ 不备份 Kerberos keytab（安全敏感）

6. 验证备份（如果 --verify=true）
   ├─ 验证 snapshot 文件格式
   ├─ 验证每个 txnlog 完整性
   ├─ 检测并修复损坏的 txnlog
   └─ 生成验证报告

7. 生成元数据
   ├─ 创建 backup_info.json
   ├─ 创建 MANIFEST.txt（人类可读）
   ├─ 保存 ZooKeeper 状态（mntr, stat, conf）
   └─ 记录 ACL 配置信息（检测到的认证方式）

8. 输出结果
   ├─ 打印备份摘要
   ├─ 提示配置文件备份状态
   ├─ 提示需要手动备份的安全敏感文件
   ├─ 返回备份目录路径
   └─ 退出码：0=成功，1=失败，2=部分成功（有损坏但已修复）
```

#### 3.1.3 备份目录结构

```
<output-dir>/<backup-id>/
├── snapshots/
│   ├── snapshot.100000000
│   ├── snapshot.200000000
│   └── snapshot.300000000
├── txnlogs/
│   ├── log.100000000
│   ├── log.200000000
│   ├── log.300000000
│   └── log.400000000 (可能损坏，会被验证)
├── metadata/
│   ├── backup_info.json      # 机器可读的元数据
│   ├── MANIFEST.txt           # 人类可读的清单
│   ├── zk_mntr.txt            # ZooKeeper mntr 输出
│   ├── zk_stat.txt            # ZooKeeper stat 输出
│   └── zk_conf.txt            # ZooKeeper conf 输出
└── logs/
    ├── backup.log             # 备份日志
    └── verify.log             # 验证日志（如果执行）
```

#### 3.1.4 backup_info.json 格式

```json
{
  "version": "1.0",
  "backup_id": "backup-20250115-103000",
  "backup_timestamp": "2025-01-15T10:30:00+08:00",
  "backup_zxid": {
    "hex": "0x500000001",
    "decimal": 21474836481
  },
  "zookeeper": {
    "version": "3.8.0",
    "host": "localhost:2181",
    "data_dir": "/zookeeper/data/version-2",
    "log_dir": "/zookeeper/datalog/version-2"
  },
  "files": {
    "snapshots": [
      {
        "name": "snapshot.100000000",
        "zxid": "0x100000000",
        "size": 1048576,
        "checksum": "sha256:abc123..."
      }
    ],
    "txnlogs": [
      {
        "name": "log.100000000",
        "start_zxid": "0x100000000",
        "end_zxid": "0x1ffffffff",
        "size": 5242880,
        "status": "valid",
        "transaction_count": 1024
      },
      {
        "name": "log.400000000",
        "start_zxid": "0x400000000",
        "end_zxid": "0x500000001",
        "size": 2097152,
        "status": "truncated",
        "transaction_count": 512,
        "note": "Truncated to backup_zxid"
      }
    ]
  },
  "validation": {
    "enabled": true,
    "total_files": 8,
    "valid_files": 7,
    "corrupted_files": 1,
    "repaired_files": 1,
    "unrecoverable_files": 0
  },
  "statistics": {
    "total_size": 104857600,
    "compressed_size": 52428800,
    "duration_seconds": 12.5
  }
}
```

---

### 3.2 Restore（恢复）

#### 3.2.1 命令接口

```bash
zkbackup restore [flags]

Flags:
  --backup-dir string       备份目录路径（必需）
  --zk-data-dir string      ZooKeeper dataDir 路径（必需）
  --zk-log-dir string       ZooKeeper dataLogDir 路径（必需）
  --force                   强制恢复，不进行确认（危险）
  --dry-run                 模拟恢复，不实际执行
  --skip-verify             跳过恢复前的验证（不推荐）
  --truncate-to-zxid string 恢复到指定 ZXID（可选，默认使用备份的 backup_zxid）
  --verbose                 详细输出
```

#### 3.2.2 恢复流程

```
1. 预检查
   ├─ 验证备份目录结构完整
   ├─ 读取 backup_info.json
   ├─ 检查目标目录是否为空（非空需要确认）
   └─ 检查 ZooKeeper 是否在运行（必须停止）

2. 验证备份完整性（如果 --skip-verify=false）
   ├─ 验证所有 snapshot 文件
   ├─ 验证所有 txnlog 文件
   └─ 确认备份可用

3. 确认恢复操作（如果 --force=false）
   ├─ 显示备份信息（时间、ZXID、文件数）
   ├─ 显示目标信息（dataDir、logDir）
   ├─ 询问用户确认
   └─ 等待用户输入 yes/no

4. 备份现有数据（安全措施）
   ├─ 创建 /zookeeper/backup_before_restore_<timestamp>
   ├─ 移动现有文件到备份目录
   └─ 记录原始数据位置

5. 恢复 Snapshot 文件
   ├─ 复制所有 snapshot 到 dataDir
   └─ 验证复制后的文件

6. 处理 TxnLog 文件（关键步骤）
   ├─ 对每个 txnlog：
   │  ├─ 读取并验证完整性
   │  ├─ 截断到 truncate-to-zxid（如果指定）
   │  └─ 复制到 logDir
   └─ 跳过标记为 corrupted 的文件

7. 设置文件权限
   ├─ chown zookeeper:zookeeper
   └─ chmod 755

8. 生成恢复报告
   ├─ 记录恢复的文件列表
   ├─ 记录预期的 ZXID
   └─ 输出下一步操作指南

9. 输出结果
   └─ 退出码：0=成功，1=失败
```

#### 3.2.3 恢复后验证

恢复完成后，工具会输出验证指令：

```bash
# 恢复完成后的建议操作
zkbackup restore --backup-dir /backup/backup-20250115-103000 \
                 --zk-data-dir /zookeeper/data/version-2 \
                 --zk-log-dir /zookeeper/datalog/version-2

# 输出:
✅ 恢复完成！

下一步操作:
1. 启动 ZooKeeper:
   zkServer.sh start

2. 验证 ZXID:
   echo mntr | nc localhost 2181 | grep zk_zxid
   预期: 0x500000001

3. 验证数据完整性:
   zkCli.sh -server localhost:2181
   ls /
   get /your/important/node

4. 如果恢复失败，可以回滚:
   zkServer.sh stop
   rm -rf /zookeeper/data/version-2/* /zookeeper/datalog/version-2/*
   mv /zookeeper/backup_before_restore_20250115103000/* /zookeeper/data/version-2/
   zkServer.sh start
```

---

### 3.3 Verify（验证）

#### 3.3.1 命令接口

```bash
zkbackup verify [flags]

Flags:
  --backup-dir string       备份目录路径（必需）
  --fix                     自动修复损坏的文件
  --output-format string    输出格式: text|json（默认: text）
  --verbose                 详细输出
```

#### 3.3.2 验证内容

```
1. 目录结构验证
   ├─ 检查必需的子目录存在
   ├─ 检查 backup_info.json 存在且有效
   └─ 检查 MANIFEST.txt 存在

2. Snapshot 文件验证
   ├─ 验证文件格式（Magic Number）
   ├─ 验证文件大小 > 0
   ├─ 计算并比对 checksum（如果有）
   └─ 解析 ZXID

3. TxnLog 文件验证
   ├─ 验证文件格式（Magic Number）
   ├─ 逐个读取事务，验证 checksum
   ├─ 统计事务数量
   ├─ 检测损坏位置
   └─ 如果 --fix=true，截断损坏部分

4. 元数据一致性验证
   ├─ 验证文件列表与实际文件匹配
   ├─ 验证 ZXID 范围覆盖 snapshot
   └─ 验证备份完整性

5. 生成验证报告
   ├─ 总体健康度评分
   ├─ 详细问题列表
   └─ 修复建议
```

#### 3.3.3 输出示例

```
╔════════════════════════════════════════════════════════════╗
║           备份验证报告                                      ║
╚════════════════════════════════════════════════════════════╝

备份ID: backup-20250115-103000
备份时间: 2025-01-15T10:30:00+08:00
备份ZXID: 0x500000001

目录结构: ✅ 完整
元数据文件: ✅ 有效

Snapshot 文件验证:
  snapshot.100000000: ✅ 有效 (ZXID: 0x100000000, 1.0MB)
  snapshot.200000000: ✅ 有效 (ZXID: 0x200000000, 1.5MB)
  snapshot.300000000: ✅ 有效 (ZXID: 0x300000000, 2.0MB)

  总计: 3 个文件，全部有效

TxnLog 文件验证:
  log.100000000: ✅ 有效 (1024 个事务, 5.0MB)
  log.200000000: ✅ 有效 (2048 个事务, 10.0MB)
  log.300000000: ✅ 有效 (1536 个事务, 7.5MB)
  log.400000000: ⚠️  部分损坏 (512 个有效事务, 2.0MB)
                 └─ 截断到事务 512（ZXID: 0x500000001）

  总计: 4 个文件，3 个完整，1 个已修复

备份覆盖范围:
  最早 ZXID: 0x100000000
  最晚 ZXID: 0x500000001
  Snapshot 覆盖: ✅ 有效
  TxnLog 覆盖: ✅ 完整

整体评估: ✅ 备份可用
  - 所有必需文件完整
  - 有 1 个文件需要修复（已自动修复）
  - 可以安全用于恢复

建议:
  - 备份质量良好，可以用于恢复
  - 建议保留此备份作为恢复点
```

---

### 3.4 List（列出备份）

#### 3.4.1 命令接口

```bash
zkbackup list [flags]

Flags:
  --backup-base-dir string  备份基础目录（默认: /backup/zookeeper）
  --format string           输出格式: table|json|simple（默认: table）
  --sort-by string          排序方式: time|size|zxid（默认: time）
  --limit int               限制显示数量（默认: 20）
```

#### 3.4.2 输出示例

```
╔══════════════════════════════════════════════════════════════════════════════════╗
║                            ZooKeeper 备份列表                                     ║
╚══════════════════════════════════════════════════════════════════════════════════╝

备份基础目录: /backup/zookeeper
总备份数: 15

┌──────────────────────────┬─────────────────────┬───────────────┬──────────┬────────┐
│ 备份 ID                   │ 时间                 │ ZXID          │ 大小     │ 状态   │
├──────────────────────────┼─────────────────────┼───────────────┼──────────┼────────┤
│ backup-20250115-103000   │ 2025-01-15 10:30:00 │ 0x500000001   │ 100.5 MB │ ✅ 有效│
│ backup-20250115-020000   │ 2025-01-15 02:00:00 │ 0x4ffffffff   │ 98.2 MB  │ ✅ 有效│
│ backup-20250114-020000   │ 2025-01-14 02:00:00 │ 0x4fffffff0   │ 95.1 MB  │ ✅ 有效│
│ backup-20250113-020000   │ 2025-01-13 02:00:00 │ 0x4fffffe00   │ 92.3 MB  │ ⚠️ 部分│
│ backup-20250112-020000   │ 2025-01-12 02:00:00 │ 0x4fffffd00   │ 89.5 MB  │ ✅ 有效│
│ ...                      │ ...                 │ ...           │ ...      │ ...    │
└──────────────────────────┴─────────────────────┴───────────────┴──────────┴────────┘

提示:
  - ✅ 有效: 备份完整可用
  - ⚠️ 部分: 有损坏但已修复，可用
  - ❌ 损坏: 无法恢复

使用 'zkbackup info <backup-id>' 查看详细信息
```

---

### 3.5 Info（备份详情）

#### 3.5.1 命令接口

```bash
zkbackup info <backup-id> [flags]

Flags:
  --backup-base-dir string  备份基础目录（默认: /backup/zookeeper）
  --format string           输出格式: text|json（默认: text）
```

#### 3.5.2 输出示例

```
╔════════════════════════════════════════════════════════════╗
║           备份详细信息                                      ║
╚════════════════════════════════════════════════════════════╝

备份 ID: backup-20250115-103000
位置: /backup/zookeeper/backup-20250115-103000

基本信息:
  备份时间: 2025-01-15 10:30:00 +08:00
  备份 ZXID: 0x500000001 (21474836481)
  ZooKeeper 版本: 3.8.0
  源主机: zk-server-01:2181

文件统计:
  Snapshot 文件: 3 个
  TxnLog 文件: 4 个
  总大小: 100.5 MB
  压缩后: 50.2 MB

Snapshot 列表:
  ├─ snapshot.100000000  ZXID: 0x100000000  1.0 MB
  ├─ snapshot.200000000  ZXID: 0x200000000  1.5 MB
  └─ snapshot.300000000  ZXID: 0x300000000  2.0 MB

TxnLog 列表:
  ├─ log.100000000  ZXID: 0x100000000~0x1ffffffff  5.0 MB  1024 txns
  ├─ log.200000000  ZXID: 0x200000000~0x2ffffffff  10.0 MB  2048 txns
  ├─ log.300000000  ZXID: 0x300000000~0x3ffffffff  7.5 MB  1536 txns
  └─ log.400000000  ZXID: 0x400000000~0x500000001  2.0 MB  512 txns ⚠️

验证状态:
  总文件数: 7
  有效文件: 6
  损坏已修复: 1
  无法恢复: 0

  整体状态: ✅ 可用

备份质量评分: 95/100
  - 数据完整性: ✅ 完整
  - 文件健康度: ⚠️ 良好（1个修复）
  - 覆盖范围: ✅ 完整

恢复命令:
  zkbackup restore --backup-dir /backup/zookeeper/backup-20250115-103000 \
                   --zk-data-dir /zookeeper/data/version-2 \
                   --zk-log-dir /zookeeper/datalog/version-2
```

---

### 3.6 Prune（清理旧备份）

#### 3.6.1 命令接口

```bash
zkbackup prune [flags]

Flags:
  --backup-base-dir string  备份基础目录（默认: /backup/zookeeper）
  --keep-days int           保留天数（默认: 7）
  --keep-count int          保留数量（默认: 0，不限制）
  --keep-min-count int      最少保留数量（默认: 3）
  --dry-run                 模拟删除，不实际执行
  --force                   强制删除，不确认
  --verbose                 详细输出
```

#### 3.6.2 清理策略

```
清理规则（按优先级）:

1. 保护最近的备份
   └─ 始终保留最近 keep-min-count 个备份（默认 3）

2. 按时间清理
   └─ 删除超过 keep-days 天的备份

3. 按数量清理（如果指定 keep-count）
   └─ 保留最近 keep-count 个备份

4. 跳过损坏的备份
   └─ 只删除已验证有效的备份
      （避免误删唯一可用的备份）
```

#### 3.6.3 输出示例

```
╔════════════════════════════════════════════════════════════╗
║           清理旧备份                                        ║
╚════════════════════════════════════════════════════════════╝

备份目录: /backup/zookeeper
清理策略: 保留 7 天，最少 3 个

扫描结果:
  总备份数: 15
  符合删除条件: 8
  受保护（最近3个）: 3
  将删除: 5

待删除备份:
  ├─ backup-20250108-020000  7 天前  89.5 MB  ✅
  ├─ backup-20250107-020000  8 天前  87.2 MB  ✅
  ├─ backup-20250106-020000  9 天前  85.1 MB  ✅
  ├─ backup-20250105-020000  10 天前  83.5 MB  ❌ 损坏（跳过）
  └─ backup-20250104-020000  11 天前  81.2 MB  ✅

将释放空间: 345.5 MB

确认删除? (yes/no): yes

删除中...
  ✅ backup-20250108-020000
  ✅ backup-20250107-020000
  ✅ backup-20250106-020000
  ⏭️  backup-20250105-020000 (跳过：损坏)
  ✅ backup-20250104-020000

完成！
  删除: 4 个备份
  跳过: 1 个备份
  释放空间: 341.7 MB
```

---

## 4. TxnLog 处理核心

### 4.1 TxnLog 文件格式

```
┌─────────────────────────────────────────────────────────┐
│                     File Header                          │
├─────────────────────────────────────────────────────────┤
│ Magic Number (4 bytes): 0x5a4b4c47 ("ZKLG")            │
│ Version (4 bytes): 2                                    │
│ DbId (8 bytes): Cluster Database ID                     │
├─────────────────────────────────────────────────────────┤
│                   Transaction Records                    │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ Record 1:                                           │ │
│ │   Checksum (8 bytes): Adler32/CRC32                 │ │
│ │   Length (4 bytes): Record body length              │ │
│ │   Body (variable):                                  │ │
│ │     ├─ Client ID (8 bytes)                          │ │
│ │     ├─ Cxid (4 bytes)                               │ │
│ │     ├─ ZXID (8 bytes)     ← 关键字段                │ │
│ │     ├─ Timestamp (8 bytes)                          │ │
│ │     ├─ Type (4 bytes)                               │ │
│ │     └─ TxnData (variable)                           │ │
│ └─────────────────────────────────────────────────────┘ │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ Record 2: ...                                       │ │
│ └─────────────────────────────────────────────────────┘ │
│ ...                                                     │
├─────────────────────────────────────────────────────────┤
│          Padding (zero bytes, pre-allocated)            │
└─────────────────────────────────────────────────────────┘
```

### 4.2 验证算法

```go
// TxnLog 验证伪代码
func ValidateTxnLog(logPath string) (*ValidationResult, error) {
    file := OpenFile(logPath)

    // 1. 验证文件头
    header := ReadHeader(file)
    if header.Magic != 0x5a4b4c47 {
        return nil, ErrInvalidMagic
    }

    result := &ValidationResult{}

    // 2. 逐个验证事务
    for {
        pos := file.CurrentPosition()

        // 读取 checksum
        checksum, err := ReadInt64(file)
        if err == io.EOF {
            break // 正常结束
        }

        // 读取长度
        length, err := ReadInt32(file)
        if err != nil || length <= 0 || length > MaxRecordSize {
            result.LastValidPos = pos
            result.CorruptionType = "InvalidLength"
            break
        }

        // 读取记录体
        body, err := ReadBytes(file, length)
        if err != nil {
            result.LastValidPos = pos
            result.CorruptionType = "TruncatedBody"
            break
        }

        // 验证 checksum
        calculatedChecksum := CalculateChecksum(body)
        if calculatedChecksum != checksum {
            result.LastValidPos = pos
            result.CorruptionType = "ChecksumMismatch"
            break
        }

        // 解析 ZXID
        zxid := ParseZxid(body)
        result.Transactions = append(result.Transactions, zxid)
        result.ValidTransactionCount++
    }

    result.IsValid = (result.CorruptionType == "")
    return result, nil
}
```

### 4.3 截断算法

```go
// TxnLog 截断伪代码
func TruncateTxnLog(inputPath, outputPath string, maxZxid uint64) error {
    inFile := OpenFile(inputPath)
    outFile := CreateFile(outputPath)

    // 1. 复制文件头
    header := ReadHeader(inFile)
    WriteHeader(outFile, header)

    // 2. 逐个处理事务
    for {
        // 记录位置
        pos := inFile.CurrentPosition()

        // 读取事务
        checksum := ReadInt64(inFile)
        length := ReadInt32(inFile)
        body := ReadBytes(inFile, length)

        // 解析 ZXID
        zxid := ParseZxid(body)

        // 检查是否超过截断点
        if zxid > maxZxid {
            log.Info("Truncating at ZXID", zxid)
            break
        }

        // 写入到输出文件
        WriteInt64(outFile, checksum)
        WriteInt32(outFile, length)
        WriteBytes(outFile, body)
    }

    outFile.Close()
    return nil
}
```

### 4.4 修复策略

```
修复策略决策树:

损坏类型 → 修复方法
├─ InvalidMagic
│  └─ ❌ 无法修复（文件头损坏）
│
├─ InvalidLength
│  └─ ✅ 截断到上一个有效事务
│
├─ TruncatedBody
│  └─ ✅ 截断到上一个有效事务
│
├─ ChecksumMismatch
│  ├─ 位置在文件末尾？
│  │  └─ ✅ 截断到上一个有效事务
│  └─ 位置在文件中间？
│     └─ ⚠️  严重损坏，尝试恢复部分数据
│
└─ EOF
   └─ ✅ 正常结束，无需修复
```

---

## 5. 配置文件

### 5.1 配置文件位置

```
优先级（从高到低）:
1. 命令行参数
2. 环境变量
3. 当前目录配置文件: ./zkbackup.yaml
4. 用户配置文件: ~/.zkbackup.yaml
5. 系统配置文件: /etc/zkbackup/config.yaml
```

### 5.2 配置文件格式

```yaml
# zkbackup.yaml

# ZooKeeper 配置
zookeeper:
  # ZooKeeper 数据目录
  data_dir: /zookeeper/data/version-2

  # ZooKeeper 事务日志目录
  log_dir: /zookeeper/datalog/version-2

  # ZooKeeper 主机地址（用于获取 ZXID）
  host: localhost:2181

  # 连接超时（秒）
  timeout: 5

# 备份配置
backup:
  # 备份基础目录
  base_dir: /backup/zookeeper

  # 默认压缩方式: none|gzip|zstd
  compression: gzip

  # 压缩级别（1-9，gzip/zstd）
  compression_level: 6

  # 备份后自动验证
  auto_verify: true

  # 备份后自动修复损坏文件
  auto_repair: true

  # 并发复制文件数
  concurrent_copies: 4

# 恢复配置
restore:
  # 恢复前必须确认
  require_confirmation: true

  # 恢复前自动验证备份
  verify_before_restore: true

  # 恢复前备份现有数据
  backup_before_restore: true

# 清理配置
prune:
  # 默认保留天数
  keep_days: 7

  # 最少保留数量
  keep_min_count: 3

  # 删除前确认
  require_confirmation: true

# 日志配置
logging:
  # 日志级别: debug|info|warn|error
  level: info

  # 日志格式: text|json
  format: text

  # 日志输出: stdout|stderr|file
  output: stdout

  # 日志文件路径（当 output=file 时）
  file: /var/log/zkbackup/zkbackup.log

# 高级配置
advanced:
  # TxnLog 最大记录大小（字节）
  max_txn_record_size: 10485760  # 10MB

  # 文件复制缓冲区大小（字节）
  copy_buffer_size: 1048576  # 1MB

  # 验证超时（秒）
  validation_timeout: 300  # 5分钟
```

---

## 6. 错误处理

### 6.1 错误码定义

```go
const (
    // 成功
    ExitCodeSuccess = 0

    // 一般错误
    ExitCodeError = 1

    // 部分成功（有警告）
    ExitCodePartialSuccess = 2

    // 用户取消
    ExitCodeCanceled = 3

    // 验证失败
    ExitCodeValidationFailed = 10

    // 备份失败
    ExitCodeBackupFailed = 20

    // 恢复失败
    ExitCodeRestoreFailed = 30

    // 配置错误
    ExitCodeConfigError = 40
)
```

### 6.2 错误类型

```go
// 错误分类
type ErrorCategory int

const (
    ErrorCategoryIO ErrorCategory = iota          // 文件IO错误
    ErrorCategoryValidation                       // 验证错误
    ErrorCategoryCorruption                       // 数据损坏
    ErrorCategoryConfiguration                    // 配置错误
    ErrorCategoryZooKeeper                        // ZooKeeper错误
    ErrorCategoryUser                             // 用户错误（参数等）
)

// 结构化错误
type BackupError struct {
    Category ErrorCategory
    Message  string
    Cause    error
    Context  map[string]interface{}
}
```

### 6.3 错误恢复策略

```
错误类型 → 处理策略
├─ IO Error
│  ├─ 文件不存在 → 明确提示，退出
│  ├─ 权限不足 → 提示权限要求，退出
│  └─ 磁盘空间不足 → 提示清理空间，退出
│
├─ Validation Error
│  ├─ Snapshot 损坏 → 跳过该 snapshot，使用其他
│  └─ TxnLog 损坏 → 尝试修复，失败则跳过
│
├─ Configuration Error
│  └─ 提示正确的配置格式，退出
│
└─ ZooKeeper Error
   ├─ 连接失败 → 提示检查 ZK 状态，可选择继续
   └─ ZXID 获取失败 → 警告，使用本地文件推断
```

---

## 7. 测试策略

### 7.1 单元测试

```
pkg/zkfile/
├─ snapshot_test.go      # Snapshot 读写测试
├─ txnlog_test.go        # TxnLog 解析测试
├─ validator_test.go     # 验证器测试
├─ truncator_test.go     # 截断器测试
└─ zxid_test.go          # ZXID 解析测试

测试覆盖率目标: > 80%
```

### 7.2 集成测试

```
tests/integration/
├─ backup_test.go        # 完整备份流程测试
├─ restore_test.go       # 完整恢复流程测试
├─ verify_test.go        # 验证流程测试
└─ corruption_test.go    # 损坏恢复测试

测试场景:
1. 正常备份和恢复
2. 损坏文件的备份和修复
3. 大数据量备份（100GB+）
4. 并发备份
5. 不同 ZooKeeper 版本兼容性
```

### 7.3 性能测试

```
性能指标:
- 备份速度: > 100 MB/s（本地磁盘）
- 恢复速度: > 100 MB/s（本地磁盘）
- 验证速度: > 50 MB/s
- 内存占用: < 100 MB（不论数据大小）
- CPU 占用: < 50%（单核）
```

---

## 8. 部署和使用

### 8.1 安装

```bash
# 方式 1: 从源码构建
git clone https://gitlab.woqutech.com/zkbackup.git
cd zkbackup
go build -o zkbackup cmd/zkbackup/main.go
sudo mv zkbackup /usr/local/bin/

# 方式 2: 下载二进制
wget https://github.com/xxx/zkbackup/releases/download/v1.0.0/zkbackup-linux-amd64
chmod +x zkbackup-linux-amd64
sudo mv zkbackup-linux-amd64 /usr/local/bin/zkbackup

# 方式 3: Docker
docker pull registry.woqutech.com/zkbackup:latest
```

### 8.2 与现有系统集成

#### 8.2.1 Kubernetes CronJob

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
            image: registry.woqutech.com/zkbackup:latest
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

#### 8.2.2 与备份服务集成

```bash
#!/bin/bash
# backup-wrapper.sh
# 集成到现有备份系统的包装脚本

set -euo pipefail

# 1. 执行 zkbackup
BACKUP_DIR=$(zkbackup backup \
    --zk-data-dir=/zookeeper/data/version-2 \
    --zk-log-dir=/zookeeper/datalog/version-2 \
    --output-dir=/backup/zookeeper \
    --zk-host=localhost:2181 \
    --format=json | jq -r '.backup_dir')

# 2. 上传到远程存储（S3/NFS）
echo "Uploading to remote storage..."
aws s3 sync "$BACKUP_DIR" "s3://my-backups/zookeeper/$(basename $BACKUP_DIR)"

# 3. 注册到备份服务
curl -X POST https://backup-service/api/backups \
    -H "Content-Type: application/json" \
    -d "{
        \"type\": \"zookeeper\",
        \"backup_id\": \"$(basename $BACKUP_DIR)\",
        \"location\": \"s3://my-backups/zookeeper/$(basename $BACKUP_DIR)\",
        \"metadata\": $(cat $BACKUP_DIR/metadata/backup_info.json)
    }"

# 4. 清理本地旧备份
zkbackup prune --keep-days=3 --force

echo "Backup completed: $(basename $BACKUP_DIR)"
```

---

## 9. 监控和告警

### 9.1 关键指标

```
备份指标:
- backup_duration_seconds: 备份耗时
- backup_size_bytes: 备份大小
- backup_file_count: 文件数量
- backup_corruption_count: 损坏文件数
- backup_success: 备份是否成功（0/1）

恢复指标:
- restore_duration_seconds: 恢复耗时
- restore_file_count: 恢复文件数
- restore_success: 恢复是否成功（0/1）

验证指标:
- verify_duration_seconds: 验证耗时
- verify_valid_files: 有效文件数
- verify_corrupted_files: 损坏文件数
- verify_repaired_files: 已修复文件数
```

### 9.2 Prometheus 集成

```go
// metrics.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
)

var (
    backupDuration = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name: "zkbackup_backup_duration_seconds",
            Help: "Duration of backup operations",
        },
    )

    backupSize = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "zkbackup_backup_size_bytes",
            Help: "Size of the backup in bytes",
        },
    )

    backupSuccess = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "zkbackup_backup_success_total",
            Help: "Total number of successful backups",
        },
    )

    backupFailure = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "zkbackup_backup_failure_total",
            Help: "Total number of failed backups",
        },
    )
)

func init() {
    prometheus.MustRegister(backupDuration)
    prometheus.MustRegister(backupSize)
    prometheus.MustRegister(backupSuccess)
    prometheus.MustRegister(backupFailure)
}
```

### 9.3 告警规则

```yaml
# prometheus-alerts.yaml
groups:
- name: zkbackup
  rules:
  - alert: ZKBackupFailed
    expr: zkbackup_backup_failure_total > 0
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "ZooKeeper backup failed"
      description: "ZooKeeper backup has failed {{ $value }} times in the last 5 minutes"

  - alert: ZKBackupDurationHigh
    expr: zkbackup_backup_duration_seconds > 3600
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "ZooKeeper backup taking too long"
      description: "Backup duration is {{ $value }} seconds"

  - alert: ZKBackupCorruptionHigh
    expr: zkbackup_verify_corrupted_files > 5
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High number of corrupted files in backup"
      description: "Found {{ $value }} corrupted files"
```

---

## 10. 路线图

### 10.1 v1.0（MVP）

- ✅ 基础备份功能
- ✅ 基础恢复功能
- ✅ TxnLog 验证和修复
- ✅ 命令行接口
- ✅ 文档

### 10.2 v1.1

- 🔲 ZooKeeper 3.9+ Admin Server API 支持
- 🔲 增量备份（仅备份新增的 txnlog）
- 🔲 压缩支持（gzip, zstd）
- 🔲 并行备份
- 🔲 Prometheus metrics

### 10.3 v1.2

- 🔲 远程备份（直接备份到 S3/NFS）
- 🔲 备份加密
- 🔲 备份签名验证
- 🔲 Web UI（只读查看）

### 10.4 v2.0

- 🔲 跨数据中心备份
- 🔲 备份去重
- 🔲 自动恢复测试
- 🔲 AI 驱动的备份优化建议

---

## 11. 附录

### 11.1 ZooKeeper 版本兼容性

| ZooKeeper 版本 | TxnLog 格式版本 | Snapshot 格式版本 | 支持状态 |
|---------------|----------------|-----------------|---------|
| 3.4.x         | 2              | -               | ✅ 完全支持 |
| 3.5.x         | 2              | -               | ✅ 完全支持 |
| 3.6.x         | 2              | -               | ✅ 完全支持 |
| 3.7.x         | 2              | -               | ✅ 完全支持 |
| 3.8.x         | 2              | -               | ✅ 完全支持 |
| 3.9.x         | 2              | -               | ✅ 完全支持 + API |

### 11.2 性能基准

```
测试环境:
- CPU: Intel Xeon E5-2680 v4 @ 2.40GHz (8核)
- 内存: 64 GB
- 磁盘: SSD RAID 10
- ZooKeeper 数据量: 50 GB (500万 znode)

性能结果:
- 备份速度: 125 MB/s
- 恢复速度: 150 MB/s
- 验证速度: 80 MB/s
- 内存占用: 45 MB（恒定）
- CPU 占用: 35%（备份期间）

备份时间:
- 50 GB: ~7 分钟
- 100 GB: ~14 分钟
- 500 GB: ~70 分钟
```

### 11.3 FAQ

**Q: zkbackup 和现有的 qfb 工具有什么区别？**

A:
- `qfb`: 应用层备份工具，包含存储挂载、调度、集群管理等功能
- `zkbackup`: 专注于 ZooKeeper 数据备份/恢复，不涉及应用层逻辑
- 两者可以配合使用：qfb 负责调度和存储，zkbackup 负责实际备份

**Q: 为什么需要验证和修复 txnlog？**

A: 因为备份时 txnlog 可能正在写入，导致：
- 文件不完整（缺少部分事务）
- Checksum 错误
- 如果不验证，恢复时会失败

**Q: 支持在线备份吗？**

A: 是的，zkbackup 支持在线备份（ZooKeeper 无需停止）。但建议：
- 在低峰期备份
- 监控备份对性能的影响

**Q: 可以备份单个 znode 吗？**

A: 不支持。zkbackup 只做全量备份。如需备份单个 znode，请使用 ZooKeeper 客户端。

**Q: 如何验证备份是否可用？**

A:
```bash
# 方式 1: 使用 verify 命令
zkbackup verify --backup-dir /path/to/backup

# 方式 2: 在测试环境恢复并启动 ZooKeeper
zkbackup restore --backup-dir /path/to/backup --dry-run
```

---

## 12. 参考资料

- [ZooKeeper Administrator's Guide](https://zookeeper.apache.org/doc/current/zookeeperAdmin.html)
- [ZooKeeper Internals](https://zookeeper.apache.org/doc/current/zookeeperInternals.html)
- [ZooKeeper File TxnLog Format](https://github.com/apache/zookeeper/blob/master/zookeeper-server/src/main/java/org/apache/zookeeper/server/persistence/FileTxnLog.java)
- [ZooKeeper Snapshot Format](https://github.com/apache/zookeeper/blob/master/zookeeper-server/src/main/java/org/apache/zookeeper/server/persistence/FileSnap.java)

---

**文档版本**: 1.0
**最后更新**: 2025-01-15
**维护者**: ZooKeeper Backup Team
