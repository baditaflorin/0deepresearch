---
title: 'Hetzner Storage for PostgreSQL: Volumes, Storage Box, Object Storage and Storage Share'
date: 2026-08-05T14:44:00
draft: false
description: |-
  Executive conclusion

  For a live PostgreSQL data directory, the four Hetzner products are not interchangeable storage tiers.

  The practical ranking is:

  1. Local NVMe storage on the Cloud Server or a dedicated server — preferred for PostgreSQL performance and explicitly recommended by Hetzner.
  2. Hetzner Cloud Volume — technically valid block storage and the only one of the four products that behaves like a normal disk, but slower and substantially more expensive than bulk storage.
  3. Storage Box — excellent low-cost backup and archive storage, but a poor and unsupported architectural choice for a running PostgreSQL cluster.
  4. Object Storage — excellent for PostgreSQL backup repositories and WAL archives, but impossible to use directly as `PGDATA`.
  5. Storage Share — managed Nextcloud for human file collaboration; it should not be part of a PostgreSQL storage design.
---

# Hetzner Storage for PostgreSQL: Volumes, Storage Box, Object Storage and Storage Share

## Executive conclusion

For a live PostgreSQL data directory, the four Hetzner products are **not interchangeable storage tiers**.

The practical ranking is:

1. **Local NVMe storage on the Cloud Server or a dedicated server** — preferred for PostgreSQL performance and explicitly recommended by Hetzner.
2. **Hetzner Cloud Volume** — technically valid block storage and the only one of the four products that behaves like a normal disk, but slower and substantially more expensive than bulk storage.
3. **Storage Box** — excellent low-cost backup and archive storage, but a poor and unsupported architectural choice for a running PostgreSQL cluster.
4. **Object Storage** — excellent for PostgreSQL backup repositories and WAL archives, but impossible to use directly as `PGDATA`.
5. **Storage Share** — managed Nextcloud for human file collaboration; it should not be part of a PostgreSQL storage design.

Hetzner’s own general storage-selection documentation is unusually explicit: it marks **all four network-storage products as “not recommended” for storing a database**, citing latency and inadequate filesystem and caching guarantees, and recommends the local storage of a Cloud Server or dedicated server instead. Volumes are still materially safer than Storage Box because they expose a real block device with normal filesystem semantics, but Hetzner’s conservative guidance is that local disk remains the correct primary database storage. citeturn1view2

The most sensible architecture for the use case described is therefore:

> **PostgreSQL on local NVMe, continuous physical backups and WAL archiving to Hetzner Object Storage, and optionally a second independent backup copy in Storage Box.**

A Cloud Volume becomes appropriate when the database must be larger than the server’s included disk, or when detach-and-reattach portability is more important than maximum database performance. A Storage Box should be treated as a **backup destination**, not as a €3.20 substitute for a €57.20 database disk.

## What the Hetzner products actually provide

The apparent price anomaly comes from comparing products that sell different things. Storage capacity is only one dimension; access semantics, media type, latency, durability behaviour, IOPS, attachment model and intended workload are more important for PostgreSQL.

| Product | Underlying model | Access model | Media and redundancy | Current entry price excluding VAT | Appropriate PostgreSQL role |
|---|---|---|---|---:|---|
| Cloud Volume | Network-attached block storage | Linux block device with ext4, XFS or another filesystem | SSD; every block stored on three physical servers | Approximately €0.0572/GB-month, or €57.20 for 1,000 GB | Possible live `PGDATA`; not necessarily optimal |
| Storage Box BX11 | Shared network file storage / backup service | SMB/CIFS, SFTP, SCP, FTP, rsync over SSH, Borg, Restic, Rclone and WebDAV | HDD-based shared backend with RAID | €3.20/month for a 1 TB quota | Backups, WAL archives, dumps and cold files |
| Object Storage | Distributed object store | S3-compatible HTTP API | HDD-based shared backend with erasure coding | €6.49/month base price including 1 TB storage and 1 TB egress | Physical backup repository, WAL archive, logical dumps |
| Storage Share NX11 | Managed Nextcloud | HTTPS, Nextcloud clients and WebDAV | HDD/RAID-based managed service with automatic service backups | €4.29/month for 1 TB | Human file sharing; not database infrastructure |
| Local server storage | Disk physically local to the Cloud or dedicated server host | Native block device and filesystem | Usually local SSD/NVMe; characteristics depend on server type | Included in the server price | Preferred live `PGDATA` |

