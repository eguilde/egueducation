# MinIO WORM precondition

The application only claims storage-side WORM when the configured archive
bucket proves the first two prerequisites through the S3 API at startup and
each immutable upload supplies the remaining controls:

- bucket versioning is `Enabled`;
- Object Lock is `Enabled`;
- every stored version has its policy-derived `retention_until` sent to MinIO;
- every stored version uses `COMPLIANCE` mode and records the returned MinIO
  version ID and ETag;
- legal hold is applied per object when the business record requires it.

The application does not enable or repair these controls on an existing
bucket. An operator must first approve a retention schedule for each archive
series, configure storage consistently with that schedule and independently
retain MinIO audit logs, replication/backup evidence and the bucket-policy
export. Only then may GitOps set
`ARCHIVE_STORAGE_REQUIRE_OBJECT_LOCK=true`.

`PutImmutableObject` is the fail-closed storage boundary for these writes. The
current HTTP upload and Registratură outbox contracts do not yet carry a legally
approved per-document retention deadline, so they still use the legacy write
path. Therefore the application deliberately does not enable the flag in the
base production manifest and does not invent a one-size-fits-all duration.
Until the archive classification/retention schedule is bound to every producer,
all producers call the immutable boundary, and the live bucket passes the
startup proof, storage WORM remains a rollout blocker, not a declared
capability.

Object Lock has to be enabled when a bucket is created. If the existing
`earhive` bucket cannot satisfy the startup check, migrate objects into a new
Object-Lock-enabled bucket and verify every source SHA-256 before changing the
application secret. Do not disable the startup check to make rollout pass.

The database migration `0126_earchiva_worm_and_signature_validation_reports.sql`
also makes archive-version bitstream provenance append-only and binds signed
artifact evidence to the exact `(institution, document, version)` tuple. This
database control complements Object Lock; it does not replace it.
