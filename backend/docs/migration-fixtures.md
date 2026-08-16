# Migration fixture data policy

Migration fixtures are documentation and test-only data. They must never
contain a real person's name, address, phone number, login, credential, or
tenant entitlement.

- Email values must use the reserved `example.test` domain.
- The only allowed Romanian-looking phone fixture range is `+401xxxxxxxx`.
  It is deliberately non-routable and is used only where schema validation
  needs a phone-shaped value.
- Privileged users and cross-tenant memberships are provisioned after deploy by
  the audited administration workflow. Migrations must not create a real or
  operational administrator identity.

`TestMigrationFixturesAreSynthetic` scans every embedded migration and runs in
the standard backend CI test job.