Hetzner describes Cloud Volumes as SSD-based, network-attached block storage. They can be expanded from 10 GB to 10 TB, attached to one Cloud Server at a time, and formatted and mounted like an additional hard drive. Each block is stored on three physical servers. Current documented limits are up to 5,000 sustained read/write IOPS, 7,500 burst IOPS, 200 MB/s sustained throughput and 300 MB/s burst throughput. citeturn1view0turn1view1

Storage Box, by contrast, is marketed as an inexpensive online backup solution. The 1 TB BX11 plan costs €3.20 per month excluding VAT and provides a storage quota, ten simultaneous connections, ten snapshot slots and access through file-transfer and network-file protocols. Hetzner’s product comparison identifies the Storage Box backend as HDD-based shared storage protected by RAID. citeturn0search10turn1view2turn1view3

The reference to “LSync” is probably a reference to **rsync**. Storage Box supports rsync over its restricted SSH service on port 23. It does not run arbitrary applications, does not provide a normal shell, and cannot run rsync jobs autonomously on the Storage Box itself; another machine must initiate the transfer. citeturn4search0

Object Storage is not a network filesystem. It stores complete objects in buckets through an S3-compatible API. Objects are independently addressed by keys, and directory structures displayed by clients are only visual representations of common key prefixes. Hetzner currently allows up to 100 TB and 50 million objects per bucket, with service limits including 750 requests per second per bucket and a maximum individual object size of 5 TB. citeturn7search1turn7search3

Storage Share is a hosted Nextcloud service intended for document storage, sharing, calendars, contacts and collaborative access. It provides HTTPS and WebDAV rather than a database-grade block device. Hetzner marks it as unsuitable for storing or backing up a live database. citeturn0search3turn1view2

## Why a Volume costs almost eighteen times more

At current prices, 1,000 GB of Cloud Volume capacity costs approximately:

\[
1{,}000 \times €0.0572 = €57.20/\text{month}
\]

A 1 TB Storage Box costs €3.20 per month. The nominal capacity-price ratio is therefore approximately:

\[
€57.20 / €3.20 = 17.9
\]

That difference does not mean that Storage Box is an unusually cheap equivalent disk. It means that Hetzner is selling two different service contracts.

### Cloud Volume economics

A Volume must serve small, latency-sensitive, random reads and writes while looking to the operating system like a disk. It must support filesystem metadata updates, write barriers, cache flushes, concurrent database files, file truncation, page overwrites, WAL synchronisation and recovery after a server crash.

Hetzner also stores every Volume block on three physical servers. The provisioned capacity must be available regardless of whether the filesystem has filled it, and the service offers documented IOPS and throughput ceilings. It is consequently priced as **online SSD block storage**, not as archival capacity. citeturn1view0turn1view1

A 1 TB Volume is also not automatically faster than a smaller Volume. Hetzner publishes a common service-level performance ceiling rather than an IOPS-per-GB scaling model. Buying more gigabytes primarily buys capacity, not proportionally higher throughput. citeturn1view1

### Storage Box economics

Storage Box sells a quota on a shared HDD-based file service designed primarily for backups and bulk transfer. A backup workload normally consists of relatively large sequential files, limited concurrency and periods of inactivity. The provider can aggregate many customers efficiently because most customers are not performing thousands of latency-sensitive synchronous writes every second. citeturn1view2turn1view3

The 1 TB quota does not imply:

- a dedicated 1 TB drive;
- dedicated IOPS;
- SSD latency;
- a block device;
- PostgreSQL-compatible durability guarantees;
- unrestricted filesystem operations;
- exclusive bandwidth.

Storage Box does provide useful data-protection functionality, including manual and automated snapshots. Those snapshots are valuable for backup repositories, but they do not transform the service into database block storage. citeturn1view3

### Object Storage economics

Object Storage is cheaper than block storage because complete objects are uploaded and retrieved through an API. Applications do not issue arbitrary 8 KB overwrites to disk sectors. Updating an object generally creates or replaces an object rather than modifying database pages in place.

Hetzner’s current base price is €6.49 per month excluding VAT and includes up to 744 TB-hours—equivalent to 1 TB stored for a full 31-day month—and 1 TB of outgoing traffic. Ingress, S3 API calls and internal traffic in Hetzner’s `eu-central` network zone are free. Excess storage and egress are billed separately. citeturn0search0turn0search2turn0search9

This is an excellent economic model for compressed PostgreSQL backups because backup files are immutable or append-oriented and do not require normal filesystem semantics.

## PostgreSQL’s actual storage requirements

PostgreSQL does far more than read and write large files. Its correctness depends on the storage stack preserving specific semantics during crashes, network failures and sudden host termination.

At commit time, PostgreSQL normally writes WAL records and asks the operating system to flush them to durable storage. During recovery, it assumes that successfully acknowledged writes behaved according to the filesystem and storage guarantees. PostgreSQL’s data directory also contains relation files, transaction-status files, control files and temporary state that must remain mutually consistent. PostgreSQL’s WAL exists specifically to restore data files to a consistent state after a crash. citeturn3view2

The relevant characteristics are therefore:

| Characteristic | Why PostgreSQL cares |
|---|---|
| Low write latency | A durable commit may wait for WAL synchronisation |
| Reliable `fsync` or equivalent | PostgreSQL must know that committed WAL reached durable media |
| Correct write ordering | WAL must become durable before dependent data pages are considered safe |
| POSIX-like filesystem behaviour | PostgreSQL expects normal file creation, rename, permissions, truncation and error semantics |
| Small random I/O | Tables and indexes are accessed in database pages rather than only as large sequential files |
| Predictable error behaviour | An interrupted operation must produce an error rather than silent partial success |
| Stable availability | Loss of the storage mount can stall or crash the database |
| Bounded tail latency | Rare multi-second I/O delays can freeze commits and accumulate connections |

### Network filesystems are conditionally possible, not generally equivalent

PostgreSQL’s documentation says that an NFS filesystem *can* host a data directory, but only under important assumptions. PostgreSQL expects NFS to behave like locally attached storage. It requires the client to use the `hard` mount option; with a hard mount, PostgreSQL processes can hang indefinitely during network problems. It also strongly recommends a synchronous NFS export so that client `fsync` calls actually reach permanent storage. Incorrect server-side export semantics can create corruption risks similar to disabling PostgreSQL `fsync`. citeturn3view0

That documentation does **not** mean that any remotely mounted drive is safe.

Storage Box exposes SMB/CIFS and WebDAV, not a customer-controlled NFS server. As a Storage Box customer, you do not administer the server-side filesystem, export configuration, write cache, RAID controller or durability path. You cannot independently guarantee that a successful CIFS flush has the exact end-to-end semantics PostgreSQL expects.

More importantly, Hetzner itself categorises Storage Box as not recommended for database storage. That direct provider statement should outweigh theoretical arguments that PostgreSQL can sometimes operate over carefully engineered network filesystems. citeturn1view2

### Why SMB-mounted Storage Box is particularly problematic

A PostgreSQL installation on a CIFS-mounted Storage Box could fail operationally even without immediate data corruption.

Every page miss, index update, relation extension, checkpoint write and WAL synchronisation traverses:

1. PostgreSQL;
2. the Linux virtual filesystem;
3. the CIFS client;
4. the VM’s network stack;
5. Hetzner’s network;
6. the Storage Box file server;
7. its shared HDD/RAID backend.

This adds network round trips to I/O operations that would normally take microseconds on local NVMe. The low price also implies that no dedicated random-I/O performance is being reserved.

The Storage Box’s limit of ten concurrent service connections is not directly equivalent to PostgreSQL’s internal I/O concurrency, because the kernel may multiplex operations over fewer SMB connections. Nevertheless, it demonstrates that this is a constrained shared file-transfer service rather than a high-concurrency database storage system. citeturn1view3

Sequential transfer tests are misleading here. Hetzner’s documentation includes an SFTP example showing roughly 79–111 MB/s for a single 100 MB file under one test condition, but a database performance bottleneck is usually synchronous latency and small random I/O, not peak large-file transfer speed. citeturn4search0

### Cloud Volume is technically suitable, but bounded

A Cloud Volume exposes a block device. Linux creates a normal filesystem on it, and PostgreSQL sees ordinary filesystem operations rather than SMB or WebDAV. This removes most of the semantic uncertainty associated with Storage Box.

Hetzner’s product page even describes Volumes as suitable for databases. However, its broader storage-selection document recommends local disk rather than any network storage for running databases. These two statements are not irreconcilable:

- **“Suitable”** means that the product can correctly host a database.
- **“Recommended”** addresses the best-performing and least ambiguous design.

For a small or moderate PostgreSQL workload, a Volume’s 5,000 sustained IOPS and 200 MB/s may be entirely adequate. For write-heavy OLTP, large analytical scans, aggressive autovacuum, index builds or checkpoint bursts, that ceiling may become material. citeturn1view0turn1view1turn1view2

A public 2024 fio result for a 1 TB Hetzner Volume reported approximately 3,500 read IOPS plus 3,500 write IOPS in a mixed 4 KB test and around 286 MB/s aggregate mixed throughput with 64 KB operations. This is broadly compatible with the published burst limits, but it is only one user’s point-in-time benchmark and should not be treated as a service guarantee. citeturn10search9

Local Hetzner NVMe can be much faster. A March 2026 independent Cloud Server benchmark measured about 40,900 4 KB random IOPS and gigabytes per second of sequential throughput on a VM’s local disk. That was a different workload and server, so it is not a controlled comparison, but it illustrates the order-of-magnitude advantage that local NVMe can have over capped network block storage. citeturn10search0

## Product-by-product PostgreSQL assessment

### Cloud Volume

**Verdict: acceptable for live PostgreSQL when its operational advantages are required, but benchmark it first.**

Advantages:

- It is a real block device and can use ext4 or XFS.
- Each block is replicated across three physical servers.
- It survives independently of a particular VM.
- It can be detached and reattached to another Cloud Server.
- Capacity can be increased without replacing the Volume.
- Its published performance is adequate for many small and medium databases.
- It separates database capacity from VM sizing. citeturn1view0turn1view1

Disadvantages:

- It introduces network latency into PostgreSQL’s critical write path.
- Sustained performance is capped at approximately 5,000 IOPS and 200 MB/s.
- It can be attached to only one server at a time.
- It cannot be shrunk after expansion.
- Hetzner does not provide native backups or snapshots for Volumes.
- Cloud Server backups and snapshots do not include attached Volumes.
- At 1 TB, the storage alone costs roughly €57.20 per month excluding VAT. citeturn1view0turn1view1

The absence of native Volume snapshots is a particularly important hidden difference from AWS EBS-style expectations. A Volume’s triple replication protects against underlying hardware failures; it does not protect against accidental deletion, dropped tables, ransomware, application bugs or a corrupt database state replicated correctly to all three copies.

### Storage Box

**Verdict: do not place live `PGDATA`, tablespaces or `pg_wal` on it. Use it for backups.**

Good uses include:

- `pg_dump` and `pg_dumpall` output;
- compressed `pg_basebackup` archives;
- pgBackRest repositories over SFTP;
- Borg or Restic backup repositories;
- exported CSV or Parquet data;
- old WAL archives under a controlled retention policy;
- a second backup copy separate from Object Storage.

Storage Box natively supports BorgBackup, Restic, Rclone, rsync and SFTP, and its snapshots provide additional protection against accidental modification of a backup repository. citeturn1view3turn4search0

Poor uses include:

- PostgreSQL’s main data directory;
- `pg_wal`;
- active tablespaces;
- temporary tablespaces;
- SQLite or other write-intensive database files;
- virtual-machine disk images under active random-write workloads.

Even placing only cold PostgreSQL tablespaces on Storage Box is unattractive. PostgreSQL maintenance operations, VACUUM, index scans and checkpoint activity can still access or modify apparently cold data, while a Storage Box outage would affect the whole database instance.

### Object Storage

**Verdict: the strongest Hetzner option for PostgreSQL backups and WAL retention.**

Object Storage cannot hold `PGDATA` directly because PostgreSQL needs files that can be opened, modified, truncated and synchronised in place. S3 objects do not provide those operations. Mounting a bucket through an S3 FUSE driver creates a compatibility façade, not a database-grade filesystem.

It is, however, well matched to physical backup tools. pgBackRest officially supports repositories on S3-compatible object stores, including parallel backup and restore, full, differential and incremental backups, compression, client-side repository encryption and WAL archiving. citeturn7search0turn7search7

Hetzner Object Storage also supports:

- object versioning;
- lifecycle rules;
- Object Lock legal holds;
- governance and compliance retention modes.

Object Lock must be enabled when the bucket is created; it cannot be added later to an existing ordinary bucket. These capabilities can make a PostgreSQL backup repository much more resistant to accidental deletion or compromised database credentials. citeturn12search0turn12search2turn12search4

Important limitations include:

- Hetzner Object Storage does not enable at-rest encryption by default.
- Its supported server-side encryption method is SSE-C, using a customer-provided key.
- Losing an SSE-C key makes the encrypted objects unrecoverable.
- Alternatively, tools such as pgBackRest can encrypt backup content client-side. citeturn12search7turn12search10turn7search7

For most self-managed PostgreSQL installations, client-side pgBackRest encryption is operationally simpler than manually managing SSE-C headers for every object.

### Storage Share

**Verdict: exclude it from the database architecture.**

Storage Share is a managed Nextcloud environment. Its strengths are browser access, desktop synchronisation, mobile applications, user management, file sharing and collaborative applications. Its automatic backups protect the Nextcloud service, not an external PostgreSQL cluster. citeturn0search3

Uploading `pg_dump` files manually to Storage Share is technically possible, but Object Storage or Storage Box is better for automation, access control, lifecycle retention and infrastructure tooling.

## Recommended architectures

### Cost-efficient single-server PostgreSQL

This is the default recommendation when the database fits on the Cloud Server’s included local disk.

```text
Hetzner Cloud Server
├── Local NVMe/SSD
│   ├── PostgreSQL PGDATA
│   └── PostgreSQL WAL
│
├── pgBackRest
│   ├── Weekly full backup
│   ├── Daily differential or incremental backup
│   └── Continuous WAL archiving
│
└── Hetzner Object Storage
    ├── Encrypted backup repository
    ├── Versioning
    ├── Object Lock or restricted credentials
    └── Lifecycle/retention policy
```

Advantages:

- Highest likely I/O performance.
- No extra Volume charge.
- Simplest PostgreSQL storage path.
- Object Storage provides inexpensive, independent backups.
- Point-in-time recovery can restore the database to a chosen time between base backups. PostgreSQL’s continuous-archiving model combines a base backup with an uninterrupted sequence of WAL files. citeturn3view2

Disadvantages:

- The data disk remains tied to that Cloud Server.
- Replacing the server requires restoration or replication.
- Capacity is limited by the selected server’s local disk.
- A host or local-disk failure requires failover or restore.

The database must never rely solely on Hetzner’s Cloud Server backup feature. Database-aware physical backups plus WAL archiving provide much more precise recovery than a periodic VM image.

### PostgreSQL using a Cloud Volume

Use this architecture when the database exceeds local disk capacity, when independent Volume lifecycle is valuable, or when restoring a one-terabyte database from Object Storage would violate the required recovery-time objective.

```text
Hetzner Cloud Server
├── Local disk
│   ├── Operating system
│   ├── PostgreSQL binaries
│   └── Optional temporary files
│
├── Cloud Volume
│   └── /var/lib/postgresql/<version>/main
│
└── Object Storage
    ├── pgBackRest base backups
    └── Continuous WAL archive
```

The Volume should be manually formatted and mounted. A dedicated directory beneath the mount point should be owned by the `postgres` user; PostgreSQL recommends not using the root of a secondary filesystem directly as the data directory. This makes offline-volume failures cleaner and avoids permission complications during upgrades. citeturn3view0

A reasonable layout is:

```text
/mnt/pg-volume/
└── postgres/
    └── 18/
        └── main/
```

Use a filesystem such as ext4 or XFS with normal durability settings. Do not disable `fsync`, `full_page_writes` or write barriers to compensate for Volume latency. Those changes exchange measurable performance for weakened crash safety.

Advantages:

- More capacity than most Cloud Server root disks.
- Storage can outlive and move between servers.
- Triple-replicated backend.
- Easier server replacement than restoring a very large backup.

Disadvantages:

- Approximately €57.20/month for 1,000 GB before the VM and backup costs.
- Lower and more bounded I/O than local NVMe.
- No Hetzner Volume snapshots.
- Still requires independent PostgreSQL backups.
- Single attachment prevents shared-disk active/active clustering.

### Dedicated server with mirrored NVMe

Once a PostgreSQL database needs approximately one terabyte of active data, substantial RAM and sustained I/O, a dedicated server can be economically and technically stronger than a Cloud VM plus a 1 TB Volume.

For example, Hetzner’s current EX63 base configuration includes 64 GB RAM and two 1 TB NVMe devices, while the AX102 includes 128 GB RAM and two 1.92 TB datacentre NVMe devices. The EX63 currently starts at €74 per month excluding VAT plus setup, although final prices depend on the selected configuration. citeturn8search7turn8search8

With software RAID1:

```text
NVMe device A ─┐
               ├── RAID1 ── filesystem ── PostgreSQL
NVMe device B ─┘
```

Advantages:

- Much higher IOPS and lower latency than a network Volume.
- Mirroring protects against one-device failure.
- Large RAM configurations improve PostgreSQL cache hit rates.
- Predictable CPU resources and no VM CPU-steal concerns.
- The complete machine may cost only modestly more than a large Cloud instance plus €57.20 of block storage.

Disadvantages:

- A single server remains a single host and motherboard failure domain.
- RAID is not a backup.
- Server replacement and automation are less cloud-like.
- Some plans have setup fees.
- Vertical scaling usually requires hardware or server migration.
- High availability requires a second server, replication and failover orchestration.

For a one-terabyte production database, this option deserves explicit cost modelling rather than assuming Cloud is automatically cheaper.

### High-availability PostgreSQL

A resilient design separates three different goals:

- **Storage redundancy** protects against disk hardware failure.
- **Streaming replication** protects availability when the primary server fails.
- **Backups and WAL archives** protect against logical corruption, deletion and operator error.

Triple replication on a Cloud Volume or RAID1 on a dedicated server addresses only the first goal. Neither provides point-in-time recovery from `DROP TABLE`, bad application writes or compromised credentials.

A robust topology is:

```text
Primary PostgreSQL
├── Local NVMe or Cloud Volume
└── Streams WAL to standby

Standby PostgreSQL
├── Different server/host
└── Independent local NVMe or Volume

Object Storage
├── Base backups
├── Continuous WAL archive
├── Versioning/Object Lock
└── Credentials separate from ordinary database users

Optional Storage Box
└── Periodic second backup copy
```

PostgreSQL can continuously stream WAL to a standby, while archived WAL plus base backups support point-in-time recovery. These solve different failure scenarios and are best deployed together for important data. citeturn2search9turn3view2

## Operational design and validation

### Preferred backup implementation

For a serious production database, use a physical backup manager rather than synchronising a live PostgreSQL directory with rsync.

PostgreSQL warns that directly copying a live data directory does not produce a usable backup unless the database is shut down, a consistent filesystem snapshot is taken, or the backup is coordinated with WAL archiving. An ordinary rsync of changing PostgreSQL files can capture mutually inconsistent files. citeturn3view3

A preferred stack is:

```text
PostgreSQL
    ↓
pgBackRest
    ↓
Hetzner Object Storage
```

A representative policy might be:

- continuous WAL archiving;
- one full backup each week;
- one differential backup each day;
- optional block-incremental backups between them;
- two to four retained full backup generations;
- client-side repository encryption;
- Object Lock or a dedicated restricted backup credential;
- an automated restore test at least monthly.

The exact retention period should be driven by recovery-point requirements and WAL generation rate, not merely database size. A one-terabyte database that changes 2% daily has a completely different backup profile from a one-terabyte database rewriting hundreds of gigabytes per day.

Storage Box can serve as a pgBackRest SFTP repository or Borg/Restic target, but restores will generally be slower and its file-service model is less scalable than an S3 repository. pgBackRest’s own guidance notes that SFTP repositories are relatively slow and benefit from parallel transfer processes. citeturn7search0

### Benchmark before migration

Do not benchmark only sequential bandwidth. A valid PostgreSQL storage evaluation should test:

1. synchronous write latency;
2. 8 KB random reads and writes;
3. mixed random I/O;
4. sustained performance after burst credits or caches are exhausted;
5. checkpoint-like sequential writes;
6. behaviour during network interruption;
7. full backup and restore speed;
8. real `pgbench` or application-query throughput.

For the filesystem that will contain `pg_wal`, run PostgreSQL’s `pg_test_fsync`. The tool reports average synchronisation latency for the available WAL synchronisation methods and should be executed on the same filesystem intended for WAL. citeturn3view1

A basic storage test can include:

```bash
# Run in a disposable test directory on the target filesystem.
# Do not run against a production database directory.

fio \
  --name=pg-random \
  --filename=/mnt/test/pg-fio.dat \
  --size=20G \
  --direct=1 \
  --ioengine=libaio \
  --bs=8k \
  --rw=randrw \
  --rwmixread=70 \
  --iodepth=1 \
  --numjobs=4 \
  --runtime=300 \
  --time_based \
  --group_reporting
```

Queue depth one matters because PostgreSQL commit latency is often sensitive to individual durable writes, not only maximum throughput at queue depth 64. Repeat with different queue depths and inspect latency percentiles, especially the 99th and 99.9th percentiles.

Then test PostgreSQL itself:

```bash
createdb storage_benchmark
pgbench -i -s 1000 storage_benchmark

pgbench \
  -c 32 \
  -j 8 \
  -T 900 \
  -P 10 \
  storage_benchmark
```

A serious comparison should run identical server sizes, PostgreSQL configurations, datasets and client loads on:

- local server disk;
- a Cloud Volume;
- optionally a dedicated server.

Storage Box does not merit a production `PGDATA` benchmark unless the objective is specifically to quantify why it is unsuitable.

### Monitoring thresholds

For a Volume-backed database, monitor at least:

- PostgreSQL transaction latency;
- `pg_stat_database` block read/write time;
- checkpoints requested versus timed;
- checkpoint write and sync duration;
- WAL generation rate;
- Volume utilisation and available space;
- Linux I/O latency from `iostat -x`;
- stalled processes and filesystem errors;
- pgBackRest archive queue and failures;
- object-storage backup age;
- restore-test result and duration.

A WAL archive failure is operationally dangerous because PostgreSQL retains unarchived WAL. If the filesystem containing `pg_wal` fills, PostgreSQL will eventually perform a PANIC shutdown and remain unavailable until space is recovered. citeturn3view2

### Security and deletion resistance

Use separate credentials for:

- the PostgreSQL application;
- PostgreSQL administration;
- backup uploads;
- backup deletion and retention management.

The backup uploader should ideally be able to create new backup objects but not shorten Object Lock retention or delete existing generations.

For Hetzner Object Storage, create the bucket with Object Lock enabled from the start when immutability is required. Compliance mode prevents early deletion even by privileged users; governance mode allows specially authorised users to bypass retention. citeturn12search3turn12search4

Do not count replication, RAID or Storage Box snapshots as the sole backup. All can preserve an already-corrupt or maliciously modified database state.

## Final recommendation

The strongest practical recommendation is:

### For a database that fits on the server’s local disk

Run PostgreSQL on local NVMe and use pgBackRest with continuous WAL archiving to Hetzner Object Storage. Enable repository encryption, versioning and either Object Lock or tightly restricted backup credentials.

### For a database requiring more Cloud storage or portable persistence

Use a Cloud Volume for `PGDATA`, but accept the approximately €57.20 per 1,000 GB monthly cost, the 5,000 sustained IOPS ceiling and the lack of native Volume snapshots. Keep PostgreSQL backups in Object Storage because triple replication is not a backup.

### For sustained one-terabyte production workloads

Price a dedicated server with mirrored NVMe before buying a large Cloud VM plus a 1 TB Volume. Dedicated local NVMe will normally offer materially better database latency and throughput, and the total platform cost may be competitive once the €57.20 Volume charge and a suitably sized Cloud Server are included.

### For Storage Box

Use the €3.20 BX11 as inexpensive backup capacity, not as PostgreSQL’s live disk. Suitable contents include encrypted pgBackRest archives, `pg_dump` output, exported datasets and a second copy of backups already stored elsewhere.

### For Object Storage

Use it as the primary PostgreSQL backup repository. At €6.49 per month for the included 1 TB quota and 1 TB egress, it is the most attractive combination of automation, scalability, retention controls and restore tooling among Hetzner’s low-cost storage products. citeturn0search2turn7search0turn12search0

### Bottom line

- **Cloud Volume and Storage Box are not substitute products:** one is SSD block storage; the other is shared HDD backup storage.
- **Do not run PostgreSQL directly on Storage Box, Object Storage or Storage Share.**
- **Prefer local NVMe for live PostgreSQL.**
- **Use a Cloud Volume only where capacity independence or detachability justifies its latency and cost.**
- **Use Object Storage for pgBackRest backups and continuous WAL archiving.**
- **Use Storage Box as an inexpensive second backup destination.**
- **For approximately one terabyte of active PostgreSQL data, compare a dedicated mirrored-NVMe server against Cloud Server plus Volume before committing.**
